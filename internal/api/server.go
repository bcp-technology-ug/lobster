// Package api wires the gRPC server, gRPC-Gateway HTTP mux, and all service
// implementations together. Call Build to get a ready-to-serve Server.
package api

import (
	"context"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	adminv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/admin"
	integrationsv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/integrations"
	planv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/plan"
	runv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/run"
	stackv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/stack"

	"github.com/bcp-technology-ug/lobster/internal/api/adminsvc"
	"github.com/bcp-technology-ug/lobster/internal/api/integrationsvc"
	"github.com/bcp-technology-ug/lobster/internal/api/middleware"
	"github.com/bcp-technology-ug/lobster/internal/api/plansvc"
	"github.com/bcp-technology-ug/lobster/internal/api/runsvc"
	"github.com/bcp-technology-ug/lobster/internal/api/stacksvc"
	"github.com/bcp-technology-ug/lobster/internal/store"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Config groups all settings needed to build the server.
type Config struct {
	// Auth controls JWKS token validation.
	Auth middleware.AuthConfig
	// Logger is the base structured logger injected into every RPC context.
	// When nil, a no-op logger is used (no output, no panics).
	Logger *zap.Logger
	// Version is the daemon binary version string reported in health responses.
	Version string
	// WorkspaceID and ActiveProfile are surfaced in GetConfigSummary.
	WorkspaceID   string
	ActiveProfile string
	// ConfigSummaryFunc returns the sanitized effective config on demand.
	// When nil, GetConfigSummary returns codes.Unavailable.
	ConfigSummaryFunc func() *adminv1.ConfigSummary
}

// Services bundles optional capability providers that back the gRPC handlers.
// Each field may be nil; the corresponding RPCs return Unimplemented until wired.
type Services struct {
	Runner       runsvc.Runner
	Planner      plansvc.Planner
	Orchestrator stacksvc.Orchestrator
	Validator    integrationsvc.AdapterValidator
	// Notifier is called when an adapter's enabled state changes so in-process
	// consumers (e.g. the runner's adapter registry) stay in sync with the DB.
	Notifier integrationsvc.StateNotifier
}

// Server holds the running gRPC server and HTTP gateway mux.
type Server struct {
	GRPCServer *grpc.Server
	GatewayMux *runtime.ServeMux
	RunSvc     *runsvc.Service
	PlanSvc    *plansvc.Service
	StackSvc   *stacksvc.Service
	AdminSvc   *adminsvc.Service
	IntSvc     *integrationsvc.Service
	HealthSrv  *health.Server
}

// Build constructs a fully wired Server without starting any listeners.
func Build(st *store.Store, cfg Config, svc Services) (*Server, error) {
	auth, err := middleware.NewJWKSAuth(cfg.Auth)
	if err != nil {
		return nil, err
	}

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	// Interceptor ordering is load-bearing:
	//  1. Logging — injects a request-scoped logger into ctx first so that all
	//     subsequent interceptors and handlers can emit correlated log entries.
	//  2. Auth   — rejects unauthenticated requests before any business logic
	//     runs. Must come before validation so we never leak validation details
	//     to unauthenticated callers.
	//  3. ProtoValidate — validates request fields. Runs after auth so invalid
	//     requests from authenticated callers produce clear error messages.
	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.Logging(logger),
			auth.UnaryInterceptor(),
			middleware.ProtoValidate(),
		),
		grpc.ChainStreamInterceptor(
			middleware.LoggingStream(logger),
			auth.StreamInterceptor(),
			middleware.ProtoValidateStream(),
		),
	)

	adminCfgFunc := cfg.ConfigSummaryFunc
	if adminCfgFunc == nil {
		adminCfgFunc = func() *adminv1.ConfigSummary { return nil }
	}

	runS := runsvc.New(st, svc.Runner)
	planS := plansvc.New(st, svc.Planner)
	stackS := stacksvc.New(st, svc.Orchestrator)
	adminS := adminsvc.New(st, cfg.Version, adminCfgFunc)
	intS := integrationsvc.New(st, svc.Validator).WithNotifier(svc.Notifier)

	runv1.RegisterRunServiceServer(grpcSrv, runS)
	planv1.RegisterPlanServiceServer(grpcSrv, planS)
	stackv1.RegisterStackServiceServer(grpcSrv, stackS)
	adminv1.RegisterAdminServiceServer(grpcSrv, adminS)
	integrationsv1.RegisterIntegrationServiceServer(grpcSrv, intS)

	// gRPC standard health check protocol (grpc.health.v1.Health).
	// Enables: grpcurl health checks, k8s liveness probes, load balancers.
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("lobster.v1.run.RunService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("lobster.v1.plan.PlanService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("lobster.v1.stack.StackService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("lobster.v1.admin.AdminService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("lobster.v1.integrations.IntegrationService", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcSrv)

	return &Server{
		GRPCServer: grpcSrv,
		RunSvc:     runS,
		PlanSvc:    planS,
		StackSvc:   stackS,
		AdminSvc:   adminS,
		IntSvc:     intS,
		HealthSrv:  healthSrv,
	}, nil
}

// GatewayMuxFor builds an HTTP/JSON gateway mux that proxies requests to the
// gRPC server already bound on grpcAddr (e.g. "localhost:9443").
// TLS is intentionally skipped for in-process gateway connections.
func GatewayMuxFor(ctx context.Context, grpcAddr string) (*runtime.ServeMux, error) {
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	if err := runv1.RegisterRunServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		return nil, err
	}
	if err := planv1.RegisterPlanServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		return nil, err
	}
	if err := stackv1.RegisterStackServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		return nil, err
	}
	if err := adminv1.RegisterAdminServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		return nil, err
	}
	if err := integrationsv1.RegisterIntegrationServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		return nil, err
	}
	return mux, nil
}

// ServeGRPC starts serving gRPC requests on lis. It blocks until the server
// stops or the context is cancelled. On context cancellation it performs a
// graceful stop.
func ServeGRPC(ctx context.Context, srv *grpc.Server, lis net.Listener) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(lis)
	}()
	select {
	case <-ctx.Done():
		srv.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}

// ServeHTTP starts serving the HTTP gateway on httpListen. It blocks until the
// server stops or ctx is cancelled.
func ServeHTTP(ctx context.Context, mux http.Handler, httpListen string) error {
	httpSrv := &http.Server{
		Addr:              httpListen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- httpSrv.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

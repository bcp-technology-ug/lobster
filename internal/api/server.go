// Package api wires the gRPC server, gRPC-Gateway HTTP mux, and all service
// implementations together. Call Build to get a ready-to-serve Server.
package api

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	adminv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/admin"
	integrationsv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/integrations"
	planv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/plan"
	runv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/run"
	stackv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/stack"

	"github.com/bcp-technology/lobster/internal/api/adminsvc"
	"github.com/bcp-technology/lobster/internal/api/integrationsvc"
	"github.com/bcp-technology/lobster/internal/api/middleware"
	"github.com/bcp-technology/lobster/internal/api/plansvc"
	"github.com/bcp-technology/lobster/internal/api/runsvc"
	"github.com/bcp-technology/lobster/internal/api/stacksvc"
	"github.com/bcp-technology/lobster/internal/store"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

// Config groups all settings needed to build the server.
type Config struct {
	// Auth controls JWKS token validation.
	Auth middleware.AuthConfig
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
}

// Build constructs a fully wired Server without starting any listeners.
func Build(st *store.Store, cfg Config, svc Services) (*Server, error) {
	auth, err := middleware.NewJWKSAuth(cfg.Auth)
	if err != nil {
		return nil, err
	}

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			auth.UnaryInterceptor(),
			middleware.ProtoValidate(),
		),
		grpc.ChainStreamInterceptor(
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
	intS := integrationsvc.New(st, svc.Validator)

	runv1.RegisterRunServiceServer(grpcSrv, runS)
	planv1.RegisterPlanServiceServer(grpcSrv, planS)
	stackv1.RegisterStackServiceServer(grpcSrv, stackS)
	adminv1.RegisterAdminServiceServer(grpcSrv, adminS)
	integrationsv1.RegisterIntegrationServiceServer(grpcSrv, intS)

	reflection.Register(grpcSrv)

	return &Server{
		GRPCServer: grpcSrv,
		RunSvc:     runS,
		PlanSvc:    planS,
		StackSvc:   stackS,
		AdminSvc:   adminS,
		IntSvc:     intS,
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

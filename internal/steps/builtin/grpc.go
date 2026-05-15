package builtin

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/bcp-technology-ug/lobster/internal/steps"
)

const srcGRPC = "builtin:grpc"

func registerGRPCSteps(r *steps.Registry) error {
	defs := []struct {
		pattern string
		handler steps.StepHandler
	}{
		{
			`the gRPC service at "([^"]+)" should be healthy`,
			stepGRPCServiceHealthy,
		},
		{
			`the gRPC service at "([^"]+)" serving "([^"]+)" should be healthy`,
			stepGRPCServiceNameHealthy,
		},
		{
			`I wait up to (\d+)s? for the gRPC service at "([^"]+)" to be healthy`,
			stepWaitForGRPCService,
		},
	}
	for _, d := range defs {
		if err := r.Register(d.pattern, d.handler, srcGRPC); err != nil {
			return err
		}
	}
	return nil
}

// stepGRPCServiceHealthy handles: the gRPC service at "TARGET" should be healthy
//
// Uses the gRPC Health Checking Protocol (grpc.health.v1) with an empty
// service name, which checks the overall server health.
func stepGRPCServiceHealthy(ctx *steps.ScenarioContext, args ...string) error {
	target := args[0]
	status, err := grpcHealthCheck(target, "")
	if err != nil {
		return softOrHard(ctx, fmt.Errorf("gRPC health check for %q failed: %w", target, err))
	}
	if status != healthpb.HealthCheckResponse_SERVING {
		e := fmt.Errorf("gRPC service at %q is not SERVING (status: %s)", target, status)
		return softOrHard(ctx, e)
	}
	return nil
}

// stepGRPCServiceNameHealthy handles:
//
//	the gRPC service at "TARGET" serving "SERVICE_NAME" should be healthy
//
// Checks the health of a named gRPC service within the server.
func stepGRPCServiceNameHealthy(ctx *steps.ScenarioContext, args ...string) error {
	target := args[0]
	service := args[1]
	status, err := grpcHealthCheck(target, service)
	if err != nil {
		return softOrHard(ctx, fmt.Errorf("gRPC health check for %q service %q failed: %w", target, service, err))
	}
	if status != healthpb.HealthCheckResponse_SERVING {
		e := fmt.Errorf("gRPC service %q at %q is not SERVING (status: %s)", service, target, status)
		return softOrHard(ctx, e)
	}
	return nil
}

// stepWaitForGRPCService handles: I wait up to Ns for the gRPC service at "TARGET" to be healthy
func stepWaitForGRPCService(_ *steps.ScenarioContext, args ...string) error {
	var timeoutSecs int
	if _, err := fmt.Sscanf(args[0], "%d", &timeoutSecs); err != nil {
		return fmt.Errorf("invalid timeout %q", args[0])
	}
	target := args[1]
	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)

	for {
		status, err := grpcHealthCheck(target, "")
		if err == nil && status == healthpb.HealthCheckResponse_SERVING {
			return nil
		}
		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("gRPC service at %q did not become healthy within %ds: %w", target, timeoutSecs, err)
			}
			return fmt.Errorf("gRPC service at %q did not become SERVING within %ds (last status: %s)",
				target, timeoutSecs, status)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// grpcHealthCheck dials the target and calls the gRPC Health Check protocol.
// service may be empty to check overall server health.
func grpcHealthCheck(target, service string) (healthpb.HealthCheckResponse_ServingStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return healthpb.HealthCheckResponse_UNKNOWN, fmt.Errorf("dial %q: %w", target, err)
	}
	defer conn.Close()

	resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{
		Service: service,
	})
	if err != nil {
		return healthpb.HealthCheckResponse_UNKNOWN, err
	}
	return resp.GetStatus(), nil
}

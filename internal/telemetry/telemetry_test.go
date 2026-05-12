package telemetry

import (
	"context"
	"testing"
)

func TestSetup_emptyEndpoint_returnsNoopProvider(t *testing.T) {
	t.Parallel()
	p, err := Setup(context.Background(), Config{})
	if err != nil {
		t.Fatalf("Setup with empty endpoint: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil Provider")
	}
}

func TestSetup_emptyEndpoint_shutdownIsNoOp(t *testing.T) {
	t.Parallel()
	p, err := Setup(context.Background(), Config{})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown on noop provider: %v", err)
	}
}

func TestSetup_defaultServiceName(t *testing.T) {
	t.Parallel()
	// Empty ServiceName should not cause an error.
	p, err := Setup(context.Background(), Config{ServiceName: ""})
	if err != nil {
		t.Fatalf("Setup with empty ServiceName: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil Provider")
	}
}

func TestSetup_customServiceName(t *testing.T) {
	t.Parallel()
	p, err := Setup(context.Background(), Config{ServiceName: "my-service"})
	if err != nil {
		t.Fatalf("Setup with custom ServiceName: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil Provider")
	}
}

func TestTracer_returnsNonNil(t *testing.T) {
	t.Parallel()
	// Ensure the global provider is initialised (noop).
	if _, err := Setup(context.Background(), Config{}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	tr := Tracer("test")
	if tr == nil {
		t.Fatal("Tracer() returned nil")
	}
}

func TestProvider_shutdownMultipleTimes_noError(t *testing.T) {
	t.Parallel()
	p, err := Setup(context.Background(), Config{})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	// Calling Shutdown twice should not panic or error on a noop provider.
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("first Shutdown: %v", err)
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("second Shutdown: %v", err)
	}
}

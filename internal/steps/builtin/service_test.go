package builtin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bcp-technology-ug/lobster/internal/steps"
)

func TestService_isRunning_up(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	ctx := steps.NewScenarioContext(srv.URL, nil, nil)
	ctx.HTTPClient = srv.Client()

	if err := stepServiceIsRunning(ctx, "myservice"); err != nil {
		t.Fatalf("stepServiceIsRunning: %v", err)
	}
}

func TestService_isRunning_down(t *testing.T) {
	t.Parallel()
	// No server; use unreachable address.
	ctx := steps.NewScenarioContext("http://127.0.0.1:1", nil, nil)

	if err := stepServiceIsRunning(ctx, "myservice"); err == nil {
		t.Error("expected error for unreachable service")
	}
}

func TestService_isRunningAt(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	ctx := steps.NewScenarioContext("", nil, nil)
	ctx.HTTPClient = srv.Client()

	if err := stepServiceIsRunningAt(ctx, "ignored", srv.URL); err != nil {
		t.Fatalf("stepServiceIsRunningAt: %v", err)
	}
}

func TestService_waitForService_immediate(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	ctx := steps.NewScenarioContext(srv.URL, nil, nil)
	ctx.HTTPClient = srv.Client()

	if err := stepWaitForService(ctx, "2", "myservice"); err != nil {
		t.Fatalf("stepWaitForService: %v", err)
	}
}

func TestService_waitForService_timeout(t *testing.T) {
	t.Parallel()
	ctx := steps.NewScenarioContext("http://127.0.0.1:1", nil, nil)

	if err := stepWaitForService(ctx, "1", "myservice"); err == nil {
		t.Error("expected timeout error for unreachable service")
	}
}

// Package adminsvc_test provides unit tests for the AdminService.
package adminsvc_test

import (
	"context"
	"testing"

	adminv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/admin"
	configv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/config"
	"github.com/bcp-technology-ug/lobster/internal/api/adminsvc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// newService creates an adminsvc.Service with a nil store (sufficient for
// capability and config-summary RPCs, which do not touch the store).
func newService(cfgFn func() *adminv1.ConfigSummary) *adminsvc.Service {
	return adminsvc.New(nil, "v0.1.0-test", cfgFn)
}

func TestGetCapabilities_ReturnsAllCapabilities(t *testing.T) {
	t.Parallel()

	svc := newService(nil)
	resp, err := svc.GetCapabilities(context.Background(), &adminv1.GetCapabilitiesRequest{})
	if err != nil {
		t.Fatalf("GetCapabilities: %v", err)
	}
	if resp.ApiPackage != "lobster.v1" {
		t.Errorf("ApiPackage = %q, want %q", resp.ApiPackage, "lobster.v1")
	}
	if resp.ApiVersion != "v0.1" {
		t.Errorf("ApiVersion = %q, want %q", resp.ApiVersion, "v0.1")
	}
	if len(resp.Capabilities) == 0 {
		t.Error("Capabilities is empty")
	}
	// All reported capabilities must be enabled in v0.1.
	for _, c := range resp.Capabilities {
		if !c.Enabled {
			t.Errorf("capability %q is disabled, want enabled", c.Name)
		}
	}
}

func TestGetConfigSummary_NilConfigFunc_ReturnsUnavailable(t *testing.T) {
	t.Parallel()

	svc := newService(nil)
	_, err := svc.GetConfigSummary(context.Background(), &adminv1.GetConfigSummaryRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T", err)
	}
	if st.Code() != codes.Unavailable {
		t.Errorf("code = %v, want %v", st.Code(), codes.Unavailable)
	}
}

func TestGetConfigSummary_WithConfigFunc(t *testing.T) {
	t.Parallel()

	cfgFn := func() *adminv1.ConfigSummary {
		return adminsvc.MakeConfigSummary("ws1", "staging", nil, nil, nil)
	}
	svc := newService(cfgFn)

	resp, err := svc.GetConfigSummary(context.Background(), &adminv1.GetConfigSummaryRequest{})
	if err != nil {
		t.Fatalf("GetConfigSummary: %v", err)
	}
	if resp.Config == nil {
		t.Fatal("Config is nil")
	}
	if resp.Config.WorkspaceId != "ws1" {
		t.Errorf("WorkspaceId = %q, want %q", resp.Config.WorkspaceId, "ws1")
	}
	if resp.Config.ActiveProfile != "staging" {
		t.Errorf("ActiveProfile = %q, want %q", resp.Config.ActiveProfile, "staging")
	}
}

func TestMakeConfigSummary_PopulatesAllFields(t *testing.T) {
	t.Parallel()

	exec := &configv1.ExecutionConfig{SoftAssert: true, FailFast: false}
	compose := &configv1.ComposeConfig{ProjectName: "myproject"}
	persist := &configv1.PersistenceConfig{SqlitePath: "/tmp/lobster.db"}

	sum := adminsvc.MakeConfigSummary("workspace-abc", "dev", exec, compose, persist)

	if sum.WorkspaceId != "workspace-abc" {
		t.Errorf("WorkspaceId = %q", sum.WorkspaceId)
	}
	if sum.ActiveProfile != "dev" {
		t.Errorf("ActiveProfile = %q", sum.ActiveProfile)
	}
	if sum.Execution == nil || !sum.Execution.SoftAssert {
		t.Error("Execution.SoftAssert not set")
	}
	if sum.Compose == nil || sum.Compose.ProjectName != "myproject" {
		t.Error("Compose.ProjectName not set")
	}
	if sum.Persistence == nil || sum.Persistence.SqlitePath != "/tmp/lobster.db" {
		t.Error("Persistence.SqlitePath not set")
	}
}

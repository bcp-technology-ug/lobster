package integrationsvc_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	integrationsv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/integrations"
	integrationstore "github.com/bcp-technology-ug/lobster/gen/sqlc/integrations"
	"github.com/bcp-technology-ug/lobster/internal/api/convert"
	"github.com/bcp-technology-ug/lobster/internal/api/integrationsvc"
	"github.com/bcp-technology-ug/lobster/internal/store"
	"github.com/bcp-technology-ug/lobster/internal/testutil"
)

func upsertAdapter(t *testing.T, st *store.Store, ctx context.Context, adapterID, name string) {
	t.Helper()
	if err := st.Integrations.UpsertIntegrationAdapter(ctx, integrationstore.UpsertIntegrationAdapterParams{
		AdapterID: adapterID,
		Name:      name,
		Type:      "keycloak",
		State:     int64(integrationsv1.AdapterState_ADAPTER_STATE_READY),
		UpdatedAt: convert.NowDB(),
	}); err != nil {
		t.Fatalf("UpsertIntegrationAdapter: %v", err)
	}
}

// _ silences the unused warning since it's used below.
var _ = upsertAdapter

func TestListIntegrationAdapters_empty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := integrationsvc.New(st, nil)

	resp, err := svc.ListIntegrationAdapters(ctx, &integrationsv1.ListIntegrationAdaptersRequest{})
	if err != nil {
		t.Fatalf("ListIntegrationAdapters error: %v", err)
	}
	if len(resp.Adapters) != 0 {
		t.Errorf("expected 0 adapters, got %d", len(resp.Adapters))
	}
}

func TestGetIntegrationAdapter_notFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := integrationsvc.New(st, nil)

	_, err := svc.GetIntegrationAdapter(ctx, &integrationsv1.GetIntegrationAdapterRequest{AdapterId: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for missing adapter")
	}
	st2, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if st2.Code() != codes.NotFound {
		t.Errorf("code: got %v want NotFound", st2.Code())
	}
}

func TestGetIntegrationAdapter_found(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := integrationsvc.New(st, nil)

	if err := st.Integrations.UpsertIntegrationAdapter(ctx, integrationstore.UpsertIntegrationAdapterParams{
		AdapterID: "adapter-001",
		Name:      "Keycloak Primary",
		Type:      "keycloak",
		State:     int64(integrationsv1.AdapterState_ADAPTER_STATE_READY),
		UpdatedAt: convert.NowDB(),
	}); err != nil {
		t.Fatalf("UpsertIntegrationAdapter: %v", err)
	}

	resp, err := svc.GetIntegrationAdapter(ctx, &integrationsv1.GetIntegrationAdapterRequest{AdapterId: "adapter-001"})
	if err != nil {
		t.Fatalf("GetIntegrationAdapter error: %v", err)
	}
	if resp.Adapter.AdapterId != "adapter-001" {
		t.Errorf("AdapterId: got %q want %q", resp.Adapter.AdapterId, "adapter-001")
	}
}

func TestSetIntegrationAdapterState_disable(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := integrationsvc.New(st, nil)

	if err := st.Integrations.UpsertIntegrationAdapter(ctx, integrationstore.UpsertIntegrationAdapterParams{
		AdapterID: "adapter-disable",
		Name:      "Keycloak",
		Type:      "keycloak",
		State:     int64(integrationsv1.AdapterState_ADAPTER_STATE_READY),
		UpdatedAt: convert.NowDB(),
	}); err != nil {
		t.Fatalf("UpsertIntegrationAdapter: %v", err)
	}

	resp, err := svc.SetIntegrationAdapterState(ctx, &integrationsv1.SetIntegrationAdapterStateRequest{
		AdapterId: "adapter-disable",
		Enabled:   false,
		Reason:    "test disable",
	})
	if err != nil {
		t.Fatalf("SetIntegrationAdapterState error: %v", err)
	}
	if resp.Adapter.State != integrationsv1.AdapterState_ADAPTER_STATE_DISABLED {
		t.Errorf("State: got %v want DISABLED", resp.Adapter.State)
	}
}

func TestValidateIntegrationAdapter_noValidator(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := integrationsvc.New(st, nil) // nil validator

	_, err := svc.ValidateIntegrationAdapter(ctx, &integrationsv1.ValidateIntegrationAdapterRequest{AdapterId: "any"})
	if err == nil {
		t.Fatal("expected error when no validator configured")
	}
	st2, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if st2.Code() != codes.Unimplemented {
		t.Errorf("code: got %v want Unimplemented", st2.Code())
	}
}

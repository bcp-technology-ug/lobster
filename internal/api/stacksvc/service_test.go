package stacksvc_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	stackv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/stack"
	stackstore "github.com/bcp-technology-ug/lobster/gen/sqlc/stack"
	"github.com/bcp-technology-ug/lobster/internal/api/convert"
	"github.com/bcp-technology-ug/lobster/internal/api/stacksvc"
	"github.com/bcp-technology-ug/lobster/internal/testutil"
)

func TestGetStackStatus_notFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := stacksvc.New(st, nil)

	_, err := svc.GetStackStatus(ctx, &stackv1.GetStackStatusRequest{WorkspaceId: "ws-missing"})
	if err == nil {
		t.Fatal("expected error for missing stack")
	}
	st2, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if st2.Code() != codes.NotFound {
		t.Errorf("code: got %v want NotFound", st2.Code())
	}
}

func TestGetStackStatus_found(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := stacksvc.New(st, nil)

	now := convert.NowDB()
	stackID := "stack-001"
	if err := st.Stack.UpsertStack(ctx, stackstore.UpsertStackParams{
		StackID:     stackID,
		WorkspaceID: "ws-stack",
		ProfileName: "default",
		ProjectName: "test-project",
		Status:      int64(stackv1.StackStatus_STACK_STATUS_HEALTHY),
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("UpsertStack: %v", err)
	}

	resp, err := svc.GetStackStatus(ctx, &stackv1.GetStackStatusRequest{WorkspaceId: "ws-stack"})
	if err != nil {
		t.Fatalf("GetStackStatus error: %v", err)
	}
	if resp.Stack == nil {
		t.Fatal("expected non-nil stack")
	}
	if resp.Stack.WorkspaceId != "ws-stack" {
		t.Errorf("WorkspaceId: got %q want %q", resp.Stack.WorkspaceId, "ws-stack")
	}
}

func TestGetStackStatus_withComponents(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := stacksvc.New(st, nil)

	now := convert.NowDB()
	stackID := "stack-comps"
	if err := st.Stack.UpsertStack(ctx, stackstore.UpsertStackParams{
		StackID:     stackID,
		WorkspaceID: "ws-comps",
		ProfileName: "default",
		ProjectName: "test-project",
		Status:      int64(stackv1.StackStatus_STACK_STATUS_HEALTHY),
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("UpsertStack: %v", err)
	}
	if err := st.Stack.UpsertStackComponent(ctx, stackstore.UpsertStackComponentParams{
		StackID:   stackID,
		Name:      "api",
		Health:    int64(stackv1.ServiceHealth_SERVICE_HEALTH_HEALTHY),
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("UpsertStackComponent: %v", err)
	}

	resp, err := svc.GetStackStatus(ctx, &stackv1.GetStackStatusRequest{WorkspaceId: "ws-comps"})
	if err != nil {
		t.Fatalf("GetStackStatus error: %v", err)
	}
	if len(resp.Stack.Services) != 1 {
		t.Errorf("expected 1 service component, got %d", len(resp.Stack.Services))
	}
	if resp.Stack.Services[0].Name != "api" {
		t.Errorf("component name: got %q want %q", resp.Stack.Services[0].Name, "api")
	}
}

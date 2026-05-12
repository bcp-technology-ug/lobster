package plansvc_test

import (
	"context"
	"testing"
	"time"

	planv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/plan"
	planstore "github.com/bcp-technology-ug/lobster/gen/sqlc/plan"
	"github.com/bcp-technology-ug/lobster/internal/api/convert"
	"github.com/bcp-technology-ug/lobster/internal/api/plansvc"
	"github.com/bcp-technology-ug/lobster/internal/testutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetPlan_notFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := plansvc.New(st, nil)

	_, err := svc.GetPlan(ctx, &planv1.GetPlanRequest{PlanId: "nonexistent-plan"})
	if err == nil {
		t.Fatal("expected error for missing plan")
	}
	st2, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if st2.Code() != codes.NotFound {
		t.Errorf("code: got %v want NotFound", st2.Code())
	}
}

func TestGetPlan_found(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := plansvc.New(st, nil)

	now := convert.NowDB()
	planID := "plan-get-001"
	if err := st.Plan.CreateExecutionPlan(ctx, planstore.CreateExecutionPlanParams{
		PlanID:      planID,
		WorkspaceID: "ws-001",
		ProfileName: "default",
		CreatedAt:   now,
	}); err != nil {
		t.Fatalf("CreateExecutionPlan: %v", err)
	}

	resp, err := svc.GetPlan(ctx, &planv1.GetPlanRequest{PlanId: planID})
	if err != nil {
		t.Fatalf("GetPlan error: %v", err)
	}
	if resp.Plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if resp.Plan.PlanId != planID {
		t.Errorf("PlanId: got %q want %q", resp.Plan.PlanId, planID)
	}
}

func TestListPlans_empty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := plansvc.New(st, nil)

	resp, err := svc.ListPlans(ctx, &planv1.ListPlansRequest{WorkspaceId: "ws-empty"})
	if err != nil {
		t.Fatalf("ListPlans error: %v", err)
	}
	if len(resp.Plans) != 0 {
		t.Errorf("expected 0 plans, got %d", len(resp.Plans))
	}
}

func TestListPlans_withPlans(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := plansvc.New(st, nil)

	for i := 0; i < 2; i++ {
		now := convert.NowDB()
		time.Sleep(time.Millisecond)
		planID := convert.EncodeCursor(now, string(rune('a'+i)))
		if err := st.Plan.CreateExecutionPlan(ctx, planstore.CreateExecutionPlanParams{
			PlanID:      planID,
			WorkspaceID: "ws-list",
			ProfileName: "default",
			CreatedAt:   now,
		}); err != nil {
			t.Fatalf("CreateExecutionPlan[%d]: %v", i, err)
		}
	}

	resp, err := svc.ListPlans(ctx, &planv1.ListPlansRequest{WorkspaceId: "ws-list"})
	if err != nil {
		t.Fatalf("ListPlans error: %v", err)
	}
	if len(resp.Plans) != 2 {
		t.Errorf("expected 2 plans, got %d", len(resp.Plans))
	}
}

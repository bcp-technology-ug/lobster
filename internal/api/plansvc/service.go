// Package plansvc implements the gRPC PlanService server.
package plansvc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/bcp-technology/lobster/internal/api/convert"
	"github.com/bcp-technology/lobster/internal/store"

	planv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/plan"
	planstore "github.com/bcp-technology/lobster/gen/sqlc/plan"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Planner is the interface the PlanService delegates execution planning to.
// Batch 2 (runner/parser package) will provide the production implementation.
type Planner interface {
	Plan(ctx context.Context, req *planv1.PlanRequest) (*planv1.PlanResponse, error)
}

// Service implements planv1.PlanServiceServer backed by SQLite persistence.
type Service struct {
	planv1.UnimplementedPlanServiceServer

	store   *store.Store
	planner Planner
}

// New creates a Service. planner may be nil; Plan will return Unimplemented
// until a planner is wired.
func New(st *store.Store, planner Planner) *Service {
	return &Service{store: st, planner: planner}
}

// Plan delegates to the planner if present; otherwise returns Unimplemented.
func (s *Service) Plan(ctx context.Context, req *planv1.PlanRequest) (*planv1.PlanResponse, error) {
	if s.planner == nil {
		return nil, status.Error(codes.Unimplemented, "planner not configured")
	}
	return s.planner.Plan(ctx, req)
}

// GetPlan fetches a stored execution plan by ID.
func (s *Service) GetPlan(ctx context.Context, req *planv1.GetPlanRequest) (*planv1.GetPlanResponse, error) {
	row, err := s.store.Plan.GetExecutionPlan(ctx, req.PlanId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "plan %q not found", req.PlanId)
		}
		return nil, status.Errorf(codes.Internal, "get plan: %v", err)
	}
	plan := convert.ExecutionPlanFromDB(row)
	if err := s.attachPlanDetail(ctx, plan); err != nil {
		return nil, status.Errorf(codes.Internal, "load plan detail: %v", err)
	}
	return &planv1.GetPlanResponse{Plan: plan}, nil
}

// ListPlans returns a paginated list of plans for a workspace.
func (s *Service) ListPlans(ctx context.Context, req *planv1.ListPlansRequest) (*planv1.ListPlansResponse, error) {
	pageSize := convert.PageSizeOrDefault(req.PageSize)

	cursorCreatedAt, cursorPlanID, err := convert.DecodeCursor(req.PageToken)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	rows, err := s.store.Plan.ListExecutionPlansPage(ctx, planstore.ListExecutionPlansPageParams{
		WorkspaceID:     req.WorkspaceId,
		CursorCreatedAt: ptrStrToInterface(cursorCreatedAt),
		CursorPlanID:    cursorPlanID,
		PageSize:        pageSize,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list plans: %v", err)
	}

	plans := make([]*planv1.ExecutionPlan, 0, len(rows))
	for _, r := range rows {
		plans = append(plans, convert.ExecutionPlanFromDB(r))
	}

	var nextToken string
	if int64(len(rows)) == pageSize {
		last := rows[len(rows)-1]
		nextToken = convert.EncodeCursor(last.CreatedAt, last.PlanID)
	}

	return &planv1.ListPlansResponse{
		Plans:         plans,
		NextPageToken: nextToken,
	}, nil
}

// attachPlanDetail loads scenarios and artifact and attaches them to plan.
func (s *Service) attachPlanDetail(ctx context.Context, plan *planv1.ExecutionPlan) error {
	scenarios, err := s.store.Plan.ListExecutionPlanScenarios(ctx, plan.PlanId)
	if err != nil {
		return err
	}
	for _, sc := range scenarios {
		sp := convert.ScenarioPlanFromDB(sc)
		// Load tags for each scenario.
		tags, err := s.store.Plan.ListExecutionPlanScenarioTags(ctx, planstore.ListExecutionPlanScenarioTagsParams{
			PlanID:     plan.PlanId,
			ScenarioID: sc.ScenarioID,
		})
		if err == nil {
			for _, t := range tags {
				sp.Tags = append(sp.Tags, t.Tag)
			}
		}
		plan.Scenarios = append(plan.Scenarios, sp)
	}

	artifact, err := s.store.Plan.GetPlanArtifactByPlanID(ctx, plan.PlanId)
	if err == nil {
		plan.Artifact = convert.PlanArtifactFromDB(artifact)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return nil
}

func ptrStrToInterface(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

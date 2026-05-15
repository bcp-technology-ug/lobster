// Package stacksvc implements the gRPC StackService server.
package stacksvc

import (
	"context"
	"database/sql"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/common"
	stackv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/stack"
	stackstore "github.com/bcp-technology-ug/lobster/gen/sqlc/stack"
	"github.com/bcp-technology-ug/lobster/internal/api/convert"
	"github.com/bcp-technology-ug/lobster/internal/store"
)

// Orchestrator is the interface the StackService delegates stack lifecycle to.
// Batch 3 (orchestration package) will provide the Docker SDK implementation.
type Orchestrator interface {
	// EnsureStack starts compose services and waits for readiness.
	EnsureStack(ctx context.Context, req *stackv1.EnsureStackRequest) (*stackv1.EnsureStackResponse, error)
	// TeardownStack stops and removes compose services.
	TeardownStack(ctx context.Context, req *stackv1.TeardownStackRequest) (*stackv1.TeardownStackResponse, error)
	// GetStackLogs streams service logs.
	GetStackLogs(req *stackv1.GetStackLogsRequest, stream stackv1.StackService_GetStackLogsServer) error
}

// Service implements stackv1.StackServiceServer.
type Service struct {
	stackv1.UnimplementedStackServiceServer

	store        *store.Store
	orchestrator Orchestrator
}

// New creates a Service. orchestrator may be nil; EnsureStack, TeardownStack,
// and GetStackLogs will return Unimplemented until it is wired.
func New(st *store.Store, orch Orchestrator) *Service {
	return &Service{store: st, orchestrator: orch}
}

// EnsureStack delegates to the orchestrator if present and persists the resulting
// stack state so that GetStackStatus can reflect reality.
func (s *Service) EnsureStack(ctx context.Context, req *stackv1.EnsureStackRequest) (*stackv1.EnsureStackResponse, error) {
	if s.orchestrator == nil {
		return nil, status.Error(codes.Unimplemented, "orchestrator not configured")
	}
	resp, err := s.orchestrator.EnsureStack(ctx, req)
	if err != nil {
		return nil, err
	}
	// Persist stack state (best-effort; stack is up even if DB write fails).
	if resp.Stack != nil {
		if upsertErr := s.UpsertStack(ctx, resp.Stack); upsertErr != nil {
			// Log but do not fail the caller — the stack itself is running.
			_ = upsertErr
		}
	}
	return resp, nil
}

// GetStackStatus returns the persisted stack state for a workspace.
func (s *Service) GetStackStatus(ctx context.Context, req *stackv1.GetStackStatusRequest) (*stackv1.GetStackStatusResponse, error) {
	row, err := s.store.Stack.GetStackByWorkspace(ctx, req.WorkspaceId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "no stack found for workspace %q", req.WorkspaceId)
		}
		return nil, status.Errorf(codes.Internal, "get stack: %v", err)
	}
	stack := convert.StackFromDB(row)
	if err := s.attachComponents(ctx, stack, row.StackID); err != nil {
		return nil, status.Errorf(codes.Internal, "load stack components: %v", err)
	}

	resp := &stackv1.GetStackStatusResponse{Stack: stack}

	// Add diagnostics when stack is degraded or unhealthy.
	if row.Status == int64(stackv1.StackStatus_STACK_STATUS_DEGRADED) ||
		row.Status == int64(stackv1.StackStatus_STACK_STATUS_UNHEALTHY) {
		resp.Diagnostics = append(resp.Diagnostics, &commonv1.ErrorRef{
			Code:    commonv1.ErrorCode_ERROR_CODE_INTERNAL,
			Message: "stack is not healthy; check service logs for details",
		})
	}

	return resp, nil
}

// TeardownStack delegates to the orchestrator if present and marks the stack
// record as torn down so GetStackStatus reflects the new state.
func (s *Service) TeardownStack(ctx context.Context, req *stackv1.TeardownStackRequest) (*stackv1.TeardownStackResponse, error) {
	if s.orchestrator == nil {
		return nil, status.Error(codes.Unimplemented, "orchestrator not configured")
	}
	resp, err := s.orchestrator.TeardownStack(ctx, req)
	if err != nil {
		return nil, err
	}
	// Update persisted stack status to TEARDOWN (best-effort).
	if existing, dbErr := s.store.Stack.GetStackByWorkspace(ctx, req.WorkspaceId); dbErr == nil {
		now := convert.NowDB()
		_ = s.store.Stack.UpsertStack(ctx, stackstore.UpsertStackParams{
			StackID:     existing.StackID,
			WorkspaceID: existing.WorkspaceID,
			ProfileName: existing.ProfileName,
			ProjectName: existing.ProjectName,
			Status:      int64(stackv1.StackStatus_STACK_STATUS_TEARDOWN),
			CreatedAt:   existing.CreatedAt,
			UpdatedAt:   now,
		})
	}
	return resp, nil
}

// GetStackLogs delegates to the orchestrator if present.
func (s *Service) GetStackLogs(req *stackv1.GetStackLogsRequest, stream stackv1.StackService_GetStackLogsServer) error {
	if s.orchestrator == nil {
		return status.Error(codes.Unimplemented, "orchestrator not configured")
	}
	return s.orchestrator.GetStackLogs(req, stream)
}

// attachComponents loads stack components and appends them to the proto Stack.
func (s *Service) attachComponents(ctx context.Context, stack *stackv1.Stack, stackID string) error {
	comps, err := s.store.Stack.ListStackComponents(ctx, stackID)
	if err != nil {
		return err
	}
	for _, c := range comps {
		stack.Services = append(stack.Services, convert.StackComponentFromDB(c))
	}
	return nil
}

// UpsertStack persists a Stack record (used by the orchestration layer to
// write-back stack state without holding a direct store reference).
func (s *Service) UpsertStack(ctx context.Context, stack *stackv1.Stack) error {
	now := convert.NowDB()
	createdAt := now
	if stack.CreatedAt != nil {
		createdAt = convert.TimestampToDB(stack.CreatedAt)
	}
	if err := s.store.Stack.UpsertStack(ctx, stackstore.UpsertStackParams{
		StackID:     stack.StackId,
		WorkspaceID: stack.WorkspaceId,
		ProfileName: "",
		ProjectName: stack.ProjectName,
		Status:      int64(stack.Status),
		CreatedAt:   createdAt,
		UpdatedAt:   now,
	}); err != nil {
		return err
	}
	for _, svc := range stack.Services {
		if err := s.store.Stack.UpsertStackComponent(ctx, stackstore.UpsertStackComponentParams{
			StackID:     stack.StackId,
			Name:        svc.Name,
			Image:       nullableStr(svc.Image),
			ContainerID: nullableStr(svc.ContainerId),
			Status:      nullableStr(svc.Status),
			Health:      int64(svc.Health),
			UpdatedAt:   now,
		}); err != nil {
			return err
		}
	}
	return nil
}

func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

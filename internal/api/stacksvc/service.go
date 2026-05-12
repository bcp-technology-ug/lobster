// Package stacksvc implements the gRPC StackService server.
package stacksvc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/bcp-technology/lobster/internal/api/convert"
	"github.com/bcp-technology/lobster/internal/store"

	commonv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/common"
	stackv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/stack"
	stackstore "github.com/bcp-technology/lobster/gen/sqlc/stack"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// EnsureStack delegates to the orchestrator if present.
func (s *Service) EnsureStack(ctx context.Context, req *stackv1.EnsureStackRequest) (*stackv1.EnsureStackResponse, error) {
	if s.orchestrator == nil {
		return nil, status.Error(codes.Unimplemented, "orchestrator not configured")
	}
	return s.orchestrator.EnsureStack(ctx, req)
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

// TeardownStack delegates to the orchestrator if present.
func (s *Service) TeardownStack(ctx context.Context, req *stackv1.TeardownStackRequest) (*stackv1.TeardownStackResponse, error) {
	if s.orchestrator == nil {
		return nil, status.Error(codes.Unimplemented, "orchestrator not configured")
	}
	return s.orchestrator.TeardownStack(ctx, req)
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

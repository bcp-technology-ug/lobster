// Package runsvc implements the gRPC RunService server.
package runsvc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bcp-technology/lobster/internal/api/convert"
	"github.com/bcp-technology/lobster/internal/store"

	commonv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/common"
	runv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/run"
	runstore "github.com/bcp-technology/lobster/gen/sqlc/run"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Runner is the interface the RunService delegates actual execution to.
type Runner interface {
	// RunSync executes a run and streams events until completion.
	RunSync(ctx context.Context, req *runv1.RunSyncRequest, stream runv1.RunService_RunSyncServer) error
	// RunAsync creates a persisted run record and starts execution in the background.
	RunAsync(ctx context.Context, req *runv1.RunAsyncRequest) (*runv1.RunAsyncResponse, error)
	// Cancel signals the background goroutine for runID to stop. Idempotent.
	Cancel(ctx context.Context, runID string) error
}

// Service implements runv1.RunServiceServer backed by SQLite persistence.
type Service struct {
	runv1.UnimplementedRunServiceServer

	store  *store.Store
	runner Runner
}

// New creates a Service. runner may be nil; RunSync and RunAsync will return
// Unimplemented until a runner is wired.
func New(st *store.Store, runner Runner) *Service {
	return &Service{store: st, runner: runner}
}

// RunSync delegates to the runner if present; otherwise returns Unimplemented.
func (s *Service) RunSync(req *runv1.RunSyncRequest, stream runv1.RunService_RunSyncServer) error {
	if s.runner == nil {
		return status.Error(codes.Unimplemented, "execution engine not configured")
	}
	return s.runner.RunSync(stream.Context(), req, stream)
}

// RunAsync delegates to the runner if present; otherwise returns Unimplemented.
func (s *Service) RunAsync(ctx context.Context, req *runv1.RunAsyncRequest) (*runv1.RunAsyncResponse, error) {
	if s.runner == nil {
		return nil, status.Error(codes.Unimplemented, "execution engine not configured")
	}

	// Idempotency check: if key provided and run already exists, return replay.
	if req.IdempotencyKey != "" {
		idempKey := req.IdempotencyKey
		existing, err := s.store.Run.GetRunByWorkspaceAndIdempotencyKey(ctx, runstore.GetRunByWorkspaceAndIdempotencyKeyParams{
			WorkspaceID:    req.Selector.WorkspaceId,
			IdempotencyKey: &idempKey,
		})
		if err == nil {
			return &runv1.RunAsyncResponse{
				RunId:            existing.RunID,
				AcceptedAt:       convert.TimestampFromDBStr(existing.CreatedAt),
				IdempotentReplay: true,
				ReplayOfRunId:    existing.RunID,
			}, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.Internal, "idempotency check: %v", err)
		}
	}

	return s.runner.RunAsync(ctx, req)
}

// GetRun fetches a single run by ID.
func (s *Service) GetRun(ctx context.Context, req *runv1.GetRunRequest) (*runv1.GetRunResponse, error) {
	row, err := s.store.Run.GetRun(ctx, req.RunId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "run %q not found", req.RunId)
		}
		return nil, status.Errorf(codes.Internal, "get run: %v", err)
	}
	run := convert.RunFromDB(row)
	if err := s.attachRunDetail(ctx, run); err != nil {
		return nil, status.Errorf(codes.Internal, "load run detail: %v", err)
	}
	return &runv1.GetRunResponse{Run: run}, nil
}

// StreamRunEvents streams persisted run events from a given sequence onward.
// This implementation polls the DB for new events; a future version will use
// a pub/sub channel from the runner for low-latency live streaming.
func (s *Service) StreamRunEvents(req *runv1.StreamRunEventsRequest, stream runv1.RunService_StreamRunEventsServer) error {
	ctx := stream.Context()
	fromSeq := int64(req.FromSequence)
	const pollInterval = 500 * time.Millisecond

	// Verify run exists.
	if _, err := s.store.Run.GetRun(ctx, req.RunId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return status.Errorf(codes.NotFound, "run %q not found", req.RunId)
		}
		return status.Errorf(codes.Internal, "get run: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		events, err := s.store.Run.ListRunEventsFromSequence(ctx, runstore.ListRunEventsFromSequenceParams{
			RunID:        req.RunId,
			FromSequence: fromSeq,
		})
		if err != nil {
			return status.Errorf(codes.Internal, "list events: %v", err)
		}

		for _, e := range events {
			evt := convert.RunEventFromDB(e)
			if err := stream.Send(evt); err != nil {
				return err
			}
			fromSeq = e.Sequence + 1
			if e.Terminal == 1 {
				return nil
			}
		}

		// If we received no events and no terminal, wait before polling again.
		if len(events) == 0 {
			// Check if run is in a terminal state.
			row, err := s.store.Run.GetRun(ctx, req.RunId)
			if err == nil {
				st := commonv1.RunStatus(row.Status)
				switch st {
				case commonv1.RunStatus_RUN_STATUS_PASSED,
					commonv1.RunStatus_RUN_STATUS_FAILED,
					commonv1.RunStatus_RUN_STATUS_CANCELLED:
					return nil
				}
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(pollInterval):
			}
		}
	}
}

// CancelRun marks a running or pending run as cancelled.
func (s *Service) CancelRun(ctx context.Context, req *runv1.CancelRunRequest) (*runv1.CancelRunResponse, error) {
	row, err := s.store.Run.GetRun(ctx, req.RunId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "run %q not found", req.RunId)
		}
		return nil, status.Errorf(codes.Internal, "get run: %v", err)
	}
	st := commonv1.RunStatus(row.Status)
	switch st {
	case commonv1.RunStatus_RUN_STATUS_PASSED,
		commonv1.RunStatus_RUN_STATUS_FAILED,
		commonv1.RunStatus_RUN_STATUS_CANCELLED:
		// Already terminal — idempotent.
		return &runv1.CancelRunResponse{
			RunId:          req.RunId,
			TerminalStatus: st,
		}, nil
	}

	now := convert.NowDB()
	if err := s.store.Run.UpdateRunStatus(ctx, runstore.UpdateRunStatusParams{
		RunID:   req.RunId,
		Status:  int64(commonv1.RunStatus_RUN_STATUS_CANCELLED),
		EndedAt: &now,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "cancel run: %v", err)
	}

	// Append a cancellation event.
	seq := time.Now().UnixNano()
	statusVal := int64(commonv1.RunStatus_RUN_STATUS_CANCELLED)
	_ = s.store.Run.AppendRunEvent(ctx, runstore.AppendRunEventParams{
		RunID:            req.RunId,
		Sequence:         seq,
		ObservedAt:       now,
		EventType:        int64(runv1.RunEventType_RUN_EVENT_TYPE_RUN_STATUS),
		Terminal:         1,
		PayloadRunStatus: &statusVal,
	})

	// Signal the background goroutine to stop (idempotent if already done).
	if s.runner != nil {
		_ = s.runner.Cancel(ctx, req.RunId)
	}

	return &runv1.CancelRunResponse{
		RunId:          req.RunId,
		TerminalStatus: commonv1.RunStatus_RUN_STATUS_CANCELLED,
	}, nil
}

// ListRuns returns a paginated list of runs for a workspace.
func (s *Service) ListRuns(ctx context.Context, req *runv1.ListRunsRequest) (*runv1.ListRunsResponse, error) {
	pageSize := convert.PageSizeOrDefault(req.PageSize)

	cursorCreatedAt, cursorRunID, err := convert.DecodeCursor(req.PageToken)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var rows []runstore.Run
	if req.Status != commonv1.RunStatus_RUN_STATUS_UNSPECIFIED {
		rows, err = s.store.Run.ListRunsPageByStatus(ctx, runstore.ListRunsPageByStatusParams{
			WorkspaceID:     req.WorkspaceId,
			Status:          int64(req.Status),
			CursorCreatedAt: ptrStrToInterface(cursorCreatedAt),
			CursorRunID:     cursorRunID,
			PageSize:        pageSize,
		})
	} else {
		rows, err = s.store.Run.ListRunsPage(ctx, runstore.ListRunsPageParams{
			WorkspaceID:     req.WorkspaceId,
			CursorCreatedAt: ptrStrToInterface(cursorCreatedAt),
			CursorRunID:     cursorRunID,
			PageSize:        pageSize,
		})
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list runs: %v", err)
	}

	runs := make([]*runv1.Run, 0, len(rows))
	for _, r := range rows {
		runs = append(runs, convert.RunFromDB(r))
	}

	var nextToken string
	if int64(len(rows)) == pageSize {
		last := rows[len(rows)-1]
		nextToken = convert.EncodeCursor(last.CreatedAt, last.RunID)
	}

	return &runv1.ListRunsResponse{
		Runs:          runs,
		NextPageToken: nextToken,
	}, nil
}

// attachRunDetail loads feature tags and variables and attaches them to run.
// Full scenario/step detail is loaded on demand for GetRun only.
func (s *Service) attachRunDetail(ctx context.Context, run *runv1.Run) error {
	tags, err := s.store.Run.ListRunFeatureTags(ctx, run.RunId)
	if err != nil {
		return fmt.Errorf("list feature tags: %w", err)
	}
	if run.Feature == nil && len(tags) > 0 {
		run.Feature = &runv1.Feature{}
	}
	for _, t := range tags {
		run.Feature.Tags = append(run.Feature.Tags, t.Tag)
	}

	vars, err := s.store.Run.ListRunVariables(ctx, run.RunId)
	if err != nil {
		return fmt.Errorf("list variables: %w", err)
	}
	for _, v := range vars {
		run.Variables = append(run.Variables, &runv1.Variable{
			Name:  v.VariableName,
			Scope: runv1.VariableScope(v.Scope),
		})
	}
	return nil
}

// ptrStrToInterface converts *string to interface{} suitable for sqlc nullable cursors.
func ptrStrToInterface(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

// timestampPBToDBPtr formats a *timestamppb.Timestamp to *string for DB storage.
func timestampPBToDBPtr(ts *timestamppb.Timestamp) *string {
	if ts == nil {
		return nil
	}
	s := ts.AsTime().UTC().Format(time.RFC3339Nano)
	return &s
}

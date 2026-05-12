// Package runsvc implements the gRPC RunService server.
package runsvc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bcp-technology-ug/lobster/internal/api/convert"
	lobsterlog "github.com/bcp-technology-ug/lobster/internal/log"
	"github.com/bcp-technology-ug/lobster/internal/store"

	commonv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/common"
	runv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/run"
	runstore "github.com/bcp-technology-ug/lobster/gen/sqlc/run"

	"go.uber.org/zap"
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
	// SubscribeRunEvents returns the live event channel for a run that is still
	// executing in this process. Returns (nil, false) when the run is not
	// in-flight; callers fall back to DB replay in that case.
	SubscribeRunEvents(runID string) (<-chan *runv1.RunEvent, bool)
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

// StreamRunEvents streams run events from a given sequence onward.
//
// If the run is still executing in this process the implementation subscribes
// to the runner's in-process event bus (zero-latency, no polling). DB replay
// from fromSeq is used first to catch up on events already persisted before
// the subscription was opened, then new events are read from the channel until
// it is closed by the runner at the end of the run.
//
// If the run is not in-flight (already complete, or the daemon was restarted)
// the implementation replays all persisted events from the DB directly.
func (s *Service) StreamRunEvents(req *runv1.StreamRunEventsRequest, stream runv1.RunService_StreamRunEventsServer) error {
	ctx := stream.Context()
	logger := lobsterlog.FromContext(ctx).With(zap.String("run_id", req.RunId))
	fromSeq := int64(req.FromSequence)

	// Verify run exists.
	if _, err := s.store.Run.GetRun(ctx, req.RunId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return status.Errorf(codes.NotFound, "run %q not found", req.RunId)
		}
		return status.Errorf(codes.Internal, "get run: %v", err)
	}

	// Attempt to subscribe to the live bus before replaying DB events. This
	// ensures we receive all events even if new ones arrive between the DB
	// query and the channel read.
	var liveCh <-chan *runv1.RunEvent
	if s.runner != nil {
		liveCh, _ = s.runner.SubscribeRunEvents(req.RunId)
	}

	// Replay events already in DB (from fromSeq).
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

	// If no live channel is available the run is already complete and we have
	// replayed all persisted events above.
	if liveCh == nil {
		return nil
	}

	// Drain the live channel. Events with sequence < fromSeq were already sent
	// via DB replay; skip them to avoid duplicates.
	for {
		select {
		case <-ctx.Done():
			return nil
		case evt, ok := <-liveCh:
			if !ok {
				// Channel closed: run finished. Drain any remaining DB events as
				// a safety net for events that may have been persisted after our
				// last DB replay but before the channel was drained.
				tail, tailErr := s.store.Run.ListRunEventsFromSequence(ctx, runstore.ListRunEventsFromSequenceParams{
					RunID:        req.RunId,
					FromSequence: fromSeq,
				})
				if tailErr != nil {
					logger.Warn("tail DB drain failed", zap.Error(tailErr))
					return nil
				}
				for _, e := range tail {
					if err := stream.Send(convert.RunEventFromDB(e)); err != nil {
						return err
					}
				}
				return nil
			}
			if int64(evt.Sequence) < fromSeq {
				// Already sent via DB replay; skip.
				continue
			}
			if err := stream.Send(evt); err != nil {
				return err
			}
			fromSeq = int64(evt.Sequence) + 1
			if evt.Terminal {
				return nil
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
	if appendErr := s.store.Run.AppendRunEvent(ctx, runstore.AppendRunEventParams{
		RunID:            req.RunId,
		Sequence:         seq,
		ObservedAt:       now,
		EventType:        int64(runv1.RunEventType_RUN_EVENT_TYPE_RUN_STATUS),
		Terminal:         1,
		PayloadRunStatus: &statusVal,
	}); appendErr != nil {
		lobsterlog.FromContext(ctx).Warn("failed to append cancellation event",
			zap.String("run_id", req.RunId), zap.Error(appendErr))
	}

	// Signal the background goroutine to stop (idempotent if already done).
	if s.runner != nil {
		if cancelErr := s.runner.Cancel(ctx, req.RunId); cancelErr != nil {
			lobsterlog.FromContext(ctx).Warn("runner cancel signal failed",
				zap.String("run_id", req.RunId), zap.Error(cancelErr))
		}
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
			CursorCreatedAt: convert.PtrStrToInterface(cursorCreatedAt),
			CursorRunID:     cursorRunID,
			PageSize:        pageSize,
		})
	} else {
		rows, err = s.store.Run.ListRunsPage(ctx, runstore.ListRunsPageParams{
			WorkspaceID:     req.WorkspaceId,
			CursorCreatedAt: convert.PtrStrToInterface(cursorCreatedAt),
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

// timestampPBToDBPtr formats a *timestamppb.Timestamp to *string for DB storage.
func timestampPBToDBPtr(ts *timestamppb.Timestamp) *string {
	if ts == nil {
		return nil
	}
	s := ts.AsTime().UTC().Format(time.RFC3339Nano)
	return &s
}

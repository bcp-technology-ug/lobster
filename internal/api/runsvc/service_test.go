package runsvc_test

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/common"
	runv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/run"
	runstore "github.com/bcp-technology-ug/lobster/gen/sqlc/run"
	"github.com/bcp-technology-ug/lobster/internal/api/convert"
	"github.com/bcp-technology-ug/lobster/internal/api/runsvc"
	"github.com/bcp-technology-ug/lobster/internal/testutil"
)

func TestGetRun_found(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := runsvc.New(st, nil)

	now := convert.NowDB()
	runID := "run-get-001"
	if err := st.Run.CreateRun(ctx, runstore.CreateRunParams{
		RunID:       runID,
		WorkspaceID: "ws-001",
		ProfileName: "default",
		Status:      int64(commonv1.RunStatus_RUN_STATUS_PASSED),
		CreatedAt:   now,
	}); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	resp, err := svc.GetRun(ctx, &runv1.GetRunRequest{RunId: runID})
	if err != nil {
		t.Fatalf("GetRun error: %v", err)
	}
	if resp.Run == nil {
		t.Fatal("expected non-nil run")
	}
	if resp.Run.RunId != runID {
		t.Errorf("RunId: got %q want %q", resp.Run.RunId, runID)
	}
}

func TestGetRun_notFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := runsvc.New(st, nil)

	_, err := svc.GetRun(ctx, &runv1.GetRunRequest{RunId: "nonexistent-run"})
	if err == nil {
		t.Fatal("expected error for missing run")
	}
	st2, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if st2.Code() != codes.NotFound {
		t.Errorf("code: got %v want NotFound", st2.Code())
	}
}

func TestListRuns_empty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := runsvc.New(st, nil)

	resp, err := svc.ListRuns(ctx, &runv1.ListRunsRequest{WorkspaceId: "ws-empty"})
	if err != nil {
		t.Fatalf("ListRuns error: %v", err)
	}
	if len(resp.Runs) != 0 {
		t.Errorf("expected 0 runs, got %d", len(resp.Runs))
	}
}

func TestListRuns_withRuns(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := runsvc.New(st, nil)

	for i := 0; i < 3; i++ {
		now := convert.NowDB()
		time.Sleep(time.Millisecond) // ensure distinct created_at ordering
		if err := st.Run.CreateRun(ctx, runstore.CreateRunParams{
			RunID:       convert.EncodeCursor(now, string(rune('a'+i))),
			WorkspaceID: "ws-list",
			ProfileName: "default",
			Status:      int64(commonv1.RunStatus_RUN_STATUS_PASSED),
			CreatedAt:   now,
		}); err != nil {
			t.Fatalf("CreateRun[%d]: %v", i, err)
		}
	}

	resp, err := svc.ListRuns(ctx, &runv1.ListRunsRequest{WorkspaceId: "ws-list"})
	if err != nil {
		t.Fatalf("ListRuns error: %v", err)
	}
	if len(resp.Runs) != 3 {
		t.Errorf("expected 3 runs, got %d", len(resp.Runs))
	}
}

func TestCancelRun_notFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := runsvc.New(st, nil)

	_, err := svc.CancelRun(ctx, &runv1.CancelRunRequest{RunId: "nonexistent-run"})
	if err == nil {
		t.Fatal("expected error for missing run")
	}
	st2, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if st2.Code() != codes.NotFound {
		t.Errorf("code: got %v want NotFound", st2.Code())
	}
}

func TestCancelRun_alreadyTerminal(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := runsvc.New(st, nil)

	now := convert.NowDB()
	runID := "run-cancel-terminal"
	if err := st.Run.CreateRun(ctx, runstore.CreateRunParams{
		RunID:       runID,
		WorkspaceID: "ws-cancel",
		ProfileName: "default",
		Status:      int64(commonv1.RunStatus_RUN_STATUS_PASSED),
		CreatedAt:   now,
	}); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	resp, err := svc.CancelRun(ctx, &runv1.CancelRunRequest{RunId: runID})
	if err != nil {
		t.Fatalf("CancelRun error: %v", err)
	}
	// Should return current terminal status idempotently.
	if resp.TerminalStatus != commonv1.RunStatus_RUN_STATUS_PASSED {
		t.Errorf("TerminalStatus: got %v want PASSED", resp.TerminalStatus)
	}
}

func TestCancelRun_running(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	st := testutil.OpenStore(t)
	svc := runsvc.New(st, nil)

	now := convert.NowDB()
	runID := "run-cancel-running"
	if err := st.Run.CreateRun(ctx, runstore.CreateRunParams{
		RunID:       runID,
		WorkspaceID: "ws-cancel",
		ProfileName: "default",
		Status:      int64(commonv1.RunStatus_RUN_STATUS_RUNNING),
		CreatedAt:   now,
	}); err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	resp, err := svc.CancelRun(ctx, &runv1.CancelRunRequest{RunId: runID})
	if err != nil {
		t.Fatalf("CancelRun error: %v", err)
	}
	if resp.TerminalStatus != commonv1.RunStatus_RUN_STATUS_CANCELLED {
		t.Errorf("TerminalStatus: got %v want CANCELLED", resp.TerminalStatus)
	}
}

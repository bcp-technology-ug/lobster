package store_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	runstore "github.com/bcp-technology-ug/lobster/gen/sqlc/run"
	"github.com/bcp-technology-ug/lobster/internal/store"
	"github.com/bcp-technology-ug/lobster/internal/testutil"
)

// createTerminalRun inserts a run with a terminal status and a specified
// ended_at timestamp so retention queries can see it.
func createTerminalRun(t *testing.T, st *store.Store, workspaceID, runID string, endedAt time.Time) {
	t.Helper()
	ctx := context.Background()
	createdAt := endedAt.Add(-5 * time.Second).Format(time.RFC3339Nano)
	if err := st.Run.CreateRun(ctx, runstore.CreateRunParams{
		RunID:       runID,
		WorkspaceID: workspaceID,
		ProfileName: "default",
		Status:      3, // RUN_STATUS_PASSED
		CreatedAt:   createdAt,
	}); err != nil {
		t.Fatalf("CreateRun %s: %v", runID, err)
	}
	// Mark as ended so retention age queries can match.
	endedAtStr := endedAt.UTC().Format(time.RFC3339Nano)
	createdAtStr := createdAt
	if err := st.Run.UpdateRunStatus(ctx, runstore.UpdateRunStatusParams{
		Status:    3, // PASSED
		EndedAt:   &endedAtStr,
		StartedAt: &createdAtStr,
		RunID:     runID,
	}); err != nil {
		t.Fatalf("UpdateRunStatus %s: %v", runID, err)
	}
}

func TestPruneRuns_nilStore_noOp(t *testing.T) {
	t.Parallel()
	var s *store.Store
	err := s.PruneRuns(context.Background(), "ws", store.RetentionConfig{MaxRuns: 1})
	if err != nil {
		t.Errorf("PruneRuns on nil store should be a no-op, got: %v", err)
	}
}

func TestPruneRuns_zeroConfig_noOp(t *testing.T) {
	t.Parallel()
	st := testutil.OpenStore(t)
	err := st.PruneRuns(context.Background(), "ws", store.RetentionConfig{})
	if err != nil {
		t.Errorf("PruneRuns with zero config should be a no-op, got: %v", err)
	}
}

func TestPruneRuns_maxRuns_deletesOldest(t *testing.T) {
	t.Parallel()
	st := testutil.OpenStore(t)
	ctx := context.Background()
	ws := "ws-max-runs"
	now := time.Now().UTC()

	// Create 5 terminal runs.
	for i := range 5 {
		runID := fmt.Sprintf("run-%d", i)
		age := now.Add(-time.Duration(i+1) * time.Hour)
		createTerminalRun(t, st, ws, runID, age)
	}

	// Keep only 3.
	if err := st.PruneRuns(ctx, ws, store.RetentionConfig{MaxRuns: 3}); err != nil {
		t.Fatalf("PruneRuns error: %v", err)
	}

	count, err := st.Run.CountRunsByWorkspace(ctx, ws)
	if err != nil {
		t.Fatalf("CountRunsByWorkspace: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 runs after pruning, got %d", count)
	}
}

func TestPruneRuns_maxRuns_noDeleteWhenUnderLimit(t *testing.T) {
	t.Parallel()
	st := testutil.OpenStore(t)
	ctx := context.Background()
	ws := "ws-under-limit"
	now := time.Now().UTC()

	// Create 2 runs, limit is 10 — nothing should be deleted.
	for i := range 2 {
		runID := fmt.Sprintf("under-run-%d", i)
		createTerminalRun(t, st, ws, runID, now.Add(-time.Duration(i+1)*time.Hour))
	}

	if err := st.PruneRuns(ctx, ws, store.RetentionConfig{MaxRuns: 10}); err != nil {
		t.Fatalf("PruneRuns error: %v", err)
	}

	count, err := st.Run.CountRunsByWorkspace(ctx, ws)
	if err != nil {
		t.Fatalf("CountRunsByWorkspace: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 runs (unchanged), got %d", count)
	}
}

func TestPruneRuns_maxAge_deletesOldRuns(t *testing.T) {
	t.Parallel()
	st := testutil.OpenStore(t)
	ctx := context.Background()
	ws := "ws-max-age"
	now := time.Now().UTC()

	// Create one old run (3 days ago) and one recent run (1 hour ago).
	createTerminalRun(t, st, ws, "old-run", now.Add(-72*time.Hour))
	createTerminalRun(t, st, ws, "recent-run", now.Add(-1*time.Hour))

	// Prune runs older than 48 hours.
	if err := st.PruneRuns(ctx, ws, store.RetentionConfig{MaxAge: 48 * time.Hour}); err != nil {
		t.Fatalf("PruneRuns error: %v", err)
	}

	count, err := st.Run.CountRunsByWorkspace(ctx, ws)
	if err != nil {
		t.Fatalf("CountRunsByWorkspace: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 run after age pruning (old run deleted), got %d", count)
	}
}

func TestPruneRuns_maxAge_keepsAllRecentRuns(t *testing.T) {
	t.Parallel()
	st := testutil.OpenStore(t)
	ctx := context.Background()
	ws := "ws-all-recent"
	now := time.Now().UTC()

	// All runs are recent (within 1 hour).
	for i := range 3 {
		runID := fmt.Sprintf("recent-%d", i)
		createTerminalRun(t, st, ws, runID, now.Add(-time.Duration(i+1)*time.Minute))
	}

	if err := st.PruneRuns(ctx, ws, store.RetentionConfig{MaxAge: 24 * time.Hour}); err != nil {
		t.Fatalf("PruneRuns error: %v", err)
	}

	count, err := st.Run.CountRunsByWorkspace(ctx, ws)
	if err != nil {
		t.Fatalf("CountRunsByWorkspace: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 runs (none deleted), got %d", count)
	}
}

func TestPruneRuns_differentWorkspaces_isolatedPruning(t *testing.T) {
	t.Parallel()
	st := testutil.OpenStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// 4 runs in ws-a, 2 runs in ws-b. Prune ws-a to max 2.
	for i := range 4 {
		createTerminalRun(t, st, "ws-a", fmt.Sprintf("a-run-%d", i), now.Add(-time.Duration(i+1)*time.Hour))
	}
	for i := range 2 {
		createTerminalRun(t, st, "ws-b", fmt.Sprintf("b-run-%d", i), now.Add(-time.Duration(i+1)*time.Hour))
	}

	if err := st.PruneRuns(ctx, "ws-a", store.RetentionConfig{MaxRuns: 2}); err != nil {
		t.Fatalf("PruneRuns error: %v", err)
	}

	countA, err := st.Run.CountRunsByWorkspace(ctx, "ws-a")
	if err != nil {
		t.Fatalf("CountRunsByWorkspace ws-a: %v", err)
	}
	if countA != 2 {
		t.Errorf("ws-a: expected 2 runs after pruning, got %d", countA)
	}

	countB, err := st.Run.CountRunsByWorkspace(ctx, "ws-b")
	if err != nil {
		t.Fatalf("CountRunsByWorkspace ws-b: %v", err)
	}
	if countB != 2 {
		t.Errorf("ws-b: expected 2 runs untouched, got %d", countB)
	}
}

func TestRetentionConfig_zeroValuesNoLimits(t *testing.T) {
	t.Parallel()
	cfg := store.RetentionConfig{}
	if cfg.MaxRuns != 0 {
		t.Errorf("MaxRuns: got %d want 0", cfg.MaxRuns)
	}
	if cfg.MaxAge != 0 {
		t.Errorf("MaxAge: got %v want 0", cfg.MaxAge)
	}
}

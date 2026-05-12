package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	integrationstore "github.com/bcp-technology-ug/lobster/gen/sqlc/integrations"
	planstore "github.com/bcp-technology-ug/lobster/gen/sqlc/plan"
	runstore "github.com/bcp-technology-ug/lobster/gen/sqlc/run"
	stackstore "github.com/bcp-technology-ug/lobster/gen/sqlc/stack"
	"github.com/bcp-technology-ug/lobster/internal/store/migrations"
	"github.com/bcp-technology-ug/lobster/internal/store/sqlite"
)

// Config controls SQLite setup and migration behavior.
type Config struct {
	SQLitePath    string
	MigrationsDir string
	MigrationMode migrations.Mode

	JournalMode string
	Synchronous string
	BusyTimeout time.Duration
}

// Store bundles domain query sets over a shared SQLite connection.
type Store struct {
	db *sql.DB

	Run          *runstore.Queries
	Plan         *planstore.Queries
	Stack        *stackstore.Queries
	Integrations *integrationstore.Queries
}

// Open initializes SQLite, applies migration policy, and wires query sets.
func Open(ctx context.Context, cfg Config) (*Store, error) {
	db, err := sqlite.Open(ctx, sqlite.OpenOptions{
		Path:        cfg.SQLitePath,
		JournalMode: cfg.JournalMode,
		Synchronous: cfg.Synchronous,
		BusyTimeout: cfg.BusyTimeout,
	})
	if err != nil {
		return nil, err
	}

	if err := migrations.Ensure(db, cfg.MigrationsDir, cfg.MigrationMode); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	return &Store{
		db:           db,
		Run:          runstore.New(db),
		Plan:         planstore.New(db),
		Stack:        stackstore.New(db),
		Integrations: integrationstore.New(db),
	}, nil
}

// DB exposes the raw database handle for transaction orchestration.
func (s *Store) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.db
}

// Close releases the underlying database resources.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// RetentionConfig defines the limits used by PruneRuns.
type RetentionConfig struct {
	// MaxRuns is the maximum number of terminal runs to retain per workspace.
	// Oldest runs beyond this count are deleted. Zero means no count limit.
	MaxRuns int64

	// MaxAge is the maximum age of terminal runs to retain.
	// Runs older than this are deleted. Zero means no age limit.
	MaxAge time.Duration
}

// PruneRuns deletes terminal runs for workspaceID that exceed the configured
// retention limits. It is best-effort: individual delete errors are collected
// and returned as a combined error but do not stop the pruning of other runs.
func (s *Store) PruneRuns(ctx context.Context, workspaceID string, cfg RetentionConfig) error {
	if s == nil || s.Run == nil {
		return nil
	}
	if cfg.MaxRuns == 0 && cfg.MaxAge == 0 {
		return nil
	}

	var errs []error

	if cfg.MaxRuns > 0 {
		ids, err := s.Run.ListRetentionCandidateRunsByCount(ctx, runstore.ListRetentionCandidateRunsByCountParams{
			WorkspaceID: workspaceID,
			MaxKeepRuns: cfg.MaxRuns,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("list by count: %w", err))
		}
		for _, id := range ids {
			if delErr := s.Run.DeleteRun(ctx, id); delErr != nil {
				errs = append(errs, fmt.Errorf("delete run %s: %w", id, delErr))
			}
		}
	}

	if cfg.MaxAge > 0 {
		cutoff := time.Now().UTC().Add(-cfg.MaxAge).Format(time.RFC3339Nano)
		ids, err := s.Run.ListRetentionCandidateRunsByAge(ctx, runstore.ListRetentionCandidateRunsByAgeParams{
			WorkspaceID: workspaceID,
			CutoffTime:  &cutoff,
			LimitRows:   1000,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("list by age: %w", err))
		}
		for _, id := range ids {
			if delErr := s.Run.DeleteRun(ctx, id); delErr != nil {
				errs = append(errs, fmt.Errorf("delete run %s: %w", id, delErr))
			}
		}
	}

	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return fmt.Errorf("prune runs: %s", strings.Join(msgs, "; "))
	}
	return nil
}

package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	integrationstore "github.com/bcp-technology/lobster/gen/sqlc/integrations"
	planstore "github.com/bcp-technology/lobster/gen/sqlc/plan"
	runstore "github.com/bcp-technology/lobster/gen/sqlc/run"
	stackstore "github.com/bcp-technology/lobster/gen/sqlc/stack"
	"github.com/bcp-technology/lobster/internal/store/migrations"
	"github.com/bcp-technology/lobster/internal/store/sqlite"
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

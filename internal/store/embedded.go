package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	sqlite3driver "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	integrationstore "github.com/bcp-technology-ug/lobster/gen/sqlc/integrations"
	planstore "github.com/bcp-technology-ug/lobster/gen/sqlc/plan"
	runstore "github.com/bcp-technology-ug/lobster/gen/sqlc/run"
	stackstore "github.com/bcp-technology-ug/lobster/gen/sqlc/stack"
	"github.com/bcp-technology-ug/lobster/internal/store/sqlite"
)

// OpenWithMigrationsFS opens a Store applying migrations from an embedded fs.FS
// rather than a filesystem directory path.  This allows the TUI (and any other
// embedded-mode caller) to work regardless of the working directory.
func OpenWithMigrationsFS(ctx context.Context, cfg Config, migrationsFS fs.FS) (*Store, error) {
	db, err := sqlite.Open(ctx, sqlite.OpenOptions{
		Path:        cfg.SQLitePath,
		JournalMode: cfg.JournalMode,
		Synchronous: cfg.Synchronous,
		BusyTimeout: cfg.BusyTimeout,
	})
	if err != nil {
		return nil, err
	}

	if err := ensureWithFS(db, migrationsFS); err != nil {
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

// ensureWithFS applies auto migrations using an embedded fs.FS as the source.
func ensureWithFS(db *sql.DB, migrationsFS fs.FS) error {
	src, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("open embedded migrations source: %w", err)
	}

	driver, err := sqlite3driver.WithInstance(db, &sqlite3driver.Config{NoTxWrap: true})
	if err != nil {
		return fmt.Errorf("create sqlite migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

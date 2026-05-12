package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	defaultJournalMode  = "WAL"
	defaultSynchronous  = "NORMAL"
	defaultBusyTimeout  = 5 * time.Second
	defaultMaxOpenConns = 1
)

// OpenOptions configures SQLite connectivity and runtime pragmas.
type OpenOptions struct {
	Path         string
	JournalMode  string
	Synchronous  string
	BusyTimeout  time.Duration
	MaxOpenConns int
}

// Open returns a configured SQLite handle suitable for Lobster's single-writer model.
func Open(ctx context.Context, opts OpenOptions) (*sql.DB, error) {
	if strings.TrimSpace(opts.Path) == "" {
		return nil, fmt.Errorf("sqlite path is required")
	}

	if err := os.MkdirAll(filepath.Dir(opts.Path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}

	db, err := sql.Open("sqlite3", opts.Path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if opts.MaxOpenConns <= 0 {
		opts.MaxOpenConns = defaultMaxOpenConns
	}
	db.SetMaxOpenConns(opts.MaxOpenConns)
	db.SetMaxIdleConns(opts.MaxOpenConns)

	if opts.BusyTimeout <= 0 {
		opts.BusyTimeout = defaultBusyTimeout
	}
	if strings.TrimSpace(opts.JournalMode) == "" {
		opts.JournalMode = defaultJournalMode
	}
	if strings.TrimSpace(opts.Synchronous) == "" {
		opts.Synchronous = defaultSynchronous
	}

	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = " + opts.JournalMode,
		"PRAGMA synchronous = " + opts.Synchronous,
		"PRAGMA busy_timeout = " + strconv.FormatInt(opts.BusyTimeout.Milliseconds(), 10),
	}
	for _, stmt := range pragmas {
		if _, execErr := db.ExecContext(ctx, stmt); execErr != nil {
			_ = db.Close()
			return nil, fmt.Errorf("apply sqlite pragma %q: %w", stmt, execErr)
		}
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}

	return db, nil
}

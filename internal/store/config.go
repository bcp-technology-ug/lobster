package store

import (
	"fmt"
	"time"

	"github.com/bcp-technology/lobster/internal/store/migrations"
)

const (
	defaultSQLitePath    = ".lobster/lobster.db"
	defaultMigrationsDir = "migrations"
	defaultBusyTimeout   = 5 * time.Second
	defaultJournalMode   = "WAL"
	defaultSynchronous   = "NORMAL"
)

// LoadOptions is the runtime-friendly input shape used to build store config.
type LoadOptions struct {
	SQLitePath    string
	MigrationsDir string
	MigrationMode migrations.Mode
	JournalMode   string
	Synchronous   string
	BusyTimeout   time.Duration
}

// DefaultConfig returns conservative defaults aligned with docs/persistence.md.
func DefaultConfig() Config {
	return Config{
		SQLitePath:    defaultSQLitePath,
		MigrationsDir: defaultMigrationsDir,
		MigrationMode: migrations.ModeAuto,
		BusyTimeout:   defaultBusyTimeout,
		JournalMode:   defaultJournalMode,
		Synchronous:   defaultSynchronous,
	}
}

// ConfigFromOptions applies explicit overrides over default persistence settings.
func ConfigFromOptions(opts LoadOptions) Config {
	cfg := DefaultConfig()

	if opts.SQLitePath != "" {
		cfg.SQLitePath = opts.SQLitePath
	}
	if opts.MigrationsDir != "" {
		cfg.MigrationsDir = opts.MigrationsDir
	}
	if opts.MigrationMode != migrations.ModeUnspecified {
		cfg.MigrationMode = opts.MigrationMode
	}
	if opts.BusyTimeout > 0 {
		cfg.BusyTimeout = opts.BusyTimeout
	}
	if opts.JournalMode != "" {
		cfg.JournalMode = opts.JournalMode
	}
	if opts.Synchronous != "" {
		cfg.Synchronous = opts.Synchronous
	}

	return cfg
}

// ParseMigrationMode converts user-facing values into a migration mode.
func ParseMigrationMode(raw string) (migrations.Mode, error) {
	switch raw {
	case "", "auto":
		return migrations.ModeAuto, nil
	case "external":
		return migrations.ModeExternal, nil
	case "disabled":
		return migrations.ModeDisabled, nil
	default:
		return migrations.ModeUnspecified, fmt.Errorf("unsupported migration mode: %q", raw)
	}
}

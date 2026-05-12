package config

import (
	"testing"
	"time"
)

func TestStoreLoadOptionsFromInput_ExplicitSQLitePathWins(t *testing.T) {
	t.Parallel()

	opts, err := StoreLoadOptionsFromInput(StoreAdapterInput{
		Workspace:     "payments",
		MigrationsDir: "migrations",
		Profile: Profile{
			Compose: ComposeConfig{MigrationMode: "external"},
			Persistence: PersistenceConfig{SQLite: SQLiteConfig{
				Path:        "/var/lib/lobster/custom.db",
				JournalMode: "WAL",
			}},
		},
	})
	if err != nil {
		t.Fatalf("StoreLoadOptionsFromInput returned error: %v", err)
	}
	if opts.SQLitePath != "/var/lib/lobster/custom.db" {
		t.Fatalf("SQLitePath = %q", opts.SQLitePath)
	}
}

func TestStoreLoadOptionsFromInput_DerivesWorkspaceSQLitePath(t *testing.T) {
	t.Parallel()

	opts, err := StoreLoadOptionsFromInput(StoreAdapterInput{
		Workspace:     "payments",
		MigrationsDir: "migrations",
		Profile: Profile{
			Compose: ComposeConfig{MigrationMode: "auto"},
		},
	})
	if err != nil {
		t.Fatalf("StoreLoadOptionsFromInput returned error: %v", err)
	}
	if opts.SQLitePath != ".lobster/workspaces/payments/lobster.db" {
		t.Fatalf("SQLitePath = %q", opts.SQLitePath)
	}
}

func TestStoreLoadOptionsFromInput_AppliesSQLiteFields(t *testing.T) {
	t.Parallel()

	opts, err := StoreLoadOptionsFromInput(StoreAdapterInput{
		MigrationsDir: "db/migrations",
		Profile: Profile{
			Compose: ComposeConfig{MigrationMode: "disabled"},
			Persistence: PersistenceConfig{SQLite: SQLiteConfig{
				JournalMode: "DELETE",
				Synchronous: "FULL",
				BusyTimeout: 2 * time.Second,
			}},
		},
	})
	if err != nil {
		t.Fatalf("StoreLoadOptionsFromInput returned error: %v", err)
	}
	if opts.MigrationsDir != "db/migrations" {
		t.Fatalf("MigrationsDir = %q", opts.MigrationsDir)
	}
	if opts.JournalMode != "DELETE" {
		t.Fatalf("JournalMode = %q", opts.JournalMode)
	}
	if opts.Synchronous != "FULL" {
		t.Fatalf("Synchronous = %q", opts.Synchronous)
	}
	if opts.BusyTimeout != 2*time.Second {
		t.Fatalf("BusyTimeout = %v", opts.BusyTimeout)
	}
}

func TestStoreLoadOptionsFromInput_RejectsInvalidWorkspacePath(t *testing.T) {
	t.Parallel()

	_, err := StoreLoadOptionsFromInput(StoreAdapterInput{
		Workspace: "../escape",
		Profile:   Profile{Compose: ComposeConfig{MigrationMode: "auto"}},
	})
	if err == nil {
		t.Fatalf("expected error for invalid workspace path")
	}
}

func TestStoreLoadOptionsFromInput_RejectsInvalidMigrationMode(t *testing.T) {
	t.Parallel()

	_, err := StoreLoadOptionsFromInput(StoreAdapterInput{
		Profile: Profile{Compose: ComposeConfig{MigrationMode: "invalid-mode"}},
	})
	if err == nil {
		t.Fatalf("expected error for invalid migration mode")
	}
}

func TestStoreConfigFromInput_UsesStoreDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := StoreConfigFromInput(StoreAdapterInput{
		Profile: Profile{},
	})
	if err != nil {
		t.Fatalf("StoreConfigFromInput returned error: %v", err)
	}

	if cfg.SQLitePath != ".lobster/lobster.db" {
		t.Fatalf("SQLitePath = %q", cfg.SQLitePath)
	}
	if cfg.MigrationsDir != "migrations" {
		t.Fatalf("MigrationsDir = %q", cfg.MigrationsDir)
	}
}

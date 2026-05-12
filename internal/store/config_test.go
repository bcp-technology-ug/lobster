package store

import (
	"testing"
	"time"

	"github.com/bcp-technology-ug/lobster/internal/store/migrations"
)

func TestParseMigrationMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    migrations.Mode
		wantErr bool
	}{
		{name: "default auto", raw: "", want: migrations.ModeAuto},
		{name: "explicit auto", raw: "auto", want: migrations.ModeAuto},
		{name: "external", raw: "external", want: migrations.ModeExternal},
		{name: "disabled", raw: "disabled", want: migrations.ModeDisabled},
		{name: "invalid", raw: "nope", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseMigrationMode(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for mode %q", tc.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMigrationMode(%q) returned unexpected error: %v", tc.raw, err)
			}
			if got != tc.want {
				t.Fatalf("ParseMigrationMode(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestConfigFromOptionsAppliesOverrides(t *testing.T) {
	t.Parallel()

	cfg := ConfigFromOptions(LoadOptions{
		SQLitePath:    "./tmp/lobster.db",
		MigrationsDir: "./db/migrations",
		MigrationMode: migrations.ModeExternal,
		JournalMode:   "DELETE",
		Synchronous:   "FULL",
		BusyTimeout:   1500 * time.Millisecond,
	})

	if cfg.SQLitePath != "./tmp/lobster.db" {
		t.Fatalf("SQLitePath = %q", cfg.SQLitePath)
	}
	if cfg.MigrationsDir != "./db/migrations" {
		t.Fatalf("MigrationsDir = %q", cfg.MigrationsDir)
	}
	if cfg.MigrationMode != migrations.ModeExternal {
		t.Fatalf("MigrationMode = %v", cfg.MigrationMode)
	}
	if cfg.JournalMode != "DELETE" {
		t.Fatalf("JournalMode = %q", cfg.JournalMode)
	}
	if cfg.Synchronous != "FULL" {
		t.Fatalf("Synchronous = %q", cfg.Synchronous)
	}
	if cfg.BusyTimeout != 1500*time.Millisecond {
		t.Fatalf("BusyTimeout = %v", cfg.BusyTimeout)
	}
}

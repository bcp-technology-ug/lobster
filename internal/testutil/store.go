// Package testutil provides shared helpers for package-level tests.
package testutil

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bcp-technology-ug/lobster/internal/store"
	"github.com/bcp-technology-ug/lobster/internal/store/migrations"
	lobstermigrations "github.com/bcp-technology-ug/lobster/migrations"
)

// OpenStore opens a file-based SQLite store with auto-migrations for the
// given test. The store and its temp directory are automatically cleaned up
// when the test completes.
func OpenStore(t *testing.T) *store.Store {
	t.Helper()
	// t.TempDir() creates a unique directory per test that is guaranteed to
	// exist and is removed automatically on test completion.
	dbPath := filepath.Join(t.TempDir(), "lobster_test.db")

	st, err := store.OpenWithMigrationsFS(context.Background(), store.Config{
		SQLitePath:    dbPath,
		MigrationMode: migrations.ModeAuto,
	}, lobstermigrations.FS)
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})
	return st
}

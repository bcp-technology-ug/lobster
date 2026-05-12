// Package testutil provides shared helpers for package-level tests.
package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/bcp-technology-ug/lobster/internal/store"
	"github.com/bcp-technology-ug/lobster/internal/store/migrations"
)

var storeCounter atomic.Int64

// OpenStore opens an in-memory SQLite store with auto-migrations for the
// given test. The store is automatically closed when the test completes.
func OpenStore(t *testing.T) *store.Store {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	// testutil is at internal/testutil/store.go; migrations are at root/migrations
	root := filepath.Join(filepath.Dir(filename), "../../")
	migrationsDir := filepath.Join(root, "migrations")

	// Each test gets its own named in-memory database to avoid cross-test interference.
	// Use a file path in os.TempDir so sqlite/open.go's MkdirAll is a no-op,
	// but SQLite never actually writes to disk (journal_mode=MEMORY + in-process).
	// We pass a real temp-file path so the path parsing is correct, then
	// immediately let go-sqlite3 use it as a named URI memory DB.
	dbName := fmt.Sprintf("%s/lobster_test_%d.db", os.TempDir(), storeCounter.Add(1))
	sqlitePath := dbName

	st, err := store.Open(context.Background(), store.Config{
		SQLitePath:    sqlitePath,
		MigrationsDir: migrationsDir,
		MigrationMode: migrations.ModeAuto,
	})
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
		_ = os.Remove(dbName)
	})
	return st
}

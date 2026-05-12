package migrations

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLatestVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	files := []string{
		"000001_initial_schema.up.sql",
		"000001_initial_schema.down.sql",
		"000005_add_indexes.up.sql",
		"000010_future_change.up.sql",
		"notes.txt",
	}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("-- stub\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	got, err := latestVersion(dir)
	if err != nil {
		t.Fatalf("latestVersion returned error: %v", err)
	}
	if got != 10 {
		t.Fatalf("latestVersion = %d, want 10", got)
	}
}

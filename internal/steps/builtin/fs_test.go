package builtin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bcp-technology-ug/lobster/internal/steps"
)

// NOTE: Do NOT run fs tests in parallel — stepNewTempDir calls os.Chdir which
// affects the entire process working directory.

func withTempDir(t *testing.T, fn func(dir string)) {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
	fn(dir)
}

func TestFS_fileExists_missing(t *testing.T) {
	withTempDir(t, func(dir string) {
		ctx := steps.NewScenarioContext("", nil, nil)
		if err := stepFileExists(ctx, "missing.txt"); err == nil {
			t.Error("expected error for missing file")
		}
	})
}

func TestFS_fileExists_found(t *testing.T) {
	withTempDir(t, func(dir string) {
		path := filepath.Join(dir, "present.txt")
		if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
			t.Fatal(err)
		}
		ctx := steps.NewScenarioContext("", nil, nil)
		if err := stepFileExists(ctx, path); err != nil {
			t.Fatalf("stepFileExists: %v", err)
		}
	})
}

func TestFS_fileNotExists_missing(t *testing.T) {
	withTempDir(t, func(dir string) {
		ctx := steps.NewScenarioContext("", nil, nil)
		if err := stepFileNotExists(ctx, "missing.txt"); err != nil {
			t.Fatalf("stepFileNotExists: %v", err)
		}
	})
}

func TestFS_fileNotExists_exists(t *testing.T) {
	withTempDir(t, func(dir string) {
		path := filepath.Join(dir, "present.txt")
		if err := os.WriteFile(path, []byte("hi"), 0o644); err != nil {
			t.Fatal(err)
		}
		ctx := steps.NewScenarioContext("", nil, nil)
		if err := stepFileNotExists(ctx, path); err == nil {
			t.Error("expected error when file exists")
		}
	})
}

func TestFS_createFileInline(t *testing.T) {
	withTempDir(t, func(dir string) {
		path := filepath.Join(dir, "inline.txt")
		if err := stepCreateFileInline(nil, path, "my content"); err != nil {
			t.Fatalf("stepCreateFileInline: %v", err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(data) != "my content" {
			t.Errorf("content: got %q want %q", string(data), "my content")
		}
	})
}

func TestFS_fileContains_found(t *testing.T) {
	withTempDir(t, func(dir string) {
		path := filepath.Join(dir, "check.txt")
		if err := os.WriteFile(path, []byte("the quick brown fox"), 0o644); err != nil {
			t.Fatal(err)
		}
		ctx := steps.NewScenarioContext("", nil, nil)
		if err := stepFileContains(ctx, path, "brown"); err != nil {
			t.Fatalf("stepFileContains: %v", err)
		}
	})
}

func TestFS_fileContains_notFound(t *testing.T) {
	withTempDir(t, func(dir string) {
		path := filepath.Join(dir, "check.txt")
		if err := os.WriteFile(path, []byte("the quick brown fox"), 0o644); err != nil {
			t.Fatal(err)
		}
		ctx := steps.NewScenarioContext("", nil, nil)
		if err := stepFileContains(ctx, path, "lobster"); err == nil {
			t.Error("expected error when content not found")
		}
	})
}

func TestFS_dirExists(t *testing.T) {
	withTempDir(t, func(dir string) {
		sub := filepath.Join(dir, "subdir")
		if err := os.Mkdir(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		ctx := steps.NewScenarioContext("", nil, nil)
		if err := stepDirExists(ctx, sub); err != nil {
			t.Fatalf("stepDirExists: %v", err)
		}
	})
}

func TestFS_dirNotExists(t *testing.T) {
	withTempDir(t, func(dir string) {
		ctx := steps.NewScenarioContext("", nil, nil)
		if err := stepDirNotExists(ctx, filepath.Join(dir, "nosuch")); err != nil {
			t.Fatalf("stepDirNotExists: %v", err)
		}
	})
}

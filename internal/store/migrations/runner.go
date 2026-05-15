package migrations

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	sqlite3driver "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Mode controls how schema migrations are enforced at runtime.
type Mode int

const (
	ModeUnspecified Mode = iota
	ModeAuto
	ModeExternal
	ModeDisabled
)

var migrationFileRe = regexp.MustCompile(`^([0-9]+)_.+\.up\.sql$`)

// Ensure enforces configured migration behaviour for the current runtime mode.
func Ensure(db *sql.DB, migrationsDir string, mode Mode) error {
	if mode == ModeDisabled {
		return nil
	}
	if migrationsDir == "" {
		return fmt.Errorf("migrations directory is required")
	}

	absDir, err := filepath.Abs(migrationsDir)
	if err != nil {
		return fmt.Errorf("resolve migrations directory: %w", err)
	}

	sourceURL := "file://" + filepath.ToSlash(absDir)
	driver, err := sqlite3driver.WithInstance(db, &sqlite3driver.Config{NoTxWrap: true})
	if err != nil {
		return fmt.Errorf("create sqlite migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(sourceURL, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	// Note: m.Close() is intentionally not called here because it would also
	// close the underlying *sql.DB (via the sqlite3 driver), invalidating the
	// caller's connection pool. The source file handles are small and will be
	// reclaimed on process exit or GC.

	switch mode {
	case ModeAuto:
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("apply migrations: %w", err)
		}
	case ModeExternal:
		return validateExternalMode(m, absDir)
	default:
		return fmt.Errorf("unsupported migration mode: %v", mode)
	}

	return nil
}

func validateExternalMode(m *migrate.Migrate, migrationsDir string) error {
	latest, err := latestVersion(migrationsDir)
	if err != nil {
		return err
	}

	current, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		if latest > 0 {
			return fmt.Errorf("database schema is uninitialized; expected migration version %d", latest)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	if dirty {
		return fmt.Errorf("database migration state is dirty at version %d", current)
	}
	if current < latest {
		return fmt.Errorf("database schema version %d is older than required %d (run migrations or use auto mode)", current, latest)
	}
	if current > latest {
		return fmt.Errorf("database schema version %d is newer than binary-supported %d", current, latest)
	}

	return nil
}

func latestVersion(migrationsDir string) (uint, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return 0, fmt.Errorf("read migrations directory: %w", err)
	}

	var max uint
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		version, ok := parseVersion(entry)
		if !ok {
			continue
		}
		if version > max {
			max = version
		}
	}

	return max, nil
}

func parseVersion(entry fs.DirEntry) (uint, bool) {
	matches := migrationFileRe.FindStringSubmatch(entry.Name())
	if len(matches) != 2 {
		return 0, false
	}

	n, err := strconv.ParseUint(matches[1], 10, 32)
	if err != nil {
		return 0, false
	}

	return uint(n), true
}

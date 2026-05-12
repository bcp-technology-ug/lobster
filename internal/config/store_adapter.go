package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bcp-technology/lobster/internal/store"
)

const (
	workspaceDBRoot      = ".lobster/workspaces"
	workspaceDBName      = "lobster.db"
	defaultSQLitePath    = ".lobster/lobster.db"
	defaultMigrationsDir = "migrations"
)

// StoreLoadOptionsFromInput maps profile fields to store load options.
func StoreLoadOptionsFromInput(input StoreAdapterInput) (store.LoadOptions, error) {
	mode, err := store.ParseMigrationMode(strings.TrimSpace(input.Profile.Compose.MigrationMode))
	if err != nil {
		return store.LoadOptions{}, err
	}

	opts := store.LoadOptions{
		SQLitePath:    strings.TrimSpace(input.Profile.Persistence.SQLite.Path),
		MigrationsDir: strings.TrimSpace(input.MigrationsDir),
		MigrationMode: mode,
		JournalMode:   strings.TrimSpace(input.Profile.Persistence.SQLite.JournalMode),
		Synchronous:   strings.TrimSpace(input.Profile.Persistence.SQLite.Synchronous),
		BusyTimeout:   input.Profile.Persistence.SQLite.BusyTimeout,
	}

	if opts.SQLitePath == "" {
		workspacePath, err := deriveWorkspaceSQLitePath(strings.TrimSpace(input.Workspace))
		if err != nil {
			return store.LoadOptions{}, err
		}
		if workspacePath != "" {
			opts.SQLitePath = workspacePath
		} else {
			opts.SQLitePath = defaultSQLitePath
		}
	}

	if opts.MigrationsDir == "" {
		opts.MigrationsDir = defaultMigrationsDir
	}

	return opts, nil
}

// StoreConfigFromInput produces a fully-defaulted store configuration.
func StoreConfigFromInput(input StoreAdapterInput) (store.Config, error) {
	opts, err := StoreLoadOptionsFromInput(input)
	if err != nil {
		return store.Config{}, err
	}
	return store.ConfigFromOptions(opts), nil
}

func deriveWorkspaceSQLitePath(workspace string) (string, error) {
	if workspace == "" {
		return "", nil
	}

	cleaned := filepath.Clean(workspace)
	switch {
	case cleaned == ".":
		return "", nil
	case strings.HasPrefix(cleaned, ".."):
		return "", fmt.Errorf("workspace path must not traverse parent directories: %q", workspace)
	case filepath.IsAbs(cleaned):
		return "", fmt.Errorf("workspace path must be relative: %q", workspace)
	}

	return filepath.Join(workspaceDBRoot, cleaned, workspaceDBName), nil
}

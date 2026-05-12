package config

import "time"

// Profile captures the subset of configuration needed for persistence wiring.
type Profile struct {
	Name        string
	Compose     ComposeConfig
	Persistence PersistenceConfig
}

// ComposeConfig controls compose-level execution options relevant to persistence.
type ComposeConfig struct {
	MigrationMode string
}

// PersistenceConfig controls datastore and retention settings.
type PersistenceConfig struct {
	SQLite SQLiteConfig
}

// SQLiteConfig controls SQLite path and runtime pragmas.
type SQLiteConfig struct {
	Path        string
	JournalMode string
	Synchronous string
	BusyTimeout time.Duration
}

// StoreAdapterInput defines context required to build store configuration.
type StoreAdapterInput struct {
	Profile       Profile
	Workspace     string
	MigrationsDir string
}

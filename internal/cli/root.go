package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/bcp-technology-ug/lobster/internal/config"
	"github.com/bcp-technology-ug/lobster/internal/store"
	"github.com/bcp-technology-ug/lobster/internal/ui"
)

// Version is set at build time via -ldflags "-X github.com/bcp-technology-ug/lobster/internal/cli.Version=<tag>".
// It falls back to "dev" for local/unversioned builds.
var Version = "dev"

// ExitError signals a non-zero exit without printing an additional error panel.
// Use this when a command has already rendered its own diagnostic output.
type ExitError struct{ Code int }

func (e *ExitError) Error() string { return "" }

// Exit code constants for differentiated failure categories.
const (
	ExitScenarioFailure = 1 // One or more scenarios failed.
	ExitConfigError     = 2 // Invalid flags, config file, or validation error.
	ExitOrchestration   = 3 // Stack orchestration (Docker Compose) error.
	ExitRuntimeError    = 4 // Internal runner or infrastructure error.
)

// NewRootCommand builds the lobster CLI command tree.
func NewRootCommand() *cobra.Command {
	v := viper.New()
	var cfgFile string

	root := &cobra.Command{
		Use:     "lobster",
		Short:   "CLI-first end-to-end BDD runner",
		Version: Version,
		Long: ui.LogoBanner() + "\n\n" +
			ui.StyleMuted.Render("Contract-driven BDD end-to-end testing against real infrastructure.") + "\n" +
			ui.StyleMuted.Render("Docs: https://github.com/bcp-technology-ug/lobster"),
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return initViper(v, cfgFile)
		},
	}

	root.SetVersionTemplate(ui.LogoVersion("lobster", Version) + "\n")

	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (defaults to ./lobster.yaml)")

	root.AddCommand(newRunCommand(v))
	root.AddCommand(newConfigCommand(v))
	root.AddCommand(newInitCommand(v))
	root.AddCommand(newPlanCommand(v))
	root.AddCommand(newValidateCommand(v))
	root.AddCommand(newLintCommand(v))
	root.AddCommand(newRunsCommand(v))
	root.AddCommand(newPlansCommand(v))
	root.AddCommand(newStackCommand(v))
	root.AddCommand(newIntegrationsCommand(v))
	root.AddCommand(newAdminCommand(v))
	root.AddCommand(newDoctorCommand(v))
	root.AddCommand(newTUICommand(v))
	root.AddCommand(newCoverageCommand(v))
	root.AddCommand(newStepsCommand())
	root.AddCommand(newMCPCommand())

	return root
}

func addPersistenceFlags(fs *pflag.FlagSet) {
	fs.String("workspace", "", "workspace selection for monorepo isolation")
	fs.String("sqlite-path", "", "explicit SQLite path override")
	fs.String("migrations-dir", "migrations", "directory containing SQL migrations")
	fs.String("migration-mode", "auto", "migration mode: auto|external|disabled")
	fs.String("journal-mode", "", "SQLite journal mode override")
	fs.String("synchronous", "", "SQLite synchronous pragma override")
	fs.Duration("busy-timeout", 0, "SQLite busy timeout (for example: 5s)")
}

func buildStoreConfigFromInputs(cmd *cobra.Command, v *viper.Viper) (store.Config, error) {
	migrationMode := valueString(cmd, v, "compose.migrations.mode", "migration-mode")
	if _, err := store.ParseMigrationMode(migrationMode); err != nil {
		return store.Config{}, err
	}

	return config.StoreConfigFromInput(config.StoreAdapterInput{
		Workspace:     valueString(cmd, v, "workspace.selected", "workspace"),
		MigrationsDir: valueString(cmd, v, "persistence.migrations.dir", "migrations-dir"),
		Profile: config.Profile{
			Compose: config.ComposeConfig{MigrationMode: migrationMode},
			Persistence: config.PersistenceConfig{SQLite: config.SQLiteConfig{
				Path:        valueString(cmd, v, "persistence.sqlite.path", "sqlite-path"),
				JournalMode: valueString(cmd, v, "persistence.sqlite.journal_mode", "journal-mode"),
				Synchronous: valueString(cmd, v, "persistence.sqlite.synchronous", "synchronous"),
				BusyTimeout: valueDuration(cmd, v, "persistence.sqlite.busy_timeout", "busy-timeout"),
			}},
		},
	})
}

func valueString(cmd *cobra.Command, v *viper.Viper, key, flagName string) string {
	if f := cmd.Flags().Lookup(flagName); f != nil && f.Changed {
		return strings.TrimSpace(f.Value.String())
	}
	return strings.TrimSpace(v.GetString(key))
}

func valueDuration(cmd *cobra.Command, v *viper.Viper, key, flagName string) time.Duration {
	if f := cmd.Flags().Lookup(flagName); f != nil && f.Changed {
		d, _ := cmd.Flags().GetDuration(flagName)
		return d
	}
	return v.GetDuration(key)
}

func initViper(v *viper.Viper, cfgFile string) error {
	v.SetEnvPrefix("LOBSTER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	if strings.TrimSpace(cfgFile) != "" {
		v.SetConfigFile(cfgFile)
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("read config file: %w", err)
		}
		return nil
	}

	v.SetConfigName("lobster")
	v.AddConfigPath(".")
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			return nil
		}
		return fmt.Errorf("read config file: %w", err)
	}
	return nil
}

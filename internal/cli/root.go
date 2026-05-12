package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bcp-technology/lobster/internal/config"
	"github.com/bcp-technology/lobster/internal/store"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// NewRootCommand builds the lobster CLI command tree.
func NewRootCommand() *cobra.Command {
	v := viper.New()
	var cfgFile string

	root := &cobra.Command{
		Use:   "lobster",
		Short: "CLI-first end-to-end BDD runner",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return initViper(v, cfgFile)
		},
	}

	root.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (defaults to ./lobster.yaml)")

	root.AddCommand(newRunCommand(v))
	root.AddCommand(newConfigCommand(v))
	root.AddCommand(newNotImplementedCommand("init"))
	root.AddCommand(newNotImplementedCommand("plan"))
	root.AddCommand(newNotImplementedCommand("validate"))
	root.AddCommand(newNotImplementedCommand("lint"))

	return root
}

func newRunCommand(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute scenarios against configured stack",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			storeCfg, err := buildStoreConfigFromInputs(cmd, v)
			if err != nil {
				return err
			}

			st, err := store.Open(ctx, storeCfg)
			if err != nil {
				return fmt.Errorf("initialize store: %w", err)
			}
			defer func() { _ = st.Close() }()

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "run bootstrap ready (sqlite=%s)\n", storeCfg.SQLitePath)
			return nil
		},
	}

	addPersistenceFlags(cmd.Flags())
	return cmd
}

func newConfigCommand(v *viper.Viper) *cobra.Command {
	var printJSON bool
	var validate bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect or validate effective configuration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			storeCfg, err := buildStoreConfigFromInputs(cmd, v)
			if err != nil {
				return err
			}

			if printJSON {
				payload, err := json.MarshalIndent(map[string]any{
					"workspace":      valueString(cmd, v, "workspace.selected", "workspace"),
					"sqlite_path":    storeCfg.SQLitePath,
					"migrations_dir": storeCfg.MigrationsDir,
					"migration_mode": valueString(cmd, v, "compose.migrations.mode", "migration-mode"),
					"journal_mode":   storeCfg.JournalMode,
					"synchronous":    storeCfg.Synchronous,
					"busy_timeout":   storeCfg.BusyTimeout.String(),
				}, "", "  ")
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(payload))
			}

			if validate {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				st, err := store.Open(ctx, storeCfg)
				if err != nil {
					return fmt.Errorf("validate persistence config: %w", err)
				}
				defer func() { _ = st.Close() }()
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "configuration validation ok")
			}

			if !printJSON && !validate {
				return errors.New("no action requested; use --print or --validate")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&printJSON, "print", false, "print effective configuration as JSON")
	cmd.Flags().BoolVar(&validate, "validate", false, "validate effective configuration and persistence wiring")
	addPersistenceFlags(cmd.Flags())
	return cmd
}

func newNotImplementedCommand(use string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: "Reserved command surface",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("%s command is not implemented yet", use)
		},
	}
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
	v.SetConfigType("yaml")
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

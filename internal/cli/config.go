package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/bcp-technology-ug/lobster/internal/store"
	"github.com/bcp-technology-ug/lobster/internal/ui"
	lobstermigrations "github.com/bcp-technology-ug/lobster/migrations"
)

func newConfigCommand(v *viper.Viper) *cobra.Command {
	var format string
	var validate bool
	var printCfg bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect or validate effective configuration",
		Long:  "Display the effective configuration resolved from flags, environment variables, and the config file.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = printCfg // --print is the default; flag is an explicit alias
			storeCfg, err := buildStoreConfigFromInputs(cmd, v)
			if err != nil {
				return err
			}

			// JSON format: machine-readable output for CI/scripting.
			if strings.EqualFold(format, "json") {
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
				return nil
			}

			// Default: styled multi-section table.
			workspace := valueString(cmd, v, "workspace.selected", "workspace")
			if workspace == "" {
				workspace = ui.StyleMuted.Render("(root)")
			}

			migrationMode := valueString(cmd, v, "compose.migrations.mode", "migration-mode")
			if migrationMode == "" {
				migrationMode = "auto"
			}

			journalMode := storeCfg.JournalMode
			if journalMode == "" {
				journalMode = ui.StyleMuted.Render("(default)")
			}

			synchronous := storeCfg.Synchronous
			if synchronous == "" {
				synchronous = ui.StyleMuted.Render("(default)")
			}

			busyTimeout := storeCfg.BusyTimeout.String()
			if storeCfg.BusyTimeout == 0 {
				busyTimeout = ui.StyleMuted.Render("(default)")
			}

			sections := []ui.Section{
				{
					Title: "Workspace",
					Rows: [][2]string{
						{"Selected", workspace},
						{"SQLite path", storeCfg.SQLitePath},
						{"Migrations dir", storeCfg.MigrationsDir},
					},
				},
				{
					Title: "Persistence",
					Rows: [][2]string{
						{"Journal mode", journalMode},
						{"Synchronous", synchronous},
						{"Busy timeout", busyTimeout},
					},
				},
				{
					Title: "Compose",
					Rows: [][2]string{
						{"Migration mode", migrationMode},
					},
				},
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderSectionedTable(sections))

			// --validate: also attempt to open the store.
			if validate {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				st, err := store.OpenWithMigrationsFS(ctx, storeCfg, lobstermigrations.FS)
				if err != nil {
					_, _ = fmt.Fprint(cmd.OutOrStdout(),
						ui.RenderError("Persistence validation failed", err.Error(),
							"Check that the SQLite path is writable and migrations directory exists.",
							"https://github.com/bcp-technology-ug/lobster/blob/main/docs/persistence.md"),
					)
					return &ExitError{Code: 1}
				}
				defer func() { _ = st.Close() }()
				_, _ = fmt.Fprint(cmd.OutOrStdout(),
					ui.RenderSuccess("Configuration valid", "Persistence wiring verified successfully."),
				)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format: text|json")
	cmd.Flags().BoolVar(&validate, "validate", false, "validate persistence wiring by opening the SQLite store")
	cmd.Flags().BoolVar(&printCfg, "print", false, "print effective configuration (same as default behaviour)")
	addPersistenceFlags(cmd.Flags())
	return cmd
}

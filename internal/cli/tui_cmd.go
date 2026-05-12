package cli

import (
	"context"
	"fmt"

	"github.com/bcp-technology/lobster/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newTUICommand creates the `lobster tui` command, which opens the full tabbed
// lobby TUI connected to a running lobsterd daemon.
func newTUICommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive TUI connected to the daemon",
		Long: `Launch the full tabbed TUI (Live Runs / History / Stack / Admin).

Requires a running lobsterd daemon reachable via --executor-addr.

  lobster tui --executor-addr localhost:9443
`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			workspace, _ := cmd.Flags().GetString("workspace")

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError(
					"Daemon connection failed",
					err.Error(),
					"Ensure lobsterd is running and --executor-addr is correct.",
					"",
				))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			m := ui.NewLobbyModel(conn, workspace)
			p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("TUI error: %w", err)
			}
			return nil
		},
	}

	addDaemonPersistentFlags(cmd)
	cmd.Flags().String("workspace", "", "workspace ID filter")
	return cmd
}

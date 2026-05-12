package cli

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/bcp-technology/lobster/internal/api"
	"github.com/bcp-technology/lobster/internal/api/middleware"
	"github.com/bcp-technology/lobster/internal/store"
	"github.com/bcp-technology/lobster/internal/ui"
	lobstermigrations "github.com/bcp-technology/lobster/migrations"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// newTUICommand creates the `lobster tui` command, which opens the full tabbed
// lobby TUI. When --executor-addr is omitted it starts an embedded in-process
// gRPC server backed by the local SQLite store so no daemon is required.
func newTUICommand(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive TUI (local or connected to daemon)",
		Long: `Launch the full tabbed TUI (Live Runs / History / Stack / Admin).

Without --executor-addr the TUI runs locally against the on-disk store:

  lobster tui

To connect to a running lobsterd daemon instead:

  lobster tui --executor-addr localhost:9443
`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			workspace, _ := cmd.Flags().GetString("workspace")
			// Fall back to config-file / env workspace when flag not explicitly set.
			if workspace == "" {
				workspace = strings.TrimSpace(v.GetString("workspace.selected"))
			}
			addr := strings.TrimSpace(cmd.Flag("executor-addr").Value.String())

			var conn *grpc.ClientConn
			if addr != "" {
				// Connect to an existing daemon.
				var err error
				conn, err = daemonConnFromCmd(ctx, cmd)
				if err != nil {
					_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError(
						"Daemon connection failed",
						err.Error(),
						"Ensure lobsterd is running and --executor-addr is correct.",
						"",
					))
					return &ExitError{Code: ExitOrchestration}
				}
			} else {
				// No daemon address — start an embedded in-process server.
				var err error
				conn, err = startEmbeddedTUIServer(ctx)
				if err != nil {
					_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError(
						"Local store unavailable",
						err.Error(),
						"Check that .lobster/lobster.db is accessible.",
						"",
					))
					return &ExitError{Code: ExitOrchestration}
				}
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

// startEmbeddedTUIServer opens the local SQLite store, builds a minimal gRPC
// server on a random loopback port, and returns a client connection to it.
// The server runs for the lifetime of ctx.
func startEmbeddedTUIServer(ctx context.Context) (*grpc.ClientConn, error) {
	st, err := store.OpenWithMigrationsFS(ctx, store.DefaultConfig(), lobstermigrations.FS)
	if err != nil {
		return nil, fmt.Errorf("open local store: %w", err)
	}

	srv, err := api.Build(st, api.Config{
		Auth: middleware.AuthConfig{AllowInsecureLocal: true, ExplicitLocalMode: true},
	}, api.Services{})
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("build embedded server: %w", err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("listen on loopback: %w", err)
	}

	go func() {
		_ = api.ServeGRPC(ctx, srv.GRPCServer, lis)
		_ = st.Close()
	}()

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to embedded server: %w", err)
	}
	return conn, nil
}

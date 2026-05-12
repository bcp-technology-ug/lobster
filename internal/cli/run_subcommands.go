package cli

import (
	"context"
	"fmt"
	"strings"

	commonv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/common"
	runv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/run"
	"github.com/bcp-technology-ug/lobster/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

// newRunWatchCommand creates `lobster run watch`.
// It streams live events for an async run from the daemon.
func newRunWatchCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Stream live events for an async run",
		Long:  "Connect to the daemon and stream progress events for a previously submitted async run.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			runID, _ := cmd.Flags().GetString("run-id")
			if strings.TrimSpace(runID) == "" {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Missing flag",
					"--run-id is required", "Example: lobster run watch --run-id <id>", ""))
				return &ExitError{Code: ExitConfigError}
			}

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
					err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			client := runv1.NewRunServiceClient(conn)
			stream, err := client.StreamRunEvents(ctx, &runv1.StreamRunEventsRequest{RunId: runID})
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Stream failed",
					err.Error(), "", ""))
				return &ExitError{Code: exitCodeForRunError(err)}
			}

			// Use TUI WatchModel when attached to an interactive terminal.
			if ui.IsInteractive() {
				m := ui.NewWatchModel(runID, stream)
				p := tea.NewProgram(m, tea.WithAltScreen())
				finalModel, runErr := p.Run()
				if runErr != nil {
					return fmt.Errorf("TUI error: %w", runErr)
				}
				if wm, ok := finalModel.(ui.WatchModel); ok && wm.ExitCode() != 0 {
					return &ExitError{Code: ExitScenarioFailure}
				}
				return nil
			}

			// Non-interactive fallback: plain text stream.
			var failed bool
			for {
				ev, recvErr := stream.Recv()
				if recvErr != nil {
					break
				}
				switch p := ev.GetPayload().(type) {
				case *runv1.RunEvent_RunStatus:
					_, _ = fmt.Fprintln(cmd.OutOrStdout(),
						ui.StyleMuted.Render("status: "+p.RunStatus.String()))
				case *runv1.RunEvent_ScenarioResult:
					sc := p.ScenarioResult
					if sc.GetStatus() == commonv1.ScenarioStatus_SCENARIO_STATUS_FAILED {
						failed = true
						_, _ = fmt.Fprintln(cmd.OutOrStdout(),
							ui.StyleError.Render(ui.IconCross+" "+sc.GetScenarioId()))
					} else {
						_, _ = fmt.Fprintln(cmd.OutOrStdout(),
							ui.StyleSuccess.Render(ui.IconCheck+" "+sc.GetScenarioId()))
					}
				case *runv1.RunEvent_Summary:
					s := p.Summary
					line := fmt.Sprintf("total=%d  passed=%d  failed=%d",
						s.GetTotalScenarios(), s.GetPassedScenarios(), s.GetFailedScenarios())
					if s.GetFailedScenarios() > 0 {
						_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleError.Render(ui.IconCross+"  "+line))
					} else {
						_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleSuccess.Render(ui.IconCheck+"  "+line))
					}
				}
				if ev.GetTerminal() {
					break
				}
			}

			if failed {
				return &ExitError{Code: ExitScenarioFailure}
			}
			return nil
		},
	}

	cmd.Flags().String("run-id", "", "run ID to watch (required)")
	return cmd
}

// newRunStatusCommand creates `lobster run status`.
func newRunStatusCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the status of a run",
		Long:  "Fetch and display the current status and summary of a run from the daemon.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			runID, _ := cmd.Flags().GetString("run-id")
			if strings.TrimSpace(runID) == "" {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Missing flag",
					"--run-id is required", "Example: lobster run status --run-id <id>", ""))
				return &ExitError{Code: ExitConfigError}
			}

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
					err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			client := runv1.NewRunServiceClient(conn)
			resp, err := client.GetRun(ctx, &runv1.GetRunRequest{RunId: runID})
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Get run failed",
					err.Error(), "", ""))
				return &ExitError{Code: exitCodeForRunError(err)}
			}

			run := resp.GetRun()
			if run == nil {
				_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderWarning("Not found",
					fmt.Sprintf("No run found with ID %q", runID)))
				return nil
			}

			summary := run.GetSummary()
			rows := [][2]string{
				{"Run ID", run.GetRunId()},
				{"Status", run.GetStatus().String()},
				{"Profile", run.GetProfileName()},
			}
			if summary != nil {
				rows = append(rows,
					[2]string{"Total", fmt.Sprintf("%d", summary.GetTotalScenarios())},
					[2]string{"Passed", fmt.Sprintf("%d", summary.GetPassedScenarios())},
					[2]string{"Failed", fmt.Sprintf("%d", summary.GetFailedScenarios())},
					[2]string{"Skipped", fmt.Sprintf("%d", summary.GetSkippedScenarios())},
				)
				if d := summary.GetDuration(); d != nil {
					rows = append(rows, [2]string{"Duration", d.AsDuration().String()})
				}
			}
			if run.GetCreatedAt() != nil {
				rows = append(rows, [2]string{"Created", run.GetCreatedAt().AsTime().Format("2006-01-02 15:04:05")})
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable("Run status", rows))
			return nil
		},
	}

	cmd.Flags().String("run-id", "", "run ID to query (required)")
	return cmd
}

// newRunCancelCommand creates `lobster run cancel`.
func newRunCancelCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel an in-progress run",
		Long:  "Request cancellation of a running async run via the daemon.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			runID, _ := cmd.Flags().GetString("run-id")
			if strings.TrimSpace(runID) == "" {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Missing flag",
					"--run-id is required", "Example: lobster run cancel --run-id <id>", ""))
				return &ExitError{Code: ExitConfigError}
			}

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
					err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			client := runv1.NewRunServiceClient(conn)
			_, err = client.CancelRun(ctx, &runv1.CancelRunRequest{RunId: runID})
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Cancel failed",
					err.Error(), "", ""))
				return &ExitError{Code: exitCodeForRunError(err)}
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(),
				ui.RenderSuccess("Run cancelled", "Run ID "+runID+" has been requested for cancellation."))
			return nil
		},
	}

	cmd.Flags().String("run-id", "", "run ID to cancel (required)")
	return cmd
}

// daemonConnFromCmd reads the daemon connection flags from the command (which
// inherits them as persistent flags from the `run` parent) and dials.
func daemonConnFromCmd(ctx context.Context, cmd *cobra.Command) (*grpc.ClientConn, error) {
	addr := flagStringInherit(cmd, "executor-addr")
	if addr == "" {
		return nil, fmt.Errorf("--executor-addr is required for daemon operations")
	}
	authToken := flagStringInherit(cmd, "auth-token")
	caFile := flagStringInherit(cmd, "tls-ca-file")
	certFile := flagStringInherit(cmd, "tls-cert-file")
	keyFile := flagStringInherit(cmd, "tls-key-file")

	return dialDaemon(ctx, addr, authToken, caFile, certFile, keyFile)
}

// flagStringInherit looks up a string flag on cmd or any of its parents.
func flagStringInherit(cmd *cobra.Command, name string) string {
	if f := cmd.Flags().Lookup(name); f != nil {
		return f.Value.String()
	}
	if f := cmd.InheritedFlags().Lookup(name); f != nil {
		return f.Value.String()
	}
	return ""
}

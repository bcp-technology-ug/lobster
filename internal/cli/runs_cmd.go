package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	commonv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/common"
	runv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/run"
	"github.com/bcp-technology/lobster/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newRunsCommand creates the `lobster runs` command group.
func newRunsCommand(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runs",
		Short: "Browse and inspect run history from the daemon",
		Long:  "Query run history stored in the daemon. Requires --executor-mode daemon.",
	}
	addDaemonPersistentFlags(cmd)
	cmd.AddCommand(newRunsListCommand(v))
	cmd.AddCommand(newRunsGetCommand(v))
	cmd.AddCommand(newRunsCancelCommand())
	return cmd
}

// newRunsListCommand creates `lobster runs list`.
func newRunsListCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List run history",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			format, _ := cmd.Flags().GetString("format")
			workspace, _ := cmd.Flags().GetString("workspace")
			statusStr, _ := cmd.Flags().GetString("status")
			limit, _ := cmd.Flags().GetInt32("limit")

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
					err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			client := runv1.NewRunServiceClient(conn)

			req := &runv1.ListRunsRequest{
				WorkspaceId: workspace,
				PageSize:    uint32(limit),
			}
			if statusStr != "" {
				req.Status = runStatusFromString(statusStr)
			}

			resp, err := client.ListRuns(ctx, req)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("List runs failed", err.Error(), "", ""))
				return &ExitError{Code: ExitRuntimeError}
			}

			if format == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp.GetRuns())
			}

			// TUI mode when interactive, plain table otherwise.
			if ui.IsInteractive() && format == "" {
				m := ui.NewRunsListModel(client, workspace)
				p := tea.NewProgram(m, tea.WithAltScreen())
				if _, err := p.Run(); err != nil {
					return fmt.Errorf("TUI error: %w", err)
				}
				return nil
			}

			// Plain text table fallback.
			if len(resp.GetRuns()) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleMuted.Render("No runs found."))
				return nil
			}
			rows := make([][2]string, 0, len(resp.GetRuns()))
			for _, r := range resp.GetRuns() {
				status := runStatusLabel(r.GetStatus())
				created := ""
				if r.GetCreatedAt() != nil {
					created = r.GetCreatedAt().AsTime().Format("2006-01-02 15:04:05")
				}
				rows = append(rows, [2]string{r.GetRunId()[:min(8, len(r.GetRunId()))], status + "  " + created})
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable("Runs", rows))
			return nil
		},
	}

	cmd.Flags().String("format", "", "output format: text|json")
	cmd.Flags().String("workspace", "", "filter by workspace")
	cmd.Flags().String("status", "", "filter by status: pending|running|passed|failed|cancelled")
	cmd.Flags().Int32("limit", 20, "number of runs to return")
	return cmd
}

// newRunsGetCommand creates `lobster runs get <run-id>`.
func newRunsGetCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <run-id>",
		Short: "Get detailed information about a specific run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			runID := strings.TrimSpace(args[0])
			format, _ := cmd.Flags().GetString("format")

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
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Get run failed", err.Error(), "", ""))
				return &ExitError{Code: exitCodeForRunError(err)}
			}

			if format == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp.GetRun())
			}

			// TUI detail view when interactive.
			if ui.IsInteractive() && format == "" {
				m := ui.NewRunDetailModel(resp.GetRun())
				p := tea.NewProgram(m, tea.WithAltScreen())
				if _, err := p.Run(); err != nil {
					return fmt.Errorf("TUI error: %w", err)
				}
				return nil
			}

			// Plain text key-value table.
			run := resp.GetRun()
			rows := [][]string{
				{"Run ID", run.GetRunId()},
				{"Status", runStatusLabel(run.GetStatus())},
				{"Workspace", run.GetWorkspaceId()},
			}
			if s := run.GetSummary(); s != nil {
				rows = append(rows,
					[]string{"Total", fmt.Sprintf("%d", s.GetTotalScenarios())},
					[]string{"Passed", fmt.Sprintf("%d", s.GetPassedScenarios())},
					[]string{"Failed", fmt.Sprintf("%d", s.GetFailedScenarios())},
					[]string{"Skipped", fmt.Sprintf("%d", s.GetSkippedScenarios())},
				)
				if d := s.GetDuration(); d != nil {
					rows = append(rows, []string{"Duration", d.AsDuration().String()})
				}
			}
			if run.GetCreatedAt() != nil {
				rows = append(rows, []string{"Created", run.GetCreatedAt().AsTime().Format("2006-01-02 15:04:05")})
			}
			if run.GetStartedAt() != nil {
				rows = append(rows, []string{"Started", run.GetStartedAt().AsTime().Format("2006-01-02 15:04:05")})
			}
			if run.GetEndedAt() != nil {
				rows = append(rows, []string{"Ended", run.GetEndedAt().AsTime().Format("2006-01-02 15:04:05")})
			}

			pairs := make([][2]string, 0, len(rows))
			for _, r := range rows {
				pairs = append(pairs, [2]string{r[0], r[1]})
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable("Run detail", pairs))
			return nil
		},
	}
	cmd.Flags().String("format", "", "output format: text|json")
	return cmd
}

// runStatusFromString converts a string like "passed" to the proto enum.
func runStatusFromString(s string) commonv1.RunStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pending":
		return commonv1.RunStatus_RUN_STATUS_PENDING
	case "running":
		return commonv1.RunStatus_RUN_STATUS_RUNNING
	case "passed":
		return commonv1.RunStatus_RUN_STATUS_PASSED
	case "failed":
		return commonv1.RunStatus_RUN_STATUS_FAILED
	case "cancelled":
		return commonv1.RunStatus_RUN_STATUS_CANCELLED
	default:
		return commonv1.RunStatus_RUN_STATUS_UNSPECIFIED
	}
}

// runStatusLabel returns a short human-readable label for a RunStatus.
func runStatusLabel(s commonv1.RunStatus) string {
	switch s {
	case commonv1.RunStatus_RUN_STATUS_PENDING:
		return ui.StyleWarning.Render("pending")
	case commonv1.RunStatus_RUN_STATUS_RUNNING:
		return ui.StyleWarning.Render("running")
	case commonv1.RunStatus_RUN_STATUS_PASSED:
		return ui.StyleSuccess.Render("passed")
	case commonv1.RunStatus_RUN_STATUS_FAILED:
		return ui.StyleError.Render("failed")
	case commonv1.RunStatus_RUN_STATUS_CANCELLED:
		return ui.StyleMuted.Render("cancelled")
	default:
		return ui.StyleMuted.Render("unknown")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

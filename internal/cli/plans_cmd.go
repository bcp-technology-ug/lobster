package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	planv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/plan"
	"github.com/bcp-technology/lobster/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newPlansCommand creates the `lobster plans` command group.
func newPlansCommand(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plans",
		Short: "Browse and inspect saved execution plans from the daemon",
		Long:  "Query persisted execution plans stored in the daemon.",
	}
	addDaemonPersistentFlags(cmd)
	cmd.AddCommand(newPlansListCommand(v))
	cmd.AddCommand(newPlansGetCommand(v))
	return cmd
}

// newPlansListCommand creates `lobster plans list`.
func newPlansListCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List saved execution plans",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			format, _ := cmd.Flags().GetString("format")
			workspace, _ := cmd.Flags().GetString("workspace")
			limit, _ := cmd.Flags().GetInt32("limit")

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
					err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			client := planv1.NewPlanServiceClient(conn)
			resp, err := client.ListPlans(ctx, &planv1.ListPlansRequest{
				WorkspaceId: workspace,
				PageSize:    uint32(limit),
			})
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("List plans failed", err.Error(), "", ""))
				return &ExitError{Code: ExitRuntimeError}
			}

			if format == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp.GetPlans())
			}

			if ui.IsInteractive() && format == "" {
				m := ui.NewPlansListModel(client, workspace)
				p := tea.NewProgram(m, tea.WithAltScreen())
				if _, err := p.Run(); err != nil {
					return fmt.Errorf("TUI error: %w", err)
				}
				return nil
			}

			if len(resp.GetPlans()) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleMuted.Render("No plans found."))
				return nil
			}
			rows := make([][2]string, 0, len(resp.GetPlans()))
			for _, pl := range resp.GetPlans() {
				created := ""
				if pl.GetCreatedAt() != nil {
					created = pl.GetCreatedAt().AsTime().Format("2006-01-02 15:04:05")
				}
				scenarios := fmt.Sprintf("%d scenarios", len(pl.GetScenarios()))
				rows = append(rows, [2]string{
					pl.GetPlanId()[:min(8, len(pl.GetPlanId()))],
					scenarios + "  " + created,
				})
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable("Plans", rows))
			return nil
		},
	}
	cmd.Flags().String("format", "", "output format: text|json")
	cmd.Flags().String("workspace", "", "filter by workspace")
	cmd.Flags().Int32("limit", 20, "number of plans to return")
	return cmd
}

// newPlansGetCommand creates `lobster plans get <plan-id>`.
func newPlansGetCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <plan-id>",
		Short: "Get details of a saved execution plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			planID := strings.TrimSpace(args[0])
			format, _ := cmd.Flags().GetString("format")

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
					err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			client := planv1.NewPlanServiceClient(conn)
			resp, err := client.GetPlan(ctx, &planv1.GetPlanRequest{PlanId: planID})
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Get plan failed", err.Error(), "", ""))
				return &ExitError{Code: ExitRuntimeError}
			}

			if format == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp.GetPlan())
			}

			plan := resp.GetPlan()
			rows := [][2]string{
				{"Plan ID", plan.GetPlanId()},
				{"Workspace", plan.GetWorkspaceId()},
				{"Scenarios", fmt.Sprintf("%d", len(plan.GetScenarios()))},
			}
			if plan.GetCreatedAt() != nil {
				rows = append(rows, [2]string{"Created", plan.GetCreatedAt().AsTime().Format("2006-01-02 15:04:05")})
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable("Plan", rows))

			// Scenario list
			if len(plan.GetScenarios()) > 0 {
				scenarioRows := make([][2]string, 0, len(plan.GetScenarios()))
				for i, sc := range plan.GetScenarios() {
					scenarioRows = append(scenarioRows, [2]string{
						fmt.Sprintf("%d.", i+1),
						sc.GetFeatureName() + " › " + sc.GetScenarioName(),
					})
				}
				_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable("Scenarios", scenarioRows))
			}
			return nil
		},
	}
	cmd.Flags().String("format", "", "output format: text|json")
	return cmd
}

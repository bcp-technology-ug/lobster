package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	stackv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/stack"
	"github.com/bcp-technology-ug/lobster/internal/ui"
)

// newStackCommand creates the `lobster stack` command group.
func newStackCommand(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack",
		Short: "Manage the Docker Compose stack via the daemon",
		Long:  "Control stack lifecycle and inspect service health through the daemon.",
	}
	addDaemonPersistentFlags(cmd)
	cmd.AddCommand(newStackStatusCommand(v))
	cmd.AddCommand(newStackUpCommand(v))
	cmd.AddCommand(newStackDownCommand(v))
	cmd.AddCommand(newStackLogsCommand(v))
	return cmd
}

// newStackStatusCommand creates `lobster stack status`.
func newStackStatusCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show stack health and service status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			workspace, _ := cmd.Flags().GetString("workspace")
			watch, _ := cmd.Flags().GetBool("watch")

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
					err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			client := stackv1.NewStackServiceClient(conn)

			if ui.IsInteractive() || watch {
				m := ui.NewStackStatusModel(client, workspace, watch)
				p := tea.NewProgram(m, tea.WithAltScreen())
				if _, err := p.Run(); err != nil {
					return fmt.Errorf("TUI error: %w", err)
				}
				return nil
			}

			resp, err := client.GetStackStatus(ctx, &stackv1.GetStackStatusRequest{WorkspaceId: workspace})
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Get stack status failed", err.Error(), "", ""))
				return &ExitError{Code: ExitOrchestration}
			}

			stack := resp.GetStack()
			rows := [][2]string{
				{"Stack ID", stack.GetStackId()},
				{"Workspace", stack.GetWorkspaceId()},
				{"Project", stack.GetProjectName()},
				{"Status", stackStatusLabel(stack.GetStatus())},
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable("Stack", rows))

			if len(stack.GetServices()) > 0 {
				serviceRows := make([][2]string, 0, len(stack.GetServices()))
				for _, svc := range stack.GetServices() {
					serviceRows = append(serviceRows, [2]string{
						svc.GetName(),
						serviceHealthLabel(svc.GetHealth()) + "  " + svc.GetStatus(),
					})
				}
				_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable("Services", serviceRows))
			}
			return nil
		},
	}
	cmd.Flags().String("workspace", "", "workspace ID")
	cmd.Flags().Bool("watch", false, "keep watching and refresh every 2s")
	return cmd
}

// newStackUpCommand creates `lobster stack up`.
func newStackUpCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Provision and start the compose stack",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			workspace, _ := cmd.Flags().GetString("workspace")
			profile, _ := cmd.Flags().GetString("profile")

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
					err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			client := stackv1.NewStackServiceClient(conn)
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleMuted.Render("Provisioning stack..."))

			resp, err := client.EnsureStack(ctx, &stackv1.EnsureStackRequest{
				WorkspaceId: workspace,
				ProfileName: profile,
			})
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Stack provisioning failed", err.Error(), "", ""))
				return &ExitError{Code: ExitOrchestration}
			}

			if resp.GetStack() != nil {
				_, _ = fmt.Fprint(cmd.OutOrStdout(),
					ui.RenderSuccess("Stack ready", fmt.Sprintf("Stack %q is %s",
						resp.GetStack().GetStackId(), stackStatusLabel(resp.GetStack().GetStatus()))))
			} else {
				_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderSuccess("Stack ready", "Stack provisioned successfully."))
			}
			return nil
		},
	}
	cmd.Flags().String("workspace", "", "workspace ID")
	cmd.Flags().String("profile", "", "compose profile to activate")
	return cmd
}

// newStackDownCommand creates `lobster stack down`.
func newStackDownCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Tear down and remove the compose stack",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			workspace, _ := cmd.Flags().GetString("workspace")
			force, _ := cmd.Flags().GetBool("force")

			if !force && ui.IsInteractive() {
				var confirmed bool
				form := ui.NewConfirmForm("Tear down stack?", &confirmed)
				if formErr := form.Run(); formErr != nil || !confirmed {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleMuted.Render("Aborted."))
					return nil
				}
			}

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
					err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			client := stackv1.NewStackServiceClient(conn)
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleMuted.Render("Tearing down stack..."))

			_, err = client.TeardownStack(ctx, &stackv1.TeardownStackRequest{WorkspaceId: workspace})
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Stack teardown failed", err.Error(), "", ""))
				return &ExitError{Code: ExitOrchestration}
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderSuccess("Stack torn down", "All services stopped and removed."))
			return nil
		},
	}
	cmd.Flags().String("workspace", "", "workspace ID")
	cmd.Flags().Bool("force", false, "skip confirmation prompt")
	return cmd
}

// newStackLogsCommand creates `lobster stack logs`.
func newStackLogsCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Stream logs from compose services",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			workspace, _ := cmd.Flags().GetString("workspace")
			service, _ := cmd.Flags().GetString("service")
			tail, _ := cmd.Flags().GetInt32("tail")

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
					err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			client := stackv1.NewStackServiceClient(conn)
			stream, err := client.GetStackLogs(ctx, &stackv1.GetStackLogsRequest{
				WorkspaceId: workspace,
				ServiceName: service,
				TailLines:   uint32(tail),
			})
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Stream logs failed", err.Error(), "", ""))
				return &ExitError{Code: ExitOrchestration}
			}

			for {
				line, recvErr := stream.Recv()
				if errors.Is(recvErr, io.EOF) {
					break
				}
				if recvErr != nil {
					return fmt.Errorf("stream error: %w", recvErr)
				}
				svcLabel := ""
				if line.GetServiceName() != "" {
					svcLabel = ui.StyleLabel.Render("["+line.GetServiceName()+"]") + " "
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), svcLabel+line.GetLine())
			}
			return nil
		},
	}
	cmd.Flags().String("workspace", "", "workspace ID")
	cmd.Flags().String("service", "", "filter logs to a specific service name")
	cmd.Flags().Int32("tail", 0, "number of last log lines (0 = all)")
	return cmd
}

// stackStatusLabel returns a human-readable label for a StackStatus.
func stackStatusLabel(s stackv1.StackStatus) string {
	switch s {
	case stackv1.StackStatus_STACK_STATUS_PROVISIONING:
		return ui.StyleWarning.Render("provisioning")
	case stackv1.StackStatus_STACK_STATUS_HEALTHY:
		return ui.StyleSuccess.Render("healthy")
	case stackv1.StackStatus_STACK_STATUS_DEGRADED:
		return ui.StyleWarning.Render("degraded")
	case stackv1.StackStatus_STACK_STATUS_UNHEALTHY:
		return ui.StyleError.Render("unhealthy")
	case stackv1.StackStatus_STACK_STATUS_TEARDOWN:
		return ui.StyleMuted.Render("teardown")
	default:
		return ui.StyleMuted.Render("unknown")
	}
}

// serviceHealthLabel returns a short label for a ServiceHealth enum.
func serviceHealthLabel(h stackv1.ServiceHealth) string {
	switch h {
	case stackv1.ServiceHealth_SERVICE_HEALTH_HEALTHY:
		return ui.StyleSuccess.Render(ui.IconCheck)
	case stackv1.ServiceHealth_SERVICE_HEALTH_STARTING:
		return ui.StyleWarning.Render(ui.IconDot)
	case stackv1.ServiceHealth_SERVICE_HEALTH_UNHEALTHY:
		return ui.StyleError.Render(ui.IconCross)
	default:
		return ui.StyleMuted.Render("?")
	}
}

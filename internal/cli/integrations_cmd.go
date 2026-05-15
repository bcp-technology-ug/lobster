package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	integrationsv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/integrations"
	"github.com/bcp-technology-ug/lobster/internal/ui"
)

// newIntegrationsCommand creates the `lobster integrations` command group.
func newIntegrationsCommand(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "integrations",
		Short: "Manage integration adapters in the daemon",
		Long:  "List, inspect, enable, disable, or validate integration adapters registered in the daemon.",
	}
	addDaemonPersistentFlags(cmd)
	cmd.AddCommand(newIntegrationsListCommand(v))
	cmd.AddCommand(newIntegrationsGetCommand(v))
	cmd.AddCommand(newIntegrationsEnableCommand(v))
	cmd.AddCommand(newIntegrationsDisableCommand(v))
	cmd.AddCommand(newIntegrationsValidateCommand(v))
	return cmd
}

// newIntegrationsListCommand creates `lobster integrations list`.
func newIntegrationsListCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered integration adapters",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			format, _ := cmd.Flags().GetString("format")
			limit, _ := cmd.Flags().GetInt32("limit")

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
					err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			client := integrationsv1.NewIntegrationServiceClient(conn)
			resp, err := client.ListIntegrationAdapters(ctx, &integrationsv1.ListIntegrationAdaptersRequest{
				PageSize: uint32(limit),
			})
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("List adapters failed", err.Error(), "", ""))
				return &ExitError{Code: ExitRuntimeError}
			}

			if format == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp.GetAdapters())
			}

			if len(resp.GetAdapters()) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleMuted.Render("No adapters registered."))
				return nil
			}
			rows := make([][2]string, 0, len(resp.GetAdapters()))
			for _, a := range resp.GetAdapters() {
				rows = append(rows, [2]string{
					a.GetName() + " (" + a.GetAdapterId()[:min(8, len(a.GetAdapterId()))] + ")",
					adapterStateLabel(a.GetState()) + "  " + a.GetType(),
				})
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable("Integration Adapters", rows))
			return nil
		},
	}
	cmd.Flags().String("format", "", "output format: text|json")
	cmd.Flags().Int32("limit", 20, "max adapters to return")
	return cmd
}

// newIntegrationsGetCommand creates `lobster integrations get <adapter-id>`.
func newIntegrationsGetCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <adapter-id>",
		Short: "Get details of an integration adapter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			adapterID := strings.TrimSpace(args[0])
			format, _ := cmd.Flags().GetString("format")

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
					err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			client := integrationsv1.NewIntegrationServiceClient(conn)
			resp, err := client.GetIntegrationAdapter(ctx, &integrationsv1.GetIntegrationAdapterRequest{AdapterId: adapterID})
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Get adapter failed", err.Error(), "", ""))
				return &ExitError{Code: ExitRuntimeError}
			}

			if format == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp.GetAdapter())
			}

			a := resp.GetAdapter()
			rows := [][2]string{
				{"Adapter ID", a.GetAdapterId()},
				{"Name", a.GetName()},
				{"Type", a.GetType()},
				{"State", adapterStateLabel(a.GetState())},
			}
			if a.GetUpdatedAt() != nil {
				rows = append(rows, [2]string{"Updated", a.GetUpdatedAt().AsTime().Format("2006-01-02 15:04:05")})
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable("Adapter", rows))

			if len(a.GetCapabilities()) > 0 {
				capRows := make([][2]string, 0, len(a.GetCapabilities()))
				for _, c := range a.GetCapabilities() {
					enabled := ui.StyleMuted.Render("disabled")
					if c.GetEnabled() {
						enabled = ui.StyleSuccess.Render("enabled")
					}
					capRows = append(capRows, [2]string{c.GetName(), enabled})
				}
				_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable("Capabilities", capRows))
			}
			return nil
		},
	}
	cmd.Flags().String("format", "", "output format: text|json")
	return cmd
}

// newIntegrationsEnableCommand creates `lobster integrations enable <adapter-id>`.
func newIntegrationsEnableCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <adapter-id>",
		Short: "Enable an integration adapter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setAdapterState(cmd, strings.TrimSpace(args[0]), true)
		},
	}
	cmd.Flags().String("reason", "", "optional reason for the state change")
	return cmd
}

// newIntegrationsDisableCommand creates `lobster integrations disable <adapter-id>`.
func newIntegrationsDisableCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <adapter-id>",
		Short: "Disable an integration adapter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setAdapterState(cmd, strings.TrimSpace(args[0]), false)
		},
	}
	cmd.Flags().String("reason", "", "optional reason for the state change")
	return cmd
}

func setAdapterState(cmd *cobra.Command, adapterID string, enable bool) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	reason, _ := cmd.Flags().GetString("reason")

	conn, err := daemonConnFromCmd(ctx, cmd)
	if err != nil {
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
			err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
		return &ExitError{Code: ExitOrchestration}
	}
	defer conn.Close()

	client := integrationsv1.NewIntegrationServiceClient(conn)
	resp, err := client.SetIntegrationAdapterState(ctx, &integrationsv1.SetIntegrationAdapterStateRequest{
		AdapterId: adapterID,
		Enabled:   enable,
		Reason:    reason,
	})
	if err != nil {
		action := "enable"
		if !enable {
			action = "disable"
		}
		_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Failed to "+action+" adapter", err.Error(), "", ""))
		return &ExitError{Code: ExitRuntimeError}
	}

	a := resp.GetAdapter()
	verb := "enabled"
	if !enable {
		verb = "disabled"
	}
	_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderSuccess(
		"Adapter "+verb,
		fmt.Sprintf("%q is now %s", a.GetName(), adapterStateLabel(a.GetState())),
	))
	return nil
}

// newIntegrationsValidateCommand creates `lobster integrations validate <adapter-id>`.
func newIntegrationsValidateCommand(_ *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "validate <adapter-id>",
		Short: "Run a live diagnostic check on an integration adapter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			adapterID := strings.TrimSpace(args[0])

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
					err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleMuted.Render("Validating adapter "+adapterID+"..."))

			client := integrationsv1.NewIntegrationServiceClient(conn)
			resp, err := client.ValidateIntegrationAdapter(ctx, &integrationsv1.ValidateIntegrationAdapterRequest{
				AdapterId: adapterID,
			})
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Validation failed", err.Error(), "", ""))
				return &ExitError{Code: ExitRuntimeError}
			}

			if resp.GetOk() {
				_, _ = fmt.Fprint(cmd.OutOrStdout(),
					ui.RenderSuccess("Validation passed", "Adapter "+adapterID+" is reachable and responding."))
			} else {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(),
					ui.RenderError("Validation failed", "Adapter "+adapterID+" is not responding correctly.", "", ""))
				return &ExitError{Code: ExitRuntimeError}
			}
			return nil
		},
	}
}

// adapterStateLabel returns a styled string for an AdapterState.
func adapterStateLabel(s integrationsv1.AdapterState) string {
	switch s {
	case integrationsv1.AdapterState_ADAPTER_STATE_READY:
		return ui.StyleSuccess.Render("ready")
	case integrationsv1.AdapterState_ADAPTER_STATE_DEGRADED:
		return ui.StyleWarning.Render("degraded")
	case integrationsv1.AdapterState_ADAPTER_STATE_ERROR:
		return ui.StyleError.Render("error")
	case integrationsv1.AdapterState_ADAPTER_STATE_DISABLED:
		return ui.StyleMuted.Render("disabled")
	default:
		return ui.StyleMuted.Render("unknown")
	}
}

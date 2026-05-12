package cli

import (
	"context"
	"encoding/json"
	"fmt"

	adminv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/admin"
	"github.com/bcp-technology/lobster/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newAdminCommand creates the `lobster admin` command group.
func newAdminCommand(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Daemon diagnostics and administrative operations",
		Long:  "Inspect daemon health, API capabilities, and effective configuration.",
	}
	addDaemonPersistentFlags(cmd)
	cmd.AddCommand(newAdminHealthCommand(v))
	cmd.AddCommand(newAdminCapabilitiesCommand(v))
	cmd.AddCommand(newAdminConfigCommand(v))
	return cmd
}

// newAdminHealthCommand creates `lobster admin health`.
func newAdminHealthCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check daemon health status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			format, _ := cmd.Flags().GetString("format")

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
					err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			client := adminv1.NewAdminServiceClient(conn)
			resp, err := client.GetHealth(ctx, &adminv1.GetHealthRequest{})
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Health check failed", err.Error(), "", ""))
				return &ExitError{Code: ExitRuntimeError}
			}

			if format == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp.GetHealth())
			}

			h := resp.GetHealth()
			liveStr := ui.StyleError.Render("not live")
			if h.GetLive() {
				liveStr = ui.StyleSuccess.Render("live")
			}
			readyStr := ui.StyleError.Render("not ready")
			if h.GetReady() {
				readyStr = ui.StyleSuccess.Render("ready")
			}

			rows := [][2]string{
				{"Live", liveStr},
				{"Ready", readyStr},
				{"Version", h.GetVersion()},
			}
			if h.GetObservedAt() != nil {
				rows = append(rows, [2]string{"Observed", h.GetObservedAt().AsTime().Format("2006-01-02 15:04:05")})
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable("Daemon health", rows))

			if !h.GetLive() || !h.GetReady() {
				return &ExitError{Code: ExitRuntimeError}
			}
			return nil
		},
	}
	cmd.Flags().String("format", "", "output format: text|json")
	return cmd
}

// newAdminCapabilitiesCommand creates `lobster admin capabilities`.
func newAdminCapabilitiesCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capabilities",
		Short: "List daemon API capabilities and feature flags",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			format, _ := cmd.Flags().GetString("format")

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
					err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			client := adminv1.NewAdminServiceClient(conn)
			resp, err := client.GetCapabilities(ctx, &adminv1.GetCapabilitiesRequest{})
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Get capabilities failed", err.Error(), "", ""))
				return &ExitError{Code: ExitRuntimeError}
			}

			if format == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp)
			}

			rows := [][2]string{
				{"API Package", resp.GetApiPackage()},
				{"API Version", resp.GetApiVersion()},
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable("API", rows))

			if len(resp.GetCapabilities()) > 0 {
				capRows := make([][2]string, 0, len(resp.GetCapabilities()))
				for _, c := range resp.GetCapabilities() {
					enabled := ui.StyleMuted.Render("disabled")
					if c.GetEnabled() {
						enabled = ui.StyleSuccess.Render("enabled")
					}
					capRows = append(capRows, [2]string{c.GetName(), enabled + "  " + c.GetMinClientVersion()})
				}
				_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable("Capabilities", capRows))
			}
			return nil
		},
	}
	cmd.Flags().String("format", "", "output format: text|json")
	return cmd
}

// newAdminConfigCommand creates `lobster admin config`.
func newAdminConfigCommand(_ *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show the daemon's effective sanitized configuration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			format, _ := cmd.Flags().GetString("format")

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Daemon connection failed",
					err.Error(), "Check --executor-addr and TLS/auth flags.", ""))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			client := adminv1.NewAdminServiceClient(conn)
			resp, err := client.GetConfigSummary(ctx, &adminv1.GetConfigSummaryRequest{})
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Get config failed", err.Error(), "", ""))
				return &ExitError{Code: ExitRuntimeError}
			}

			if format == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(resp.GetConfig())
			}

			s := resp.GetConfig()
			rows := [][2]string{
				{"Workspace", s.GetWorkspaceId()},
				{"Profile", s.GetActiveProfile()},
			}
			if s.GetExecution() != nil {
				ex := s.GetExecution()
				rows = append(rows,
					[2]string{"Fail Fast", fmt.Sprintf("%v", ex.GetFailFast())},
					[2]string{"Soft Assert", fmt.Sprintf("%v", ex.GetSoftAssert())},
					[2]string{"Keep Stack", fmt.Sprintf("%v", ex.GetKeepStack())},
				)
				if ex.GetRunTimeout() != nil {
					rows = append(rows, [2]string{"Run Timeout", ex.GetRunTimeout().AsDuration().String()})
				}
				if ex.GetStepTimeout() != nil {
					rows = append(rows, [2]string{"Step Timeout", ex.GetStepTimeout().AsDuration().String()})
				}
			}
			if s.GetPersistence() != nil {
				p := s.GetPersistence()
				rows = append(rows,
					[2]string{"SQLite Path", p.GetSqlitePath()},
					[2]string{"Journal Mode", p.GetJournalMode()},
				)
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable("Daemon configuration", rows))
			return nil
		},
	}
	cmd.Flags().String("format", "", "output format: text|json")
	return cmd
}

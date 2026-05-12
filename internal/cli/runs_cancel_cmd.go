package cli

import (
	"context"
	"fmt"

	commonv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/common"
	runv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/run"
	"github.com/bcp-technology-ug/lobster/internal/ui"
	"github.com/spf13/cobra"
)

// newRunsCancelCommand creates `lobster runs cancel <run-id>`.
func newRunsCancelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel <run-id>",
		Short: "Cancel an in-progress run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			runID := args[0]
			reason, _ := cmd.Flags().GetString("reason")

			conn, err := daemonConnFromCmd(ctx, cmd)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError(
					"Daemon connection failed", err.Error(),
					"Check --executor-addr and TLS/auth flags.", "",
				))
				return &ExitError{Code: ExitOrchestration}
			}
			defer conn.Close()

			client := runv1.NewRunServiceClient(conn)
			resp, err := client.CancelRun(ctx, &runv1.CancelRunRequest{
				RunId:  runID,
				Reason: reason,
			})
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError(
					"Cancel failed", err.Error(), "", "",
				))
				return &ExitError{Code: ExitRuntimeError}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%s Run %s cancelled — terminal status: %s\n",
				ui.IconCheck,
				resp.GetRunId(),
				runStatusLabel(commonv1.RunStatus(resp.GetTerminalStatus())),
			)
			return nil
		},
	}

	cmd.Flags().String("reason", "", "optional human-readable cancellation reason")
	return cmd
}

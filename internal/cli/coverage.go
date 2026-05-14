package cli

import (
	"fmt"
	"strings"

	"github.com/bcp-technology-ug/lobster/internal/coverage"
	"github.com/bcp-technology-ug/lobster/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newCoverageCommand(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "coverage",
		Short: "Report test coverage against the API and CLI surface",
		Long: "Parse proto files and/or an OpenAPI spec to build the known service surface, " +
			"then compare it against @covers: tags in feature files. " +
			"Reports gaps and enforces minimum scenario depth thresholds.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			protoGlob, _ := cmd.Flags().GetString("proto")
			openapiPath, _ := cmd.Flags().GetString("openapi")
			featGlob := valueString(cmd, v, "features.paths.0", "features")
			minScenarios, _ := cmd.Flags().GetInt("min-scenarios")
			warnMinScenarios, _ := cmd.Flags().GetInt("warn-min-scenarios")
			format, _ := cmd.Flags().GetString("format")
			noCLI, _ := cmd.Flags().GetBool("no-cli")
			noProto, _ := cmd.Flags().GetBool("no-proto")
			noOpenAPI, _ := cmd.Flags().GetBool("no-openapi")
			strict, _ := cmd.Flags().GetBool("strict")

			if strings.TrimSpace(featGlob) == "" {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(),
					ui.RenderError("Error", "--features is required", "", ""))
				return &ExitError{Code: ExitConfigError}
			}

			// ── Build surface ─────────────────────────────────────────────

			var surface []coverage.CoverageItem

			if !noProto && strings.TrimSpace(protoGlob) != "" {
				items, err := coverage.ScanProtoGlob(protoGlob)
				if err != nil {
					_, _ = fmt.Fprint(cmd.ErrOrStderr(),
						ui.RenderError("Proto scan failed", err.Error(), "Check --proto glob.", ""))
					return &ExitError{Code: ExitConfigError}
				}
				surface = append(surface, items...)
			}

			if !noOpenAPI && strings.TrimSpace(openapiPath) != "" {
				items, err := coverage.ScanOpenAPI(openapiPath)
				if err != nil {
					_, _ = fmt.Fprint(cmd.ErrOrStderr(),
						ui.RenderError("OpenAPI scan failed", err.Error(), "Check --openapi path.", ""))
					return &ExitError{Code: ExitConfigError}
				}
				surface = append(surface, items...)
			}

			if !noCLI {
				surface = append(surface, coverage.ScanCLI()...)
			}

			if len(surface) == 0 {
				_, _ = fmt.Fprint(cmd.OutOrStdout(),
					ui.RenderWarning("Empty surface",
						"No proto files, OpenAPI spec, or CLI commands were scanned.\n"+
							ui.StyleMuted.Render("Use --proto, --openapi, or remove --no-cli / --no-proto / --no-openapi.")),
				)
				return nil
			}

			// ── Scan features ─────────────────────────────────────────────

			featData, err := coverage.ScanFeatureGlob(featGlob, surface)
			if err != nil {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(),
					ui.RenderError("Feature scan failed", err.Error(), "Check --features glob.", ""))
				return &ExitError{Code: ExitConfigError}
			}

			// ── Build report ──────────────────────────────────────────────

			report := coverage.BuildReport(
				surface,
				featData.Explicit,
				featData.ScenarioCounts,
				featData.Inferred,
				minScenarios,
				warnMinScenarios,
			)

			// ── Render output ─────────────────────────────────────────────

			if strings.EqualFold(format, "json") {
				if err := coverage.RenderJSON(cmd.OutOrStdout(), report); err != nil {
					return err
				}
			} else {
				coverage.RenderText(cmd.OutOrStdout(), report)
			}

			// ── Exit code ─────────────────────────────────────────────────

			if report.UncoveredCount > 0 || report.ThresholdViolations > 0 {
				return &ExitError{Code: ExitConfigError}
			}
			if strict && report.WarnViolations > 0 {
				return &ExitError{Code: ExitConfigError}
			}

			return nil
		},
	}

	cmd.Flags().String("proto", "proto/**/*.proto", "glob for .proto files to scan for RPC methods")
	cmd.Flags().String("openapi", "gen/openapi/openapi.yaml", "path to OpenAPI YAML spec")
	cmd.Flags().String("features", "", "feature file glob (e.g. 'features/**/*.feature')")
	cmd.Flags().Int("min-scenarios", 1, "minimum scenarios per covered item (exit 2 if violated)")
	cmd.Flags().Int("warn-min-scenarios", 3, "warn when scenarios per item is below this threshold")
	cmd.Flags().String("format", "text", "output format: text|json")
	cmd.Flags().Bool("no-cli", false, "skip CLI command surface")
	cmd.Flags().Bool("no-proto", false, "skip proto RPC surface")
	cmd.Flags().Bool("no-openapi", false, "skip OpenAPI endpoint surface")
	cmd.Flags().Bool("strict", false, "treat warn-threshold violations as errors (exit 2)")

	return cmd
}

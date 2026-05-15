package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/bcp-technology-ug/lobster/internal/parser"
	"github.com/bcp-technology-ug/lobster/internal/ui"
)

func newLintCommand(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Enforce style and quality checks on feature files",
		Long:  "Parse feature files and apply quality rules. Reports warnings and errors grouped by file.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			glob := valueString(cmd, v, "features.paths.0", "features")
			if strings.TrimSpace(glob) == "" {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Error", "--features is required (e.g. --features 'features/**/*.feature')", "", ""))
				return &ExitError{Code: ExitConfigError}
			}
			strictMode, _ := cmd.Flags().GetBool("strict")

			paths, err := filepath.Glob(glob)
			if err != nil {
				return fmt.Errorf("invalid glob pattern %q: %w", glob, err)
			}
			if len(paths) == 0 {
				_, _ = fmt.Fprint(cmd.OutOrStdout(),
					ui.RenderWarning("No feature files found",
						"Pattern "+ui.StyleCode.Render(glob)+" matched no files.\n"+
							ui.StyleMuted.Render("Check your --features glob or features.paths in lobster.yaml.")),
				)
				return nil
			}

			// Parse all files first; skip any that fail to parse.
			features, parseErrors := parseFilesForLint(paths)
			if len(parseErrors) > 0 {
				for _, pe := range parseErrors {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(),
						ui.StyleError.Render(ui.IconCross+"  "+pe))
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
			}

			// Run lint rules.
			result := parser.Lint(features)
			diagnostics := result.Diagnostics

			if len(diagnostics) == 0 && len(parseErrors) == 0 {
				_, _ = fmt.Fprint(cmd.OutOrStdout(),
					ui.RenderSuccess("All checks passed",
						fmt.Sprintf("%d file(s) linted with no issues.", len(paths))),
				)
				return nil
			}

			// Group diagnostics by file URI.
			byFile := make(map[string][]parser.Diagnostic)
			fileOrder := []string{}
			for _, d := range diagnostics {
				if _, seen := byFile[d.URI]; !seen {
					fileOrder = append(fileOrder, d.URI)
				}
				byFile[d.URI] = append(byFile[d.URI], d)
			}

			// Render per-file sections.
			for _, uri := range fileOrder {
				diags := byFile[uri]
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleSubheading.Render(shortPath(uri)))

				for _, d := range diags {
					var icon, line string
					if d.Severity == parser.SeverityError {
						icon = ui.StyleError.Render(ui.IconCross)
					} else {
						icon = ui.StyleWarning.Render(ui.IconWarning)
					}
					if d.ScenarioID != "" {
						line = fmt.Sprintf("  %s  %s  %s",
							icon,
							ui.StyleMuted.Render("["+d.ScenarioID+"]"),
							d.Message,
						)
					} else {
						line = fmt.Sprintf("  %s  %s", icon, d.Message)
					}
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
			}

			// Summary.
			warnings, errs := countDiagnostics(diagnostics)
			// In strict mode, warnings are treated as errors for exit-code purposes.
			effectiveErrors := errs
			if strictMode {
				effectiveErrors += warnings
			}
			summary := fmt.Sprintf("%d file(s)  ·  %d warning(s)  ·  %d error(s)",
				len(paths), warnings, errs)
			if strictMode && warnings > 0 {
				summary += "  (strict: warnings treated as errors)"
			}

			if effectiveErrors > 0 || len(parseErrors) > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleError.Render(ui.IconCross+"  "+summary))
				return &ExitError{Code: ExitConfigError}
			}
			if warnings > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleWarning.Render(ui.IconWarning+"  "+summary))
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleSuccess.Render(ui.IconCheck+"  "+summary))
			}
			return nil
		},
	}

	cmd.Flags().String("features", "", "feature file glob (e.g. 'features/**/*.feature')")
	cmd.Flags().Bool("strict", false, "treat warnings as errors (exit 2)")
	return cmd
}

// parseFilesForLint parses paths that can be parsed, collecting error strings for
// those that fail. It never returns a hard error — parse failures are reported in-band.
func parseFilesForLint(paths []string) ([]*parser.Feature, []string) {
	features := make([]*parser.Feature, 0, len(paths))
	var parseErrors []string
	for _, p := range paths {
		feat, err := parser.Parse(p)
		if err != nil {
			parseErrors = append(parseErrors,
				fmt.Sprintf("parse error in %s: %s", shortPath(p), err.Error()))
			continue
		}
		features = append(features, feat)
	}
	return features, parseErrors
}

func countDiagnostics(diags []parser.Diagnostic) (warnings, errors int) {
	for _, d := range diags {
		if d.Severity == parser.SeverityError {
			errors++
		} else {
			warnings++
		}
	}
	return
}

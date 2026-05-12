package cli

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bcp-technology-ug/lobster/internal/parser"
	"github.com/bcp-technology-ug/lobster/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newValidateCommand(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Parse and validate feature files",
		Long:  "Parse .feature files matching the given glob and report any syntax or structural errors.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			glob := valueString(cmd, v, "features.paths.0", "features")
			if strings.TrimSpace(glob) == "" {
				_, _ = fmt.Fprint(cmd.ErrOrStderr(), ui.RenderError("Error", "--features is required (e.g. --features 'features/**/*.feature')", "", ""))
				return &ExitError{Code: ExitConfigError}
			}
			strictMode, _ := cmd.Flags().GetBool("strict")
			format, _ := cmd.Flags().GetString("format")

			// Expand glob.
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

			// Parse each file and collect results.
			type result struct {
				path      string
				scenarios int
				steps     int
				err       error
			}

			results := make([]result, 0, len(paths))
			parseErrors := 0

			for _, p := range paths {
				feat, parseErr := parser.Parse(p)
				if parseErr != nil {
					results = append(results, result{path: p, err: parseErr})
					parseErrors++
					continue
				}
				totalSteps := 0
				for _, sc := range feat.Scenarios {
					totalSteps += len(sc.Steps)
				}
				results = append(results, result{
					path:      p,
					scenarios: len(feat.Scenarios),
					steps:     totalSteps,
				})
			}

			// JSON output.
			if strings.EqualFold(format, "json") {
				type jsonResult struct {
					Path      string `json:"path"`
					Status    string `json:"status"`
					Scenarios int    `json:"scenarios"`
					Steps     int    `json:"steps"`
					Error     string `json:"error,omitempty"`
				}
				out := make([]jsonResult, 0, len(results))
				for _, r := range results {
					jr := jsonResult{Path: r.path, Scenarios: r.scenarios, Steps: r.steps, Status: "ok"}
					if r.err != nil {
						jr.Status = "error"
						jr.Error = r.err.Error()
					}
					out = append(out, jr)
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if encErr := enc.Encode(out); encErr != nil {
					return encErr
				}
				if parseErrors > 0 {
					return &ExitError{Code: ExitConfigError}
				}
				return nil
			}

			// Build per-file table rows.
			rows := make([][2]string, 0, len(results))
			for _, r := range results {
				key := shortPath(r.path)
				var val string
				if r.err != nil {
					val = ui.StyleError.Render(ui.IconCross + " parse error: " + r.err.Error())
				} else {
					val = ui.StyleSuccess.Render(ui.IconCheck) +
						ui.StyleMuted.Render(fmt.Sprintf("  %d scenarios, %d steps", r.scenarios, r.steps))
				}
				rows = append(rows, [2]string{key, val})
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), ui.RenderKeyValueTable("Feature files", rows))

			// --strict: any parse warning (currently none) or error is a failure.
			// For now strict mode just changes the exit code message.
			summary := fmt.Sprintf("%d file(s)  ·  %d error(s)", len(results), parseErrors)
			if parseErrors == 0 {
				suffix := ""
				if strictMode {
					suffix = "  (strict)"
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleSuccess.Render(ui.IconCheck+"  "+summary+suffix))
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), ui.StyleError.Render(ui.IconCross+"  "+summary))
				return &ExitError{Code: ExitConfigError}
			}

			return nil
		},
	}

	cmd.Flags().String("features", "", "feature file glob (e.g. 'features/**/*.feature')")
	cmd.Flags().Bool("strict", false, "treat warnings as errors (exit 2)")
	cmd.Flags().String("format", "text", "output format: text|json")
	return cmd
}

// shortPath trims the working-directory prefix for display brevity.
// For paths outside the CWD (e.g. absolute /tmp/... paths), returns
// the last two path segments to keep columns readable.
func shortPath(p string) string {
	rel, err := filepath.Rel(".", p)
	if err == nil && !filepath.IsAbs(rel) {
		return rel
	}
	// Outside CWD: show parent/base.
	dir := filepath.Base(filepath.Dir(p))
	base := filepath.Base(p)
	if dir == "." || dir == "/" {
		return base
	}
	return dir + "/" + base
}

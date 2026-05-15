package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bcp-technology-ug/lobster/internal/steps"
	"github.com/bcp-technology-ug/lobster/internal/steps/builtin"
)

// docstringExamples maps raw pattern suffixes to Gherkin DocString/DataTable
// examples. Used by the markdown formatter to add concrete usage snippets for
// complex steps that require a DocString or DataTable argument.
var docstringExamples = map[string]string{
	`request to "([^"]+)" with body:`: `When I send a POST request to "/api/users" with body:
  """json
  {"name": "Alice", "email": "alice@example.com"}
  """`,

	`request to "([^"]+)" with form data:`: `When I send a POST request to "/auth/token" with form data:
  """
  grant_type=client_credentials&client_id=app&client_secret=secret
  """`,

	`"([^"]+)" content should equal:`: `Then the file "output.json" content should equal:
  """
  {"status":"ok"}
  """`,

	`"([^"]+)" with content:`: `When I create the file "config.yaml" with content:
  """yaml
  host: localhost
  port: 8080
  """`,

	`"([^"]+)" with content:$`: `When I append to file "results.csv" with content:
  """
  alice,passed,2026-05-15
  """`,

	`should include fields:`: `Then the response JSON should include fields:
  | field       | value   |
  | status      | active  |
  | role        | admin   |`,
}

// stepJSONEntry is the JSON-serialisable shape of a single step definition.
type stepJSONEntry struct {
	Pattern string `json:"pattern"`
	Source  string `json:"source"`
}

func newStepsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "steps",
		Short: "List all registered step definitions",
		Long: `Display every step pattern registered in this binary, grouped by category.

Agents should run this command first to discover available steps before
writing feature files. Only use patterns listed here — never guess or invent steps.

Formats:
  text      Human-readable grouped list (default)
  json      Machine-readable array for programmatic use
  markdown  LLM-optimised context block, paste directly into an agent prompt`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			format, _ := cmd.Flags().GetString("format")
			filter, _ := cmd.Flags().GetString("filter")

			reg := steps.NewRegistry()
			if err := builtin.Register(reg); err != nil {
				return fmt.Errorf("loading built-in steps: %w", err)
			}

			defs := reg.Defs()

			if filter != "" {
				fl := strings.ToLower(filter)
				var filtered []*steps.StepDef
				for _, d := range defs {
					if strings.Contains(strings.ToLower(d.Source), fl) {
						filtered = append(filtered, d)
					}
				}
				defs = filtered
			}

			w := cmd.OutOrStdout()
			switch strings.ToLower(format) {
			case "json":
				return renderStepsJSON(w, defs)
			case "markdown", "md":
				renderStepsMarkdown(w, defs, filter)
				return nil
			default:
				renderStepsText(w, defs)
				return nil
			}
		},
	}

	cmd.Flags().String("format", "text", "output format: text|json|markdown")
	cmd.Flags().String("filter", "", "filter by category name (e.g. http, shell, fs, service, grpc, vars, wait, assert)")
	return cmd
}

// groupBySource groups step definitions by their Source field, preserving
// insertion order for the first occurrence of each source.
func groupBySource(defs []*steps.StepDef) ([]string, map[string][]*steps.StepDef) {
	order := []string{}
	groups := map[string][]*steps.StepDef{}
	for _, d := range defs {
		if _, seen := groups[d.Source]; !seen {
			order = append(order, d.Source)
		}
		groups[d.Source] = append(groups[d.Source], d)
	}
	return order, groups
}

// categoryLabel converts a source like "builtin:http" to a friendly label.
func categoryLabel(source string) string {
	parts := strings.SplitN(source, ":", 2)
	if len(parts) == 2 {
		return strings.ToUpper(parts[1])
	}
	return strings.ToUpper(source)
}

func renderStepsText(w io.Writer, defs []*steps.StepDef) {
	order, groups := groupBySource(defs)
	total := len(defs)
	_, _ = fmt.Fprintf(w, "%d step definitions\n\n", total)
	for _, src := range order {
		grp := groups[src]
		_, _ = fmt.Fprintf(w, "── %s (%d) ──\n", categoryLabel(src), len(grp))
		for _, d := range grp {
			_, _ = fmt.Fprintf(w, "  %s\n", steps.HumanizePattern(d.RawPattern))
		}
		_, _ = fmt.Fprintln(w)
	}
}

func renderStepsJSON(w io.Writer, defs []*steps.StepDef) error {
	entries := make([]stepJSONEntry, len(defs))
	for i, d := range defs {
		entries[i] = stepJSONEntry{Pattern: d.RawPattern, Source: d.Source}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(entries)
}

func renderStepsMarkdown(w io.Writer, defs []*steps.StepDef, filter string) {
	order, groups := groupBySource(defs)
	total := len(defs)

	_, _ = fmt.Fprintf(w, "# Lobster Step Catalog\n\n")
	_, _ = fmt.Fprintf(w, "> **%d step definitions** across **%d categories**.\n", total, len(order))
	if filter == "" {
		_, _ = fmt.Fprintf(w, "> Run `lobster steps --filter <category>` to narrow results.\n")
	}
	_, _ = fmt.Fprintln(w)

	_, _ = fmt.Fprintf(w, "## Variable Interpolation\n\n")
	_, _ = fmt.Fprintf(w, "All quoted step arguments support `${VAR_NAME}` substitution.\n")
	_, _ = fmt.Fprintf(w, "Variables are resolved from scenario context → suite variables → environment.\n\n")
	_, _ = fmt.Fprintf(w, "```gherkin\n")
	_, _ = fmt.Fprintf(w, "Given I set variable \"BASE\" to \"https://api.example.com\"\n")
	_, _ = fmt.Fprintf(w, "When I send a GET request to \"${BASE}/users\"\n")
	_, _ = fmt.Fprintf(w, "```\n\n")

	for _, src := range order {
		grp := groups[src]
		_, _ = fmt.Fprintf(w, "## %s (`%s`)\n\n", categoryLabel(src), src)

		for _, d := range grp {
			_, _ = fmt.Fprintf(w, "- `%s`\n", steps.HumanizePattern(d.RawPattern))
			// Emit a Gherkin example for complex (DocString/DataTable) steps.
			if example := lookupExample(d.RawPattern); example != "" {
				_, _ = fmt.Fprintf(w, "\n  ```gherkin\n")
				for _, line := range strings.Split(example, "\n") {
					_, _ = fmt.Fprintf(w, "  %s\n", line)
				}
				_, _ = fmt.Fprintf(w, "  ```\n")
			}
		}
		_, _ = fmt.Fprintln(w)
	}

	_, _ = fmt.Fprintf(w, "## Agent Bootstrap Workflow\n\n")
	_, _ = fmt.Fprintf(w, "```sh\n")
	_, _ = fmt.Fprintf(w, "lobster steps --format markdown   # 1. discover capabilities (you are here)\n")
	_, _ = fmt.Fprintf(w, "# 2. write .feature file using ONLY the patterns above\n")
	_, _ = fmt.Fprintf(w, "lobster validate --features 'features/**/*.feature'  # 3. syntax check\n")
	_, _ = fmt.Fprintf(w, "lobster lint     --features 'features/**/*.feature'  # 4. quality check\n")
	_, _ = fmt.Fprintf(w, "lobster plan     --features 'features/**/*.feature'  # 5. dry run\n")
	_, _ = fmt.Fprintf(w, "lobster run      --features 'features/**/*.feature'  # 6. execute\n")
	_, _ = fmt.Fprintf(w, "```\n")
}

// lookupExample returns a Gherkin usage example for a raw step pattern that
// requires a DocString or DataTable, or "" if no example is registered.
func lookupExample(rawPattern string) string {
	for suffix, example := range docstringExamples {
		if strings.Contains(rawPattern, suffix) {
			return example
		}
	}
	return ""
}

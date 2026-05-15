package mcp

import (
	"fmt"
	"io"
	"strings"

	"github.com/bcp-technology-ug/lobster/internal/steps"
)

// docstringExamples contains Gherkin examples for complex steps requiring
// a DocString or DataTable body. Keyed by a unique substring of the raw pattern.
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

	`should include fields:`: `Then the response JSON should include fields:
  | field  | value  |
  | status | active |
  | role   | admin  |`,
}

func renderMarkdown(w io.Writer, defs []*steps.StepDef, filter string) {
	order, groups := groupBySource(defs)
	total := len(defs)

	_, _ = fmt.Fprintf(w, "# Lobster Step Catalog\n\n")
	_, _ = fmt.Fprintf(w, "> **%d step definitions** across **%d categories**.\n", total, len(order))
	if filter == "" {
		_, _ = fmt.Fprintf(w, "> Run `lobster steps --filter <category>` to narrow results.\n")
	}
	_, _ = fmt.Fprintln(w)

	_, _ = fmt.Fprintf(w, "## Variable Interpolation\n\n")
	_, _ = fmt.Fprintf(w, "All quoted step arguments support `${VAR_NAME}` substitution.\n\n")
	_, _ = fmt.Fprintf(w, "```gherkin\n")
	_, _ = fmt.Fprintf(w, "Given I set variable \"BASE\" to \"https://api.example.com\"\n")
	_, _ = fmt.Fprintf(w, "When I send a GET request to \"${BASE}/users\"\n")
	_, _ = fmt.Fprintf(w, "```\n\n")

	for _, src := range order {
		grp := groups[src]
		_, _ = fmt.Fprintf(w, "## %s (`%s`)\n\n", categoryLabel(src), src)
		for _, d := range grp {
			_, _ = fmt.Fprintf(w, "- `%s`\n", steps.HumanizePattern(d.RawPattern))
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
	_, _ = fmt.Fprintf(w, "lobster steps --format markdown   # 1. discover capabilities\n")
	_, _ = fmt.Fprintf(w, "# 2. write .feature file using ONLY the patterns above\n")
	_, _ = fmt.Fprintf(w, "lobster validate --features 'features/**/*.feature'  # 3. syntax\n")
	_, _ = fmt.Fprintf(w, "lobster lint     --features 'features/**/*.feature'  # 4. quality\n")
	_, _ = fmt.Fprintf(w, "lobster plan     --features 'features/**/*.feature'  # 5. dry run\n")
	_, _ = fmt.Fprintf(w, "lobster run      --features 'features/**/*.feature'  # 6. execute\n")
	_, _ = fmt.Fprintf(w, "```\n")
}

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

func categoryLabel(source string) string {
	parts := strings.SplitN(source, ":", 2)
	if len(parts) == 2 {
		return strings.ToUpper(parts[1])
	}
	return strings.ToUpper(source)
}

func lookupExample(rawPattern string) string {
	for suffix, example := range docstringExamples {
		if strings.Contains(rawPattern, suffix) {
			return example
		}
	}
	return ""
}

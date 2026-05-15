package reports

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ANSI colour codes. When the writer is not a TTY the codes are stripped by
// isColour() — we default to enabling them and let the caller pass os.Stdout.
const (
	colReset  = "\033[0m"
	colGreen  = "\033[32m"
	colRed    = "\033[31m"
	colYellow = "\033[33m"
	colCyan   = "\033[36m"
	colGray   = "\033[90m"
	colBold   = "\033[1m"
)

// ConsoleReporter writes a human-readable, coloured summary to an io.Writer.
// Verbose mode prints each step result in addition to the scenario summary.
type ConsoleReporter struct {
	w       io.Writer
	verbose bool
	colour  bool
}

// NewConsoleReporter creates a ConsoleReporter writing to w.
// Set verbose=true to print step-level results. colour controls ANSI output.
func NewConsoleReporter(w io.Writer, verbose, colour bool) *ConsoleReporter {
	if w == nil {
		w = os.Stdout
	}
	return &ConsoleReporter{w: w, verbose: verbose, colour: colour}
}

func (c *ConsoleReporter) RunStarted(r *RunResult) {
	fmt.Fprintf(c.w, "\n%s%sLobster Run%s  id=%s  profile=%s\n",
		c.bold(), c.col(colCyan), c.reset(), r.RunID, r.Profile)
	fmt.Fprintf(c.w, "%s%s%s\n\n", c.col(colGray),
		strings.Repeat("─", 60), c.reset())
}

func (c *ConsoleReporter) ScenarioStarted(sc *ScenarioResult) {
	if c.verbose {
		fmt.Fprintf(c.w, "  %s%s%s  %s%s%s\n",
			c.col(colCyan), sc.FeatureName, c.reset(),
			c.bold(), sc.Name, c.reset())
	}
}

func (c *ConsoleReporter) StepFinished(_ *ScenarioResult, step *StepResult) {
	if !c.verbose {
		return
	}
	icon, col := stepIcon(step.Status)
	fmt.Fprintf(c.w, "    %s%s%s %s%s%s  %s%s%s\n",
		c.col(col), icon, c.reset(),
		c.col(colGray), step.Keyword+step.Text, c.reset(),
		c.col(colGray), formatDuration(step.Duration), c.reset())
	if step.Err != nil {
		fmt.Fprintf(c.w, "      %s%s%s\n", c.col(colRed), step.Err.Error(), c.reset())
	}
}

func (c *ConsoleReporter) ScenarioFinished(sc *ScenarioResult) {
	icon, col := scenarioIcon(sc.Status)
	tags := ""
	if len(sc.Tags) > 0 {
		tags = " " + c.col(colGray) + strings.Join(sc.Tags, " ") + c.reset()
	}
	fmt.Fprintf(c.w, "  %s%s%s  %s  %s%s%s  %s%s\n",
		c.col(col), icon, c.reset(),
		sc.Name,
		c.col(colGray), formatDuration(sc.Duration), c.reset(),
		tags, "")
	if sc.Err != nil && sc.Status == StatusFailed {
		fmt.Fprintf(c.w, "     %s↳ %s%s\n", c.col(colRed), sc.Err.Error(), c.reset())
	}
}

func (c *ConsoleReporter) RunFinished(r *RunResult) {
	fmt.Fprintf(c.w, "\n%s%s%s\n", c.col(colGray),
		strings.Repeat("─", 60), c.reset())

	statusCol := colGreen
	if r.Status == StatusFailed {
		statusCol = colRed
	}
	fmt.Fprintf(c.w, "%s%s %s%s  %s\n",
		c.bold(), c.col(statusCol), strings.ToUpper(r.Status.String()),
		c.reset(), formatDuration(r.Duration))

	fmt.Fprintf(c.w, "\n%sScenarios:%s  %s%d passed%s",
		c.col(colGray), c.reset(), c.col(colGreen), r.Passed, c.reset())
	if r.Failed > 0 {
		fmt.Fprintf(c.w, "  %s%d failed%s", c.col(colRed), r.Failed, c.reset())
	}
	if r.Skipped > 0 {
		fmt.Fprintf(c.w, "  %s%d skipped%s", c.col(colYellow), r.Skipped, c.reset())
	}
	if r.Undefined > 0 {
		fmt.Fprintf(c.w, "  %s%d undefined%s", c.col(colYellow), r.Undefined, c.reset())
	}
	fmt.Fprintf(c.w, "  %s(%d total)%s\n\n", c.col(colGray), r.Total, c.reset())

	// Print undefined step texts for fast triage.
	undefinedSteps := collectUndefinedSteps(r)
	if len(undefinedSteps) > 0 {
		fmt.Fprintf(c.w, "%sUndefined steps:%s\n", c.col(colYellow), c.reset())
		for _, s := range undefinedSteps {
			fmt.Fprintf(c.w, "  %s? %s%s\n", c.col(colYellow), s, c.reset())
		}
		fmt.Fprintln(c.w)
	}
}

// colour/style helpers

func (c *ConsoleReporter) col(code string) string {
	if !c.colour {
		return ""
	}
	return code
}

func (c *ConsoleReporter) reset() string {
	if !c.colour {
		return ""
	}
	return colReset
}

func (c *ConsoleReporter) bold() string {
	if !c.colour {
		return ""
	}
	return colBold
}

func stepIcon(s Status) (icon, colour string) {
	switch s {
	case StatusPassed:
		return "✓", colGreen
	case StatusFailed:
		return "✗", colRed
	case StatusSkipped:
		return "-", colGray
	case StatusUndefined:
		return "?", colYellow
	case StatusPending:
		return "…", colCyan
	default:
		return " ", colGray
	}
}

func scenarioIcon(s Status) (icon, colour string) {
	switch s {
	case StatusPassed:
		return "✓", colGreen
	case StatusFailed:
		return "✗", colRed
	case StatusSkipped:
		return "○", colGray
	case StatusUndefined:
		return "?", colYellow
	default:
		return " ", colGray
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d)/float64(time.Millisecond))
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func collectUndefinedSteps(r *RunResult) []string {
	seen := make(map[string]bool)
	var out []string
	for _, sc := range r.Scenarios {
		for _, step := range sc.Steps {
			if step.Status == StatusUndefined {
				key := step.Keyword + step.Text
				if !seen[key] {
					seen[key] = true
					out = append(out, key)
				}
			}
		}
	}
	return out
}

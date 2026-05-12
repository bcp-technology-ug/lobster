package parser

import (
	"fmt"
	"strings"
)

// Severity classifies a lint diagnostic.
type Severity int

const (
	SeverityWarning Severity = iota
	SeverityError
)

func (s Severity) String() string {
	if s == SeverityError {
		return "error"
	}
	return "warning"
}

// Diagnostic is a single linting issue found in a feature file.
type Diagnostic struct {
	Severity   Severity
	URI        string
	ScenarioID string // empty for feature-level issues
	Message    string
}

func (d Diagnostic) Error() string {
	if d.ScenarioID != "" {
		return fmt.Sprintf("%s [%s] scenario %s: %s", d.Severity, d.URI, d.ScenarioID, d.Message)
	}
	return fmt.Sprintf("%s [%s]: %s", d.Severity, d.URI, d.Message)
}

// LintResult holds all diagnostics for a set of features.
type LintResult struct {
	Diagnostics []Diagnostic
}

// HasErrors returns true if any diagnostic has SeverityError.
func (r *LintResult) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Error returns a summary error if there are any error-level diagnostics,
// or nil if all diagnostics are warnings or there are none.
func (r *LintResult) Error() error {
	if !r.HasErrors() {
		return nil
	}
	var msgs []string
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			msgs = append(msgs, d.Error())
		}
	}
	return fmt.Errorf("lint errors:\n%s", strings.Join(msgs, "\n"))
}

// Lint validates a slice of parsed features and returns a LintResult.
// All lint rules are applied to every feature. Errors are accumulated rather
// than failing fast, so callers see the full diagnostic picture.
func Lint(features []*Feature) *LintResult {
	r := &LintResult{}
	for _, f := range features {
		lintFeature(r, f)
	}
	return r
}

// lintFeature applies all rules to a single feature.
func lintFeature(r *LintResult, f *Feature) {
	if strings.TrimSpace(f.Name) == "" {
		r.Diagnostics = append(r.Diagnostics, Diagnostic{
			Severity: SeverityError,
			URI:      f.URI,
			Message:  "feature has no name",
		})
	}
	if len(f.Scenarios) == 0 {
		r.Diagnostics = append(r.Diagnostics, Diagnostic{
			Severity: SeverityWarning,
			URI:      f.URI,
			Message:  "feature has no scenarios",
		})
	}
	if f.Background != nil {
		lintBackground(r, f)
	}
	for _, sc := range f.Scenarios {
		lintScenario(r, f.URI, sc)
	}
}

func lintBackground(r *LintResult, f *Feature) {
	if len(f.Background.Steps) == 0 {
		r.Diagnostics = append(r.Diagnostics, Diagnostic{
			Severity: SeverityWarning,
			URI:      f.URI,
			Message:  "background has no steps",
		})
	}
	for i, step := range f.Background.Steps {
		if strings.TrimSpace(step.Text) == "" {
			r.Diagnostics = append(r.Diagnostics, Diagnostic{
				Severity: SeverityError,
				URI:      f.URI,
				Message:  fmt.Sprintf("background step %d has empty text", i+1),
			})
		}
	}
}

func lintScenario(r *LintResult, uri string, sc *Scenario) {
	if strings.TrimSpace(sc.Name) == "" {
		r.Diagnostics = append(r.Diagnostics, Diagnostic{
			Severity: SeverityError,
			URI:      uri,
			Message:  "scenario has no name",
		})
	}
	if len(sc.Steps) == 0 {
		r.Diagnostics = append(r.Diagnostics, Diagnostic{
			Severity:   SeverityError,
			URI:        uri,
			ScenarioID: sc.DeterministicID,
			Message:    fmt.Sprintf("scenario %q has no steps", sc.Name),
		})
	}
	for i, step := range sc.Steps {
		if strings.TrimSpace(step.Text) == "" {
			r.Diagnostics = append(r.Diagnostics, Diagnostic{
				Severity:   SeverityError,
				URI:        uri,
				ScenarioID: sc.DeterministicID,
				Message:    fmt.Sprintf("scenario %q step %d has empty text", sc.Name, i+1),
			})
		}
	}
	for _, tag := range sc.Tags {
		if !strings.HasPrefix(tag, "@") {
			r.Diagnostics = append(r.Diagnostics, Diagnostic{
				Severity:   SeverityError,
				URI:        uri,
				ScenarioID: sc.DeterministicID,
				Message:    fmt.Sprintf("tag %q must start with '@'", tag),
			})
		}
	}
}

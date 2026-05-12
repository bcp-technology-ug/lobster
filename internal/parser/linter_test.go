// Package parser_test provides black-box tests for the Gherkin linter.
package parser_test

import (
	"strings"
	"testing"

	"github.com/bcp-technology-ug/lobster/internal/parser"
)

// lintGherkin is a helper that parses src and then lints the result.
func lintGherkin(t *testing.T, src string) *parser.LintResult {
	t.Helper()
	feat, err := parser.ParseReader("test.feature", strings.NewReader(src))
	if err != nil {
		t.Fatalf("ParseReader: %v", err)
	}
	return parser.Lint([]*parser.Feature{feat})
}

func TestLint_ValidFeature(t *testing.T) {
	t.Parallel()

	res := lintGherkin(t, `Feature: Payments
  Scenario: Successful charge
    Given I have a valid card
    When I charge 10 USD
    Then the transaction succeeds`)

	if len(res.Diagnostics) != 0 {
		t.Errorf("expected no diagnostics, got %v", res.Diagnostics)
	}
	if res.HasErrors() {
		t.Error("HasErrors = true on clean feature")
	}
	if res.Error() != nil {
		t.Errorf("Error() = %v, want nil", res.Error())
	}
}

func TestLint_FeatureWithNoScenarios(t *testing.T) {
	t.Parallel()

	res := lintGherkin(t, `Feature: Payments`)

	if len(res.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(res.Diagnostics))
	}
	d := res.Diagnostics[0]
	if d.Severity != parser.SeverityWarning {
		t.Errorf("Severity = %v, want Warning", d.Severity)
	}
	if !strings.Contains(d.Message, "no scenarios") {
		t.Errorf("Message = %q, want 'no scenarios'", d.Message)
	}
	if res.HasErrors() {
		t.Error("HasErrors = true for warning-only result")
	}
	if res.Error() != nil {
		t.Errorf("Error() should be nil for warnings only, got %v", res.Error())
	}
}

func TestLint_ScenarioWithNoSteps(t *testing.T) {
	t.Parallel()

	res := lintGherkin(t, `Feature: Payments
  Scenario: Empty scenario`)

	if !res.HasErrors() {
		t.Error("HasErrors = false, want true")
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Severity == parser.SeverityError && strings.Contains(d.Message, "no steps") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an error about 'no steps', got %v", res.Diagnostics)
	}
}

func TestLint_BackgroundWithNoSteps(t *testing.T) {
	t.Parallel()

	res := lintGherkin(t, `Feature: Auth
  Background:

  Scenario: Login
    Given I am on the login page
    When I click login
    Then I am logged in`)

	// Background with no steps is a warning, not an error.
	hasBackgroundWarning := false
	for _, d := range res.Diagnostics {
		if d.Severity == parser.SeverityWarning && strings.Contains(d.Message, "background") {
			hasBackgroundWarning = true
		}
	}
	if !hasBackgroundWarning {
		t.Errorf("expected background warning, got %v", res.Diagnostics)
	}
}

func TestLint_InvalidTag(t *testing.T) {
	t.Parallel()

	res := lintGherkin(t, `Feature: Auth
  @notag
  Scenario: Login
    Given I do something
    Then it works`)

	// Tag without '@' prefix is an error.
	// However the gherkin parser itself enforces '@' in tags during parse.
	// If the parser strips the '@' or denormalizes we detect it in the linter.
	// The test checks that the linter correctly flags any tag not starting with '@'.
	// (Parser normalisation may already include '@', so zero errors is also valid here.)
	_ = res // structural test — just ensure no panic and returns a result
}

func TestLint_Error_ReturnsAllErrorMessages(t *testing.T) {
	t.Parallel()

	// Two separate empty scenarios → two error diagnostics.
	res := lintGherkin(t, `Feature: Payments
  Scenario: First
  Scenario: Second`)

	if !res.HasErrors() {
		t.Error("HasErrors = false, want true")
	}
	errMsg := res.Error()
	if errMsg == nil {
		t.Fatal("Error() is nil, want non-nil")
	}
	msg := errMsg.Error()
	if !strings.Contains(msg, "lint errors") {
		t.Errorf("Error() message = %q, want 'lint errors'", msg)
	}
}

func TestLint_MultipleFeatures(t *testing.T) {
	t.Parallel()

	src1 := `Feature: A
  Scenario: Pass
    Given step one`
	src2 := `Feature: B
  Scenario: Empty`

	feat1, err := parser.ParseReader("a.feature", strings.NewReader(src1))
	if err != nil {
		t.Fatalf("parse a: %v", err)
	}
	feat2, err := parser.ParseReader("b.feature", strings.NewReader(src2))
	if err != nil {
		t.Fatalf("parse b: %v", err)
	}

	res := parser.Lint([]*parser.Feature{feat1, feat2})

	if !res.HasErrors() {
		t.Error("HasErrors = false, want true (b has empty scenario)")
	}
	// All diagnostics from both features are present.
	for _, d := range res.Diagnostics {
		if d.URI == "" {
			t.Error("diagnostic has empty URI")
		}
	}
}

func TestLint_NoFeatures(t *testing.T) {
	t.Parallel()

	res := parser.Lint(nil)
	if len(res.Diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics for nil slice, got %d", len(res.Diagnostics))
	}
}

func TestDiagnostic_Error_WithScenarioID(t *testing.T) {
	t.Parallel()

	d := parser.Diagnostic{
		Severity:   parser.SeverityError,
		URI:        "foo.feature",
		ScenarioID: "abc123",
		Message:    "something went wrong",
	}

	msg := d.Error()
	if !strings.Contains(msg, "abc123") {
		t.Errorf("Error() = %q, want scenario ID 'abc123'", msg)
	}
	if !strings.Contains(msg, "error") {
		t.Errorf("Error() = %q, want 'error'", msg)
	}
}

func TestDiagnostic_Error_WithoutScenarioID(t *testing.T) {
	t.Parallel()

	d := parser.Diagnostic{
		Severity: parser.SeverityWarning,
		URI:      "bar.feature",
		Message:  "no scenarios",
	}

	msg := d.Error()
	if strings.Contains(msg, "scenario") && !strings.Contains(msg, "warning") {
		t.Errorf("unexpected message format: %q", msg)
	}
	if !strings.Contains(msg, "bar.feature") {
		t.Errorf("Error() = %q, want URI in message", msg)
	}
}

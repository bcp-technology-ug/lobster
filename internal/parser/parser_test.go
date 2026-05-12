package parser_test

import (
	"strings"
	"testing"

	"github.com/bcp-technology/lobster/internal/parser"
)

// helper returns a Feature or immediately fails the test.
func mustParse(t *testing.T, src string) *parser.Feature {
	t.Helper()
	feat, err := parser.ParseReader("test.feature", strings.NewReader(src))
	if err != nil {
		t.Fatalf("ParseReader: %v", err)
	}
	return feat
}

func TestParseReader_SimpleFeatureAndScenario(t *testing.T) {
	t.Parallel()

	src := `Feature: Payments
  Scenario: Successful charge
    Given I have a valid payment method
    When I charge 10 USD
    Then the transaction succeeds`

	feat := mustParse(t, src)

	if feat.Name != "Payments" {
		t.Errorf("Name = %q, want %q", feat.Name, "Payments")
	}
	if len(feat.Scenarios) != 1 {
		t.Fatalf("len(Scenarios) = %d, want 1", len(feat.Scenarios))
	}
	sc := feat.Scenarios[0]
	if sc.Name != "Successful charge" {
		t.Errorf("Scenario.Name = %q", sc.Name)
	}
	if len(sc.Steps) != 3 {
		t.Errorf("len(Steps) = %d, want 3", len(sc.Steps))
	}
}

func TestParseReader_Background(t *testing.T) {
	t.Parallel()

	src := `Feature: Auth
  Background:
    Given I am authenticated

  Scenario: View profile
    When I visit my profile
    Then I see my name`

	feat := mustParse(t, src)

	if feat.Background == nil {
		t.Fatal("Background is nil")
	}
	if len(feat.Background.Steps) != 1 {
		t.Errorf("Background.Steps = %d, want 1", len(feat.Background.Steps))
	}
	if want := "I am authenticated"; feat.Background.Steps[0].Text != want {
		t.Errorf("Background step text = %q, want %q", feat.Background.Steps[0].Text, want)
	}
	if len(feat.Scenarios) != 1 {
		t.Fatalf("len(Scenarios) = %d, want 1", len(feat.Scenarios))
	}
}

func TestParseReader_ScenarioOutlineExpandsRows(t *testing.T) {
	t.Parallel()

	src := `Feature: Checkout
  Scenario Outline: Buy item
    Given I have <qty> items
    When I checkout
    Then the total is <total>

    Examples:
      | qty | total |
      | 1   | 10    |
      | 2   | 20    |`

	feat := mustParse(t, src)

	// Each Examples row becomes a distinct Scenario.
	if len(feat.Scenarios) != 2 {
		t.Fatalf("len(Scenarios) = %d, want 2", len(feat.Scenarios))
	}
	// DeterministicIDs must differ.
	if feat.Scenarios[0].DeterministicID == feat.Scenarios[1].DeterministicID {
		t.Error("DeterministicIDs are identical for distinct outline rows")
	}
}

func TestParseReader_FeatureTags(t *testing.T) {
	t.Parallel()

	src := `@smoke @regression
Feature: Tagged
  @fast
  Scenario: Tagged scenario
    Given it runs`

	feat := mustParse(t, src)

	wantFeatureTags := []string{"@smoke", "@regression"}
	for _, want := range wantFeatureTags {
		found := false
		for _, got := range feat.Tags {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("feature tag %q not found in %v", want, feat.Tags)
		}
	}

	if len(feat.Scenarios) == 0 {
		t.Fatal("no scenarios")
	}
	sc := feat.Scenarios[0]
	hasTag := func(tag string) bool {
		for _, t := range sc.Tags {
			if t == tag {
				return true
			}
		}
		return false
	}
	if !hasTag("@fast") {
		t.Errorf("scenario should have @fast tag, got %v", sc.Tags)
	}
}

func TestParseReader_DeterministicID_StableAndUnique(t *testing.T) {
	t.Parallel()

	src := `Feature: IDs
  Scenario: Alpha
    Given step one
  Scenario: Beta
    Given step two`

	feat1 := mustParse(t, src)
	feat2 := mustParse(t, src)

	if len(feat1.Scenarios) != 2 {
		t.Fatalf("want 2 scenarios")
	}

	// Stable across multiple parses.
	for i, s1 := range feat1.Scenarios {
		s2 := feat2.Scenarios[i]
		if s1.DeterministicID != s2.DeterministicID {
			t.Errorf("scenario[%d] DeterministicID changed across parses: %q vs %q", i, s1.DeterministicID, s2.DeterministicID)
		}
	}

	// Unique per scenario.
	if feat1.Scenarios[0].DeterministicID == feat1.Scenarios[1].DeterministicID {
		t.Error("distinct scenarios share a DeterministicID")
	}
}

func TestParseReader_DocString(t *testing.T) {
	t.Parallel()

	src := "Feature: Body\n  Scenario: Post\n    Given I send a body:\n      \"\"\"\n      {\"key\":\"value\"}\n      \"\"\""

	feat := mustParse(t, src)
	if len(feat.Scenarios) == 0 {
		t.Fatal("no scenarios")
	}
	steps := feat.Scenarios[0].Steps
	if len(steps) == 0 {
		t.Fatal("no steps")
	}
	if steps[0].DocString == nil {
		t.Fatal("DocString is nil")
	}
	if !strings.Contains(steps[0].DocString.Content, "key") {
		t.Errorf("DocString.Content = %q", steps[0].DocString.Content)
	}
}

func TestParseReader_DataTable(t *testing.T) {
	t.Parallel()

	src := `Feature: Table
  Scenario: Table step
    Given the following users:
      | name  | role  |
      | Alice | admin |
      | Bob   | user  |`

	feat := mustParse(t, src)
	if len(feat.Scenarios) == 0 {
		t.Fatal("no scenarios")
	}
	steps := feat.Scenarios[0].Steps
	if len(steps) == 0 {
		t.Fatal("no steps")
	}
	dt := steps[0].DataTable
	if dt == nil {
		t.Fatal("DataTable is nil")
	}
	if len(dt.Rows) != 3 { // header + 2 data rows
		t.Errorf("DataTable rows = %d, want 3", len(dt.Rows))
	}
}

func TestParseReader_EmptyFeature(t *testing.T) {
	t.Parallel()

	src := `Feature: Empty`
	feat := mustParse(t, src)

	if feat.Name != "Empty" {
		t.Errorf("Name = %q", feat.Name)
	}
	if len(feat.Scenarios) != 0 {
		t.Errorf("len(Scenarios) = %d, want 0", len(feat.Scenarios))
	}
}

func TestParseReader_InvalidGherkin(t *testing.T) {
	t.Parallel()

	_, err := parser.ParseReader("bad.feature", strings.NewReader("this is not gherkin @@##"))
	if err == nil {
		t.Error("expected parse error for invalid Gherkin, got nil")
	}
}

func TestParseGlob_NoMatches(t *testing.T) {
	t.Parallel()

	features, err := parser.ParseGlob("testdata/nonexistent/**/*.feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(features) != 0 {
		t.Errorf("expected 0 features, got %d", len(features))
	}
}

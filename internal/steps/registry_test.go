package steps_test

import (
	"errors"
	"testing"

	"github.com/bcp-technology-ug/lobster/internal/parser"
	"github.com/bcp-technology-ug/lobster/internal/steps"
)

func noopHandler(_ *steps.ScenarioContext, _ ...string) error { return nil }

func TestRegistry_Match_Found(t *testing.T) {
	t.Parallel()

	r := steps.NewRegistry()
	if err := r.Register(`the response status should be (\d+)`, noopHandler, "test"); err != nil {
		t.Fatal(err)
	}

	def, args, err := r.Match("the response status should be 200")
	if err != nil {
		t.Fatalf("Match returned error: %v", err)
	}
	if def == nil {
		t.Fatal("expected a StepDef, got nil")
	}
	if len(args) != 1 || args[0] != "200" {
		t.Errorf("args = %v, want [\"200\"]", args)
	}
}

func TestRegistry_Match_Undefined(t *testing.T) {
	t.Parallel()

	r := steps.NewRegistry()
	_, _, err := r.Match("this step has no definition")
	if !errors.Is(err, steps.ErrUndefined) {
		t.Errorf("err = %v, want ErrUndefined", err)
	}
}

func TestRegistry_Match_Ambiguous(t *testing.T) {
	t.Parallel()

	r := steps.NewRegistry()
	_ = r.Register(`I .+ a request`, noopHandler, "a")
	_ = r.Register(`I send a request`, noopHandler, "b")

	_, _, err := r.Match("I send a request")
	if err == nil {
		t.Fatal("expected error for ambiguous match")
	}
	var ambig *steps.AmbiguousError
	if !errors.As(err, &ambig) {
		t.Errorf("err type = %T, want *steps.AmbiguousError", err)
	}
	if ambig.StepText != "I send a request" {
		t.Errorf("AmbiguousError.StepText = %q", ambig.StepText)
	}
	if len(ambig.Matches) != 2 {
		t.Errorf("AmbiguousError.Matches = %d, want 2", len(ambig.Matches))
	}
}

func TestRegistry_Register_InvalidPattern(t *testing.T) {
	t.Parallel()

	r := steps.NewRegistry()
	err := r.Register(`[invalid`, noopHandler, "test")
	if err == nil {
		t.Error("expected error for invalid regexp, got nil")
	}
}

func TestRegistry_Register_MultiplePatterns(t *testing.T) {
	t.Parallel()

	r := steps.NewRegistry()
	patterns := []string{
		`I do step one`,
		`I do step two`,
		`I do step three`,
	}
	for _, p := range patterns {
		if err := r.Register(p, noopHandler, "test"); err != nil {
			t.Fatalf("Register(%q): %v", p, err)
		}
	}
	for _, p := range patterns {
		_, _, err := r.Match(p)
		if err != nil {
			t.Errorf("Match(%q): %v", p, err)
		}
	}
}

func TestRegistry_MatchStep(t *testing.T) {
	t.Parallel()

	r := steps.NewRegistry()
	_ = r.Register(`the service "([^"]+)" is running`, noopHandler, "svc")

	step := &parser.Step{Text: `the service "api" is running`}
	def, args, err := r.MatchStep(step)
	if err != nil {
		t.Fatalf("MatchStep: %v", err)
	}
	if def == nil {
		t.Fatal("expected def, got nil")
	}
	if len(args) != 1 || args[0] != "api" {
		t.Errorf("args = %v, want [\"api\"]", args)
	}
}

func TestRegistry_CaptureGroups(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pattern string
		text    string
		args    []string
	}{
		{
			`I send a (GET|POST) request to "([^"]+)"`,
			`I send a POST request to "/users"`,
			[]string{"POST", "/users"},
		},
		{
			`the response status should be (\d+)`,
			`the response status should be 404`,
			[]string{"404"},
		},
		{
			`I wait up to (\d+)s? for the service "([^"]+)" to be running`,
			`I wait up to 30s for the service "db" to be running`,
			[]string{"30", "db"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.text, func(t *testing.T) {
			t.Parallel()

			r := steps.NewRegistry()
			_ = r.Register(tc.pattern, noopHandler, "test")
			_, got, err := r.Match(tc.text)
			if err != nil {
				t.Fatalf("Match: %v", err)
			}
			if len(got) != len(tc.args) {
				t.Fatalf("args length = %d, want %d", len(got), len(tc.args))
			}
			for i, want := range tc.args {
				if got[i] != want {
					t.Errorf("args[%d] = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

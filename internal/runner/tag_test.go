package runner

import "testing"

func TestMatchesTagExpression_Empty(t *testing.T) {
	t.Parallel()

	// Empty expression matches everything.
	if !matchesTagExpression(nil, "") {
		t.Error("empty expression should match nil tags")
	}
	if !matchesTagExpression([]string{"@smoke"}, "") {
		t.Error("empty expression should match any tags")
	}
}

func TestMatchesTagExpression_ExactMatch(t *testing.T) {
	t.Parallel()

	tags := []string{"@smoke", "@regression"}
	if !matchesTagExpression(tags, "@smoke") {
		t.Error("@smoke should match tags containing @smoke")
	}
	if matchesTagExpression(tags, "@wip") {
		t.Error("@wip should not match tags not containing @wip")
	}
}

func TestMatchesTagExpression_ORSemantics(t *testing.T) {
	t.Parallel()

	tags := []string{"@smoke"}
	// Comma separates OR terms.
	if !matchesTagExpression(tags, "@smoke,@wip") {
		t.Error("@smoke,@wip should match when @smoke is present")
	}
	if !matchesTagExpression(tags, "@wip,@smoke") {
		t.Error("@wip,@smoke should match when @smoke is present")
	}
	if matchesTagExpression(tags, "@wip,@slow") {
		t.Error("@wip,@slow should not match when neither tag is present")
	}
}

func TestMatchesTagExpression_ANDSemantics(t *testing.T) {
	t.Parallel()

	tags := []string{"@smoke", "@fast"}
	// Space separates AND terms within a comma-clause.
	if !matchesTagExpression(tags, "@smoke @fast") {
		t.Error("@smoke @fast should match when both tags present")
	}
	if matchesTagExpression(tags, "@smoke @slow") {
		t.Error("@smoke @slow should not match when @slow is absent")
	}
}

func TestMatchesTagExpression_Negation(t *testing.T) {
	t.Parallel()

	tags := []string{"@smoke"}
	if !matchesTagExpression(tags, "~@wip") {
		t.Error("~@wip should match when @wip is absent")
	}
	if matchesTagExpression(tags, "~@smoke") {
		t.Error("~@smoke should not match when @smoke is present")
	}
}

func TestMatchesTagExpression_Combined(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		tags  []string
		expr  string
		match bool
	}{
		{
			"OR with negation — first arm",
			[]string{"@smoke"},
			"@smoke,~@wip",
			true,
		},
		{
			"OR with negation — second arm",
			[]string{"@fast"},
			"@smoke,~@wip",
			true, // ~@wip matches because @wip is absent
		},
		{
			"OR with negation — both fail",
			[]string{"@wip"},
			"@smoke,~@wip",
			false, // @smoke absent, ~@wip fails because @wip present
		},
		{
			"AND with negation",
			[]string{"@smoke", "@fast"},
			"@smoke ~@slow",
			true,
		},
		{
			"AND where negation blocks match",
			[]string{"@smoke", "@slow"},
			"@smoke ~@slow",
			false,
		},
		{
			"multi-term OR, second clause matches",
			[]string{"@regression"},
			"@smoke,@regression",
			true,
		},
		{
			"tags empty, positive expression",
			nil,
			"@smoke",
			false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := matchesTagExpression(tc.tags, tc.expr)
			if got != tc.match {
				t.Errorf("matchesTagExpression(%v, %q) = %v, want %v", tc.tags, tc.expr, got, tc.match)
			}
		})
	}
}

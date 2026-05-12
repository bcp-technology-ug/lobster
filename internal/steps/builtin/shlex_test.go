package builtin

import (
	"testing"
)

func TestSplitArgs_simple(t *testing.T) {
	t.Parallel()
	got, err := splitArgs("run --config /tmp/cfg.yaml --tags smoke")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"run", "--config", "/tmp/cfg.yaml", "--tags", "smoke"}
	if len(got) != len(want) {
		t.Fatalf("len: got %d want %d (%v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("[%d]: got %q want %q", i, got[i], w)
		}
	}
}

func TestSplitArgs_empty(t *testing.T) {
	t.Parallel()
	got, err := splitArgs("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

func TestSplitArgs_singleQuote(t *testing.T) {
	t.Parallel()
	got, err := splitArgs("run 'hello world' end")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"run", "hello world", "end"}
	if len(got) != len(want) {
		t.Fatalf("len: got %d want %d (%v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("[%d]: got %q want %q", i, got[i], w)
		}
	}
}

func TestSplitArgs_doubleQuote(t *testing.T) {
	t.Parallel()
	got, err := splitArgs(`plan --tags "@smoke and @wip"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"plan", "--tags", "@smoke and @wip"}
	if len(got) != len(want) {
		t.Fatalf("len: got %d want %d (%v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("[%d]: got %q want %q", i, got[i], w)
		}
	}
}

func TestSplitArgs_doubleQuoteEscape(t *testing.T) {
	t.Parallel()
	got, err := splitArgs(`echo "say \"hello\""`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"echo", `say "hello"`}
	if len(got) != len(want) {
		t.Fatalf("len: got %d want %d (%v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("[%d]: got %q want %q", i, got[i], w)
		}
	}
}

func TestSplitArgs_whitespaceOnly(t *testing.T) {
	t.Parallel()
	got, err := splitArgs("   \t  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

func TestSplitArgs_multipleSpaces(t *testing.T) {
	t.Parallel()
	got, err := splitArgs("a  b   c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len: got %d want %d", len(got), len(want))
	}
}

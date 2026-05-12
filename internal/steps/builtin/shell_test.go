package builtin

import (
	"testing"

	"github.com/bcp-technology-ug/lobster/internal/steps"
)

func newShellCtx() *steps.ScenarioContext {
	return steps.NewScenarioContext("", nil, nil)
}

func TestShell_runCommand_success(t *testing.T) {
	t.Parallel()
	ctx := newShellCtx()
	if err := stepRunCommand(ctx, "echo hello"); err != nil {
		t.Fatalf("stepRunCommand: %v", err)
	}
	if ctx.Variables[varShellExitCode] != "0" {
		t.Errorf("exit code: got %q want %q", ctx.Variables[varShellExitCode], "0")
	}
}

func TestShell_runCommand_failureRecorded(t *testing.T) {
	t.Parallel()
	ctx := newShellCtx()
	if err := stepRunCommand(ctx, "exit 1"); err != nil {
		t.Fatalf("stepRunCommand should not return error for non-zero exit: %v", err)
	}
	if ctx.Variables[varShellExitCode] == "0" {
		t.Error("expected non-zero exit code to be recorded")
	}
}

func TestShell_assertExitCode_correct(t *testing.T) {
	t.Parallel()
	ctx := newShellCtx()
	_ = stepRunCommand(ctx, "echo hello")
	if err := stepAssertExitCode(ctx, "0"); err != nil {
		t.Fatalf("assertExitCode: %v", err)
	}
}

func TestShell_assertExitCode_wrong(t *testing.T) {
	t.Parallel()
	ctx := newShellCtx()
	_ = stepRunCommand(ctx, "echo hello")
	if err := stepAssertExitCode(ctx, "1"); err == nil {
		t.Error("expected assertion to fail for wrong exit code")
	}
}

func TestShell_assertExitCodeNot(t *testing.T) {
	t.Parallel()
	ctx := newShellCtx()
	_ = stepRunCommand(ctx, "echo hello")
	if err := stepAssertExitCodeNot(ctx, "1"); err != nil {
		t.Fatalf("assertExitCodeNot: %v", err)
	}
}

func TestShell_assertOutputContains(t *testing.T) {
	t.Parallel()
	ctx := newShellCtx()
	_ = stepRunCommand(ctx, "echo lobster-output")
	if err := stepAssertOutputContains(ctx, "lobster-output"); err != nil {
		t.Fatalf("assertOutputContains: %v", err)
	}
}

func TestShell_assertOutputNotContains(t *testing.T) {
	t.Parallel()
	ctx := newShellCtx()
	_ = stepRunCommand(ctx, "echo lobster-output")
	if err := stepAssertOutputNotContains(ctx, "no-such-text"); err != nil {
		t.Fatalf("assertOutputNotContains: %v", err)
	}
}

func TestShell_assertOutputNotContains_fails(t *testing.T) {
	t.Parallel()
	ctx := newShellCtx()
	_ = stepRunCommand(ctx, "echo lobster-output")
	if err := stepAssertOutputNotContains(ctx, "lobster-output"); err == nil {
		t.Error("expected assertion to fail when text is present")
	}
}

func TestShell_assertOutputIsJSON(t *testing.T) {
	t.Parallel()
	ctx := newShellCtx()
	_ = stepRunCommand(ctx, `printf '{"ok":true}'`)
	if err := stepAssertOutputIsJSON(ctx); err != nil {
		t.Fatalf("assertOutputIsJSON: %v", err)
	}
}

func TestShell_assertOutputIsJSON_fails(t *testing.T) {
	t.Parallel()
	ctx := newShellCtx()
	_ = stepRunCommand(ctx, "echo not-json")
	if err := stepAssertOutputIsJSON(ctx); err == nil {
		t.Error("expected assertOutputIsJSON to fail for non-JSON output")
	}
}

func TestShell_storeOutput(t *testing.T) {
	t.Parallel()
	ctx := newShellCtx()
	_ = stepRunCommand(ctx, "echo stored-val")
	if err := stepStoreOutput(ctx, "captured"); err != nil {
		t.Fatalf("stepStoreOutput: %v", err)
	}
	if got := ctx.Variables["captured"]; got != "stored-val" {
		t.Errorf("variable: got %q want %q", got, "stored-val")
	}
}

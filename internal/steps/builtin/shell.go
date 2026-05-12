package builtin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bcp-technology/lobster/internal/steps"
)

// unescapeArg decodes a step argument captured by a regex that allows backslash
// escapes. It converts \" → " and \\ → \.
func unescapeArg(s string) string {
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}

const srcShell = "builtin:shell"

// Variable keys written into ScenarioContext.Variables by shell steps.
// They are intentionally name-spaced with a double underscore to avoid
// collisions with user-defined variables.
const (
	varShellStdout   = "__shell_stdout"
	varShellStderr   = "__shell_stderr"
	varShellExitCode = "__shell_exit_code"
)

// lobsterBin returns the path to the lobster binary under test.
// It honours the LOBSTER_BIN environment variable so CI can point at a
// freshly-built binary without modifying PATH.
func lobsterBin() string {
	if v := os.Getenv("LOBSTER_BIN"); v != "" {
		return v
	}
	return "lobster"
}

func registerShellSteps(r *steps.Registry) error {
	defs := []struct {
		pattern string
		handler steps.StepHandler
	}{
		{
			`I run the command "((?:[^"\\]|\\.)*)"`,
			stepRunCommand,
		},
		{
			`I run lobster "((?:[^"\\]|\\.)*)"`,
			stepRunLobster,
		},
		{
			`the exit code should be (\d+)`,
			stepAssertExitCode,
		},
		{
			`the exit code should not be (\d+)`,
			stepAssertExitCodeNot,
		},
		{
			`the output should contain "((?:[^"\\]|\\.)*)"`,
			stepAssertOutputContains,
		},
		{
			`the output should not contain "((?:[^"\\]|\\.)*)"`,
			stepAssertOutputNotContains,
		},
		{
			`the stderr should contain "((?:[^"\\]|\\.)*)"`,
			stepAssertStderrContains,
		},
		{
			`the stderr should not contain "((?:[^"\\]|\\.)*)"`,
			stepAssertStderrNotContains,
		},
		{
			`the output should be valid JSON`,
			stepAssertOutputIsJSON,
		},
		{
			`I store the output in variable "([^"]+)"`,
			stepStoreOutput,
		},
	}
	for _, d := range defs {
		if err := r.Register(d.pattern, d.handler, srcShell); err != nil {
			return err
		}
	}
	return nil
}

// stepRunCommand handles: I run the command "..."
// The argument is passed verbatim to sh -c so full shell syntax is available.
// ScenarioContext variables are injected into the subprocess environment so
// that values stored with "I store the output in variable" are accessible as
// shell variables.
func stepRunCommand(ctx *steps.ScenarioContext, args ...string) error {
	cmd := exec.Command("sh", "-c", unescapeArg(args[0])) //nolint:gosec // deliberate shell step
	cmd.Env = os.Environ()
	// Inject suite-scoped variables first so scenario-scoped ones take precedence.
	for k, v := range ctx.SuiteVars {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	for k, v := range ctx.Variables {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	return runAndCapture(ctx, cmd)
}

// stepRunLobster handles: I run lobster "..."
// The argument string is split with the shlex splitter so quoted flags work.
// When the scenario has changed into a temporary directory (varWorkDir is set),
// this function injects LOBSTER_PERSISTENCE_MIGRATIONS_DIR into the subprocess
// environment so sub-lobster invocations can find the schema files.
// Commands that don't use persistence (init, lint, validate) ignore the var.
func stepRunLobster(ctx *steps.ScenarioContext, args ...string) error {
	// Expand ${VAR} references from the scenario variable map before splitting.
	// Unlike stepRunCommand (which uses sh -c), this step builds an exec.Cmd
	// directly, so shell variable expansion is not automatic.
	rawArgs := os.Expand(unescapeArg(args[0]), func(key string) string {
		if v, ok := ctx.Variables[key]; ok {
			return v
		}
		if v, ok := ctx.SuiteVars[key]; ok {
			return v
		}
		return "${" + key + "}"
	})
	parts, err := splitArgs(rawArgs)
	if err != nil {
		return fmt.Errorf("parse lobster arguments: %w", err)
	}
	cmd := exec.Command(lobsterBin(), parts...) //nolint:gosec // deliberate exec step
	if orig := ctx.Variables[varWorkDir]; orig != "" {
		// Look for migrations/ in the original dir (running from repo root) and
		// also one level up (running from a subdirectory like tests/).
		candidates := []string{
			filepath.Join(orig, "migrations"),
			filepath.Join(orig, "..", "migrations"),
		}
		for _, migrationsAbs := range candidates {
			if abs, absErr := filepath.Abs(migrationsAbs); absErr == nil {
				if _, statErr := os.Stat(abs); statErr == nil {
					cmd.Env = append(os.Environ(), "LOBSTER_PERSISTENCE_MIGRATIONS_DIR="+abs)
					break
				}
			}
		}
	}
	return runAndCapture(ctx, cmd)
}

// runAndCapture executes cmd, captures stdout/stderr and exit code into
// ScenarioContext.Variables. It never returns an error for non-zero exits —
// callers assert the exit code separately.
func runAndCapture(ctx *steps.ScenarioContext, cmd *exec.Cmd) error {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	code := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			// Binary not found or other OS-level error — surface it immediately.
			return fmt.Errorf("exec: %w", runErr)
		}
	}

	ctx.Variables[varShellStdout] = stdout.String()
	ctx.Variables[varShellStderr] = stderr.String()
	ctx.Variables[varShellExitCode] = strconv.Itoa(code)
	return nil
}

func stepAssertExitCode(ctx *steps.ScenarioContext, args ...string) error {
	want, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid exit code %q", args[0])
	}
	got, _ := strconv.Atoi(ctx.Variables[varShellExitCode])
	if got != want {
		e := fmt.Errorf("expected exit code %d but got %d\nstdout: %s\nstderr: %s",
			want, got, ctx.Variables[varShellStdout], ctx.Variables[varShellStderr])
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepAssertExitCodeNot(ctx *steps.ScenarioContext, args ...string) error {
	notWant, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid exit code %q", args[0])
	}
	got, _ := strconv.Atoi(ctx.Variables[varShellExitCode])
	if got == notWant {
		e := fmt.Errorf("expected exit code not to be %d", notWant)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepAssertOutputContains(ctx *steps.ScenarioContext, args ...string) error {
	needle := unescapeArg(args[0])
	haystack := ctx.Variables[varShellStdout]
	if !strings.Contains(haystack, needle) {
		e := fmt.Errorf("output does not contain %q\noutput: %s", needle, haystack)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepAssertOutputNotContains(ctx *steps.ScenarioContext, args ...string) error {
	needle := unescapeArg(args[0])
	haystack := ctx.Variables[varShellStdout]
	if strings.Contains(haystack, needle) {
		e := fmt.Errorf("output unexpectedly contains %q", needle)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepAssertStderrContains(ctx *steps.ScenarioContext, args ...string) error {
	needle := unescapeArg(args[0])
	haystack := ctx.Variables[varShellStderr]
	if !strings.Contains(haystack, needle) {
		e := fmt.Errorf("stderr does not contain %q\nstderr: %s", needle, haystack)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepAssertStderrNotContains(ctx *steps.ScenarioContext, args ...string) error {
	needle := unescapeArg(args[0])
	haystack := ctx.Variables[varShellStderr]
	if strings.Contains(haystack, needle) {
		e := fmt.Errorf("stderr unexpectedly contains %q", needle)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepAssertOutputIsJSON(ctx *steps.ScenarioContext, _ ...string) error {
	out := ctx.Variables[varShellStdout]
	if !json.Valid([]byte(out)) {
		e := fmt.Errorf("output is not valid JSON:\n%s", out)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepStoreOutput(ctx *steps.ScenarioContext, args ...string) error {
	ctx.Variables[args[0]] = strings.TrimRight(ctx.Variables[varShellStdout], "\n")
	return nil
}

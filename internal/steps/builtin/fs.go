package builtin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bcp-technology-ug/lobster/internal/steps"
)

const srcFS = "builtin:fs"

// Variable keys written into ScenarioContext.Variables by fs steps.
const (
	varTmpDir  = "__tmpdir"
	varWorkDir = "__workdir"
)

func registerFSSteps(r *steps.Registry) error {
	defs := []struct {
		pattern string
		handler steps.StepHandler
	}{
		{
			`I am in a new temporary directory`,
			stepNewTempDir,
		},
		{
			`the file "((?:[^"\\]|\\.)+)" should exist`,
			stepFileExists,
		},
		{
			`the file "((?:[^"\\]|\\.)+)" should not exist`,
			stepFileNotExists,
		},
		{
			`the directory "((?:[^"\\]|\\.)+)" should exist`,
			stepDirExists,
		},
		{
			`the directory "((?:[^"\\]|\\.)+)" should not exist`,
			stepDirNotExists,
		},
		{
			`the file "((?:[^"\\]|\\.)+)" should contain "((?:[^"\\]|\\.)+)"`,
			stepFileContains,
		},
		{
			`I create the file "((?:[^"\\]|\\.)+)" with content:`,
			stepCreateFileDocString,
		},
		{
			`I create the file "((?:[^"\\]|\\.)+)" containing "((?:[^"\\]|\\.)+)"`,
			stepCreateFileInline,
		},
		{
			`the file "((?:[^"\\]|\\.)+)" should contain valid JSON`,
			stepFileContainsValidJSON,
		},
	}
	for _, d := range defs {
		if err := r.Register(d.pattern, d.handler, srcFS); err != nil {
			return err
		}
	}
	return nil
}

// stepNewTempDir creates a temporary directory and changes the process working
// directory into it. The original working directory and the temp path are
// stored in ctx.Variables so the AfterScenario hook can clean up.
func stepNewTempDir(ctx *steps.ScenarioContext, _ ...string) error {
	orig, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	tmp, err := os.MkdirTemp("", "lobster-test-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}

	if err := os.Chdir(tmp); err != nil {
		_ = os.RemoveAll(tmp)
		return fmt.Errorf("chdir to temp dir: %w", err)
	}

	ctx.Variables[varWorkDir] = orig
	ctx.Variables[varTmpDir] = tmp
	return nil
}

func stepFileExists(ctx *steps.ScenarioContext, args ...string) error {
	info, err := os.Stat(args[0])
	if err != nil || info.IsDir() {
		e := fmt.Errorf("expected file %q to exist", args[0])
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepFileNotExists(ctx *steps.ScenarioContext, args ...string) error {
	info, err := os.Stat(args[0])
	if err == nil && !info.IsDir() {
		e := fmt.Errorf("expected file %q not to exist", args[0])
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepDirExists(ctx *steps.ScenarioContext, args ...string) error {
	info, err := os.Stat(args[0])
	if err != nil || !info.IsDir() {
		e := fmt.Errorf("expected directory %q to exist", args[0])
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepDirNotExists(ctx *steps.ScenarioContext, args ...string) error {
	info, err := os.Stat(args[0])
	if err == nil && info.IsDir() {
		e := fmt.Errorf("expected directory %q not to exist", args[0])
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepFileContains(ctx *steps.ScenarioContext, args ...string) error {
	path := args[0]
	needle := unescapeArg(args[1])
	data, err := os.ReadFile(path) //nolint:gosec // test helper — path comes from feature file
	if err != nil {
		return fmt.Errorf("read file %q: %w", path, err)
	}
	if !strings.Contains(string(data), needle) {
		e := fmt.Errorf("file %q does not contain %q", path, needle)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

// stepCreateFileDocString handles: I create the file "path" with content: (DocString)
func stepCreateFileDocString(ctx *steps.ScenarioContext, args ...string) error {
	path := args[0]
	var content string
	if ctx.CurrentStep != nil && ctx.CurrentStep.DocString != nil {
		content = ctx.CurrentStep.DocString.Content
	}
	return writeFile(path, content)
}

// stepCreateFileInline handles: I create the file "path" containing "content"
func stepCreateFileInline(_ *steps.ScenarioContext, args ...string) error {
	return writeFile(args[0], args[1])
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directories for %q: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { //nolint:gosec // test helper
		return fmt.Errorf("write file %q: %w", path, err)
	}
	return nil
}

// stepFileContainsValidJSON handles: the file "path" should contain valid JSON
func stepFileContainsValidJSON(ctx *steps.ScenarioContext, args ...string) error {
	path := args[0]
	data, err := os.ReadFile(path) //nolint:gosec // test helper
	if err != nil {
		return fmt.Errorf("read file %q: %w", path, err)
	}
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		e := fmt.Errorf("file %q does not contain valid JSON: %w", path, err)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

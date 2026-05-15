package builtin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	yaml "go.yaml.in/yaml/v3"

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
		// Extended FS steps
		{
			`I read the file "((?:[^"\\]|\\.)+)" into variable "([^"]+)"`,
			stepReadFileIntoVar,
		},
		{
			`the file "((?:[^"\\]|\\.)+)" content should match "((?:[^"\\]|\\.)+)"`,
			stepFileContentMatches,
		},
		{
			`the file "((?:[^"\\]|\\.)+)" content should not match "((?:[^"\\]|\\.)+)"`,
			stepFileContentNotMatches,
		},
		{
			`the file "((?:[^"\\]|\\.)+)" content should equal:`,
			stepFileContentEquals,
		},
		{
			`the file "((?:[^"\\]|\\.)+)" should contain valid YAML`,
			stepFileContainsValidYAML,
		},
		{
			`the JSON file "((?:[^"\\]|\\.)+)" field "([^"]+)" should equal "([^"]+)"`,
			stepJSONFileFieldEquals,
		},
		{
			`I append to file "((?:[^"\\]|\\.)+)" with content:`,
			stepAppendToFile,
		},
		{
			`I delete the file "((?:[^"\\]|\\.)+)"`,
			stepDeleteFile,
		},
		{
			`the directory "((?:[^"\\]|\\.)+)" should contain "([^"]+)"`,
			stepDirContains,
		},
		{
			`the directory "((?:[^"\\]|\\.)+)" should not contain "([^"]+)"`,
			stepDirNotContains,
		},
		{
			`the file "((?:[^"\\]|\\.)+)" should have size less than (\d+) bytes`,
			stepFileSizeLessThan,
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
	return writeFile(args[0], unescapeArg(args[1]))
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

// stepReadFileIntoVar handles: I read the file "PATH" into variable "NAME"
func stepReadFileIntoVar(ctx *steps.ScenarioContext, args ...string) error {
	path := unescapeArg(args[0])
	varName := args[1]
	data, err := os.ReadFile(path) //nolint:gosec // test helper
	if err != nil {
		return fmt.Errorf("read file %q: %w", path, err)
	}
	ctx.Variables[varName] = string(data)
	return nil
}

// stepFileContentMatches handles: the file "PATH" content should match "REGEX"
func stepFileContentMatches(ctx *steps.ScenarioContext, args ...string) error {
	path := unescapeArg(args[0])
	pattern := unescapeArg(args[1])
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex %q: %w", pattern, err)
	}
	data, err := os.ReadFile(path) //nolint:gosec // test helper
	if err != nil {
		return fmt.Errorf("read file %q: %w", path, err)
	}
	if !re.Match(data) {
		e := fmt.Errorf("file %q content does not match regex %q", path, pattern)
		return softOrHard(ctx, e)
	}
	return nil
}

// stepFileContentNotMatches handles: the file "PATH" content should not match "REGEX"
func stepFileContentNotMatches(ctx *steps.ScenarioContext, args ...string) error {
	path := unescapeArg(args[0])
	pattern := unescapeArg(args[1])
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex %q: %w", pattern, err)
	}
	data, err := os.ReadFile(path) //nolint:gosec // test helper
	if err != nil {
		return fmt.Errorf("read file %q: %w", path, err)
	}
	if re.Match(data) {
		e := fmt.Errorf("file %q content unexpectedly matches regex %q", path, pattern)
		return softOrHard(ctx, e)
	}
	return nil
}

// stepFileContentEquals handles: the file "PATH" content should equal: (DocString)
func stepFileContentEquals(ctx *steps.ScenarioContext, args ...string) error {
	path := unescapeArg(args[0])
	var want string
	if ctx.CurrentStep != nil && ctx.CurrentStep.DocString != nil {
		want = ctx.CurrentStep.DocString.Content
	}
	data, err := os.ReadFile(path) //nolint:gosec // test helper
	if err != nil {
		return fmt.Errorf("read file %q: %w", path, err)
	}
	got := string(data)
	if got != want {
		e := fmt.Errorf("file %q content does not equal expected value\ngot:\n%s\nwant:\n%s", path, got, want)
		return softOrHard(ctx, e)
	}
	return nil
}

// stepFileContainsValidYAML handles: the file "PATH" should contain valid YAML
func stepFileContainsValidYAML(ctx *steps.ScenarioContext, args ...string) error {
	path := unescapeArg(args[0])
	data, err := os.ReadFile(path) //nolint:gosec // test helper
	if err != nil {
		return fmt.Errorf("read file %q: %w", path, err)
	}
	var v interface{}
	if err := yaml.Unmarshal(data, &v); err != nil {
		e := fmt.Errorf("file %q does not contain valid YAML: %w", path, err)
		return softOrHard(ctx, e)
	}
	return nil
}

// stepJSONFileFieldEquals handles: the JSON file "PATH" field "JSONPATH" should equal "VALUE"
func stepJSONFileFieldEquals(ctx *steps.ScenarioContext, args ...string) error {
	path := unescapeArg(args[0])
	jsonPathExpr := args[1]
	want := args[2]

	data, err := os.ReadFile(path) //nolint:gosec // test helper
	if err != nil {
		return fmt.Errorf("read file %q: %w", path, err)
	}
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("file %q is not valid JSON: %w", path, err)
	}
	val, ok := jsonPath(obj, jsonPathExpr)
	if !ok {
		e := fmt.Errorf("JSON file %q: field %q not found", path, jsonPathExpr)
		return softOrHard(ctx, e)
	}
	got := fmt.Sprintf("%v", val)
	if got != want {
		e := fmt.Errorf("JSON file %q field %q: expected %q but got %q", path, jsonPathExpr, want, got)
		return softOrHard(ctx, e)
	}
	return nil
}

// stepAppendToFile handles: I append to file "PATH" with content: (DocString)
func stepAppendToFile(ctx *steps.ScenarioContext, args ...string) error {
	path := unescapeArg(args[0])
	var content string
	if ctx.CurrentStep != nil && ctx.CurrentStep.DocString != nil {
		content = ctx.CurrentStep.DocString.Content
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directories for %q: %w", path, err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) //nolint:gosec // test helper
	if err != nil {
		return fmt.Errorf("open file %q for append: %w", path, err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("append to file %q: %w", path, err)
	}
	return nil
}

// stepDeleteFile handles: I delete the file "PATH"
func stepDeleteFile(_ *steps.ScenarioContext, args ...string) error {
	path := unescapeArg(args[0])
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file %q: %w", path, err)
	}
	return nil
}

// stepDirContains handles: the directory "PATH" should contain "FILENAME"
func stepDirContains(ctx *steps.ScenarioContext, args ...string) error {
	dir := unescapeArg(args[0])
	name := args[1]
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read directory %q: %w", dir, err)
	}
	for _, e := range entries {
		if e.Name() == name {
			return nil
		}
	}
	e := fmt.Errorf("directory %q does not contain %q", dir, name)
	return softOrHard(ctx, e)
}

// stepDirNotContains handles: the directory "PATH" should not contain "FILENAME"
func stepDirNotContains(ctx *steps.ScenarioContext, args ...string) error {
	dir := unescapeArg(args[0])
	name := args[1]
	entries, err := os.ReadDir(dir)
	if err != nil {
		// If the directory doesn't exist, it clearly doesn't contain the file.
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read directory %q: %w", dir, err)
	}
	for _, e := range entries {
		if e.Name() == name {
			e2 := fmt.Errorf("directory %q unexpectedly contains %q", dir, name)
			return softOrHard(ctx, e2)
		}
	}
	return nil
}

// stepFileSizeLessThan handles: the file "PATH" should have size less than N bytes
func stepFileSizeLessThan(ctx *steps.ScenarioContext, args ...string) error {
	path := unescapeArg(args[0])
	var limit int64
	if _, err := fmt.Sscanf(args[1], "%d", &limit); err != nil {
		return fmt.Errorf("invalid byte count %q", args[1])
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat file %q: %w", path, err)
	}
	if info.Size() >= limit {
		e := fmt.Errorf("file %q size %d bytes is not less than %d bytes", path, info.Size(), limit)
		return softOrHard(ctx, e)
	}
	return nil
}

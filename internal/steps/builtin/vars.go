package builtin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bcp-technology-ug/lobster/internal/steps"
)

const srcVars = "builtin:vars"

func registerVarSteps(r *steps.Registry) error {
	defs := []struct {
		pattern string
		handler steps.StepHandler
	}{
		// Setters
		{
			`I set variable "([^"]+)" to "([^"]+)"`,
			stepSetVariable,
		},
		{
			`I set variable "([^"]+)" to a random UUID`,
			stepSetVariableUUID,
		},
		{
			`I set variable "([^"]+)" to the current Unix timestamp`,
			stepSetVariableTimestamp,
		},
		{
			`I set variable "([^"]+)" from JSON field "([^"]+)" in the response`,
			stepSetVariableFromJSONField,
		},
		{
			`I store JSON field "([^"]+)" from the response in variable "([^"]+)"`,
			stepStoreJSONFieldInVariable,
		},
		{
			`I store the response header "([^"]+)" in variable "([^"]+)"`,
			stepStoreResponseHeaderInVariable,
		},
		{
			`I clear variable "([^"]+)"`,
			stepClearVariable,
		},
		// Assertions
		{
			`the variable "([^"]+)" should equal "([^"]+)"`,
			stepAssertVariableEquals,
		},
		{
			`the variable "([^"]+)" should not equal "([^"]+)"`,
			stepAssertVariableNotEquals,
		},
		{
			`the variable "([^"]+)" should contain "([^"]+)"`,
			stepAssertVariableContains,
		},
		{
			`the variable "([^"]+)" should not contain "([^"]+)"`,
			stepAssertVariableNotContains,
		},
		{
			`the variable "([^"]+)" should match "([^"]+)"`,
			stepAssertVariableMatches,
		},
		{
			`the variable "([^"]+)" should not be empty`,
			stepAssertVariableNotEmpty,
		},
	}
	for _, d := range defs {
		if err := r.Register(d.pattern, d.handler, srcVars); err != nil {
			return err
		}
	}
	return nil
}

func stepSetVariable(ctx *steps.ScenarioContext, args ...string) error {
	ctx.Variables[args[0]] = args[1]
	return nil
}

func stepSetVariableUUID(ctx *steps.ScenarioContext, args ...string) error {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Errorf("generate UUID: %w", err)
	}
	// Format as xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx (v4 UUID).
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	uuid := fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:]),
	)
	ctx.Variables[args[0]] = uuid
	return nil
}

func stepSetVariableTimestamp(ctx *steps.ScenarioContext, args ...string) error {
	ctx.Variables[args[0]] = strconv.FormatInt(time.Now().Unix(), 10)
	return nil
}

// stepSetVariableFromJSONField handles:
//
//	I set variable "NAME" from JSON field "PATH" in the response
func stepSetVariableFromJSONField(ctx *steps.ScenarioContext, args ...string) error {
	varName := args[0]
	path := args[1]
	return extractJSONFieldToVar(ctx, path, varName)
}

// stepStoreJSONFieldInVariable handles (ergonomic alias):
//
//	I store JSON field "PATH" from the response in variable "NAME"
func stepStoreJSONFieldInVariable(ctx *steps.ScenarioContext, args ...string) error {
	path := args[0]
	varName := args[1]
	return extractJSONFieldToVar(ctx, path, varName)
}

func extractJSONFieldToVar(ctx *steps.ScenarioContext, path, varName string) error {
	obj, err := parseBody(ctx)
	if err != nil {
		return err
	}
	val, ok := jsonPath(obj, path)
	if !ok {
		return fmt.Errorf("JSON field %q not found in response", path)
	}
	ctx.Variables[varName] = fmt.Sprintf("%v", val)
	return nil
}

// stepStoreResponseHeaderInVariable handles:
//
//	I store the response header "KEY" in variable "NAME"
func stepStoreResponseHeaderInVariable(ctx *steps.ScenarioContext, args ...string) error {
	key := args[0]
	varName := args[1]
	if ctx.LastResponse == nil {
		return fmt.Errorf("no HTTP response received")
	}
	ctx.Variables[varName] = ctx.LastResponse.Header.Get(key)
	return nil
}

func stepClearVariable(ctx *steps.ScenarioContext, args ...string) error {
	delete(ctx.Variables, args[0])
	return nil
}

func stepAssertVariableEquals(ctx *steps.ScenarioContext, args ...string) error {
	name := args[0]
	want := args[1]
	got, exists := ctx.Variables[name]
	if !exists {
		e := fmt.Errorf("variable %q is not set", name)
		return softOrHard(ctx, e)
	}
	if got != want {
		e := fmt.Errorf("variable %q: expected %q but got %q", name, want, got)
		return softOrHard(ctx, e)
	}
	return nil
}

func stepAssertVariableNotEquals(ctx *steps.ScenarioContext, args ...string) error {
	name := args[0]
	notWant := args[1]
	got := ctx.Variables[name]
	if got == notWant {
		e := fmt.Errorf("variable %q: expected not to equal %q", name, notWant)
		return softOrHard(ctx, e)
	}
	return nil
}

func stepAssertVariableContains(ctx *steps.ScenarioContext, args ...string) error {
	name := args[0]
	substr := args[1]
	got, exists := ctx.Variables[name]
	if !exists {
		e := fmt.Errorf("variable %q is not set", name)
		return softOrHard(ctx, e)
	}
	if !strings.Contains(got, substr) {
		e := fmt.Errorf("variable %q (%q) does not contain %q", name, got, substr)
		return softOrHard(ctx, e)
	}
	return nil
}

func stepAssertVariableNotContains(ctx *steps.ScenarioContext, args ...string) error {
	name := args[0]
	substr := args[1]
	got := ctx.Variables[name]
	if strings.Contains(got, substr) {
		e := fmt.Errorf("variable %q (%q) unexpectedly contains %q", name, got, substr)
		return softOrHard(ctx, e)
	}
	return nil
}

func stepAssertVariableMatches(ctx *steps.ScenarioContext, args ...string) error {
	name := args[0]
	pattern := args[1]
	got, exists := ctx.Variables[name]
	if !exists {
		e := fmt.Errorf("variable %q is not set", name)
		return softOrHard(ctx, e)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex %q: %w", pattern, err)
	}
	if !re.MatchString(got) {
		e := fmt.Errorf("variable %q (%q) does not match regex %q", name, got, pattern)
		return softOrHard(ctx, e)
	}
	return nil
}

func stepAssertVariableNotEmpty(ctx *steps.ScenarioContext, args ...string) error {
	name := args[0]
	got := ctx.Variables[name]
	if strings.TrimSpace(got) == "" {
		e := fmt.Errorf("variable %q is empty or not set", name)
		return softOrHard(ctx, e)
	}
	return nil
}

// softOrHard is a helper that either records an assertion error (soft mode) or
// returns it immediately (hard mode).
func softOrHard(ctx *steps.ScenarioContext, err error) error {
	if ctx.SoftAssertMode {
		ctx.AddAssertionError(err)
		return nil
	}
	return err
}

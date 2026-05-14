package builtin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/bcp-technology-ug/lobster/internal/steps"
)

// srcAssert is the source label for JSON assertion steps.
const srcAssert = "builtin:assert"

func registerAssertSteps(r *steps.Registry) error {
	defs := []struct {
		pattern string
		handler steps.StepHandler
	}{
		// JSON field equality: the response JSON field "$.key" should equal "value"
		{
			`the response JSON field "([^"]+)" should equal "([^"]+)"`,
			stepAssertJSONFieldEquals,
		},
		// JSON field existence: the response JSON field "$.key" should exist
		{
			`the response JSON field "([^"]+)" should exist`,
			stepAssertJSONFieldExists,
		},
		// JSON field absence: the response JSON field "$.key" should not exist
		{
			`the response JSON field "([^"]+)" should not exist`,
			stepAssertJSONFieldNotExists,
		},
		// JSON array length: the response JSON array "$.key" should have length N
		{
			`the response JSON array "([^"]+)" should have length (\d+)`,
			stepAssertJSONArrayLength,
		},
		// JSON field contains substring: the response JSON field "$.key" should contain "substr"
		{
			`the response JSON field "([^"]+)" should contain "([^"]+)"`,
			stepAssertJSONFieldContains,
		},
		// JSON field numeric equality: the response JSON field "$.key" should equal numeric N
		{
			`the response JSON field "([^"]+)" should equal numeric (\d+(?:\.\d+)?)`,
			stepAssertJSONFieldEqualsNumeric,
		},
		// JSON array non-empty: the response JSON array "$.key" should not be empty
		{
			`the response JSON array "([^"]+)" should not be empty`,
			stepAssertJSONArrayNotEmpty,
		},
		// JSON array empty: the response JSON array "$.key" should be empty
		{
			`the response JSON array "([^"]+)" should be empty`,
			stepAssertJSONArrayEmpty,
		},
	}
	for _, d := range defs {
		if err := r.Register(d.pattern, d.handler, srcAssert); err != nil {
			return err
		}
	}
	return nil
}

// jsonPath resolves a simple dot-separated JSON path against a parsed map.
// Supports keys like "plans", "items.0.id", "plan.plan_id".
// Only supports map and array access; does not implement full JSONPath.
func jsonPath(obj interface{}, path string) (interface{}, bool) {
	parts := strings.SplitN(path, ".", 2)
	key := parts[0]

	switch node := obj.(type) {
	case map[string]interface{}:
		val, ok := node[key]
		if !ok {
			return nil, false
		}
		if len(parts) == 1 {
			return val, true
		}
		return jsonPath(val, parts[1])
	case []interface{}:
		idx, err := strconv.Atoi(key)
		if err != nil || idx < 0 || idx >= len(node) {
			return nil, false
		}
		if len(parts) == 1 {
			return node[idx], true
		}
		return jsonPath(node[idx], parts[1])
	default:
		return nil, false
	}
}

// parseBody parses the last response body as a JSON object.
func parseBody(ctx *steps.ScenarioContext) (interface{}, error) {
	if len(ctx.LastBody) == 0 {
		return nil, fmt.Errorf("no response body available")
	}
	var obj interface{}
	if err := json.Unmarshal(ctx.LastBody, &obj); err != nil {
		return nil, fmt.Errorf("response body is not valid JSON: %w", err)
	}
	return obj, nil
}

func stepAssertJSONFieldEquals(ctx *steps.ScenarioContext, args ...string) error {
	path := args[0]
	want := args[1]

	obj, err := parseBody(ctx)
	if err != nil {
		return err
	}
	val, ok := jsonPath(obj, path)
	if !ok {
		e := fmt.Errorf("JSON field %q not found in response", path)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	got := fmt.Sprintf("%v", val)
	if got != want {
		e := fmt.Errorf("JSON field %q: expected %q but got %q", path, want, got)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepAssertJSONFieldExists(ctx *steps.ScenarioContext, args ...string) error {
	path := args[0]
	obj, err := parseBody(ctx)
	if err != nil {
		return err
	}
	if _, ok := jsonPath(obj, path); !ok {
		e := fmt.Errorf("JSON field %q not found in response", path)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepAssertJSONFieldNotExists(ctx *steps.ScenarioContext, args ...string) error {
	path := args[0]
	obj, err := parseBody(ctx)
	if err != nil {
		return err
	}
	if _, ok := jsonPath(obj, path); ok {
		e := fmt.Errorf("JSON field %q was present in response but should not be", path)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepAssertJSONArrayLength(ctx *steps.ScenarioContext, args ...string) error {
	path := args[0]
	wantLen, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid length %q", args[1])
	}

	obj, parseErr := parseBody(ctx)
	if parseErr != nil {
		return parseErr
	}

	val, ok := jsonPath(obj, path)
	if !ok {
		e := fmt.Errorf("JSON field %q not found in response", path)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}

	arr, ok := val.([]interface{})
	if !ok {
		e := fmt.Errorf("JSON field %q is not an array", path)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}

	if len(arr) != wantLen {
		e := fmt.Errorf("JSON array %q: expected length %d but got %d", path, wantLen, len(arr))
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepAssertJSONFieldContains(ctx *steps.ScenarioContext, args ...string) error {
	path := args[0]
	substr := args[1]

	obj, err := parseBody(ctx)
	if err != nil {
		return err
	}
	val, ok := jsonPath(obj, path)
	if !ok {
		e := fmt.Errorf("JSON field %q not found in response", path)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	got := fmt.Sprintf("%v", val)
	if !strings.Contains(got, substr) {
		e := fmt.Errorf("JSON field %q: %q does not contain %q", path, got, substr)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepAssertJSONFieldEqualsNumeric(ctx *steps.ScenarioContext, args ...string) error {
	path := args[0]
	wantStr := args[1]
	wantF, err := strconv.ParseFloat(wantStr, 64)
	if err != nil {
		return fmt.Errorf("invalid numeric value %q", wantStr)
	}

	obj, parseErr := parseBody(ctx)
	if parseErr != nil {
		return parseErr
	}
	val, ok := jsonPath(obj, path)
	if !ok {
		e := fmt.Errorf("JSON field %q not found in response", path)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}

	var gotF float64
	switch v := val.(type) {
	case float64:
		gotF = v
	case json.Number:
		gotF, err = v.Float64()
		if err != nil {
			return fmt.Errorf("JSON field %q is not numeric: %w", path, err)
		}
	default:
		e := fmt.Errorf("JSON field %q is not a number (got %T)", path, val)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}

	if gotF != wantF {
		e := fmt.Errorf("JSON field %q: expected %v but got %v", path, wantF, gotF)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepAssertJSONArrayNotEmpty(ctx *steps.ScenarioContext, args ...string) error {
	path := args[0]
	obj, err := parseBody(ctx)
	if err != nil {
		return err
	}
	val, ok := jsonPath(obj, path)
	if !ok {
		e := fmt.Errorf("JSON field %q not found in response", path)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	arr, ok := val.([]interface{})
	if !ok {
		e := fmt.Errorf("JSON field %q is not an array", path)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	if len(arr) == 0 {
		e := fmt.Errorf("JSON array %q is empty but should not be", path)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

func stepAssertJSONArrayEmpty(ctx *steps.ScenarioContext, args ...string) error {
	path := args[0]
	obj, err := parseBody(ctx)
	if err != nil {
		return err
	}
	val, ok := jsonPath(obj, path)
	if !ok {
		// Treat missing field as empty array — some APIs omit the key entirely.
		return nil
	}
	arr, ok := val.([]interface{})
	if !ok {
		e := fmt.Errorf("JSON field %q is not an array", path)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	if len(arr) != 0 {
		e := fmt.Errorf("JSON array %q should be empty but has %d element(s)", path, len(arr))
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(e)
			return nil
		}
		return e
	}
	return nil
}

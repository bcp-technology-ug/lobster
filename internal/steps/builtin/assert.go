package builtin

import (
	"encoding/json"
	"fmt"
	"regexp"
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
		// Regex match
		{
			`the response JSON field "([^"]+)" should match "([^"]+)"`,
			stepAssertJSONFieldMatches,
		},
		// Type assertions
		{
			`the response JSON field "([^"]+)" should be a number`,
			stepAssertJSONFieldIsNumber,
		},
		{
			`the response JSON field "([^"]+)" should be a string`,
			stepAssertJSONFieldIsString,
		},
		{
			`the response JSON field "([^"]+)" should be a boolean`,
			stepAssertJSONFieldIsBoolean,
		},
		{
			`the response JSON field "([^"]+)" should be null`,
			stepAssertJSONFieldIsNull,
		},
		// Numeric comparisons
		{
			`the response JSON field "([^"]+)" should be greater than (\d+(?:\.\d+)?)`,
			stepAssertJSONFieldGreaterThan,
		},
		{
			`the response JSON field "([^"]+)" should be less than (\d+(?:\.\d+)?)`,
			stepAssertJSONFieldLessThan,
		},
		{
			`the response JSON field "([^"]+)" should be between (\d+(?:\.\d+)?) and (\d+(?:\.\d+)?)`,
			stepAssertJSONFieldBetween,
		},
		// Array element search
		{
			`the response JSON array "([^"]+)" should contain an element where "([^"]+)" equals "([^"]+)"`,
			stepAssertJSONArrayHasElementWhere,
		},
		// DataTable bulk assertion
		{
			`the response JSON should include fields:`,
			stepAssertJSONIncludesFields,
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

// stepAssertJSONFieldMatches handles:
//
//	the response JSON field "PATH" should match "REGEX"
func stepAssertJSONFieldMatches(ctx *steps.ScenarioContext, args ...string) error {
	path, pattern := args[0], args[1]
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex %q: %w", pattern, err)
	}
	obj, parseErr := parseBody(ctx)
	if parseErr != nil {
		return parseErr
	}
	val, ok := jsonPath(obj, path)
	if !ok {
		e := fmt.Errorf("JSON field %q not found in response", path)
		return softOrHard(ctx, e)
	}
	got := fmt.Sprintf("%v", val)
	if !re.MatchString(got) {
		e := fmt.Errorf("JSON field %q (%q) does not match regex %q", path, got, pattern)
		return softOrHard(ctx, e)
	}
	return nil
}

// stepAssertJSONFieldIsNumber handles: the response JSON field "PATH" should be a number
func stepAssertJSONFieldIsNumber(ctx *steps.ScenarioContext, args ...string) error {
	return assertJSONFieldType(ctx, args[0], "number", func(v interface{}) bool {
		_, ok := v.(float64)
		return ok
	})
}

// stepAssertJSONFieldIsString handles: the response JSON field "PATH" should be a string
func stepAssertJSONFieldIsString(ctx *steps.ScenarioContext, args ...string) error {
	return assertJSONFieldType(ctx, args[0], "string", func(v interface{}) bool {
		_, ok := v.(string)
		return ok
	})
}

// stepAssertJSONFieldIsBoolean handles: the response JSON field "PATH" should be a boolean
func stepAssertJSONFieldIsBoolean(ctx *steps.ScenarioContext, args ...string) error {
	return assertJSONFieldType(ctx, args[0], "boolean", func(v interface{}) bool {
		_, ok := v.(bool)
		return ok
	})
}

// stepAssertJSONFieldIsNull handles: the response JSON field "PATH" should be null
func stepAssertJSONFieldIsNull(ctx *steps.ScenarioContext, args ...string) error {
	path := args[0]
	obj, err := parseBody(ctx)
	if err != nil {
		return err
	}
	val, ok := jsonPath(obj, path)
	if !ok {
		e := fmt.Errorf("JSON field %q not found in response", path)
		return softOrHard(ctx, e)
	}
	if val != nil {
		e := fmt.Errorf("JSON field %q is not null (got %T: %v)", path, val, val)
		return softOrHard(ctx, e)
	}
	return nil
}

func assertJSONFieldType(ctx *steps.ScenarioContext, path, typeName string, check func(interface{}) bool) error {
	obj, err := parseBody(ctx)
	if err != nil {
		return err
	}
	val, ok := jsonPath(obj, path)
	if !ok {
		e := fmt.Errorf("JSON field %q not found in response", path)
		return softOrHard(ctx, e)
	}
	if !check(val) {
		e := fmt.Errorf("JSON field %q is not a %s (got %T)", path, typeName, val)
		return softOrHard(ctx, e)
	}
	return nil
}

// stepAssertJSONFieldGreaterThan handles:
//
//	the response JSON field "PATH" should be greater than N
func stepAssertJSONFieldGreaterThan(ctx *steps.ScenarioContext, args ...string) error {
	path := args[0]
	threshold, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return fmt.Errorf("invalid threshold %q", args[1])
	}
	return assertNumericComparison(ctx, path, threshold, func(got, t float64) bool { return got > t },
		fmt.Sprintf("greater than %v", threshold))
}

// stepAssertJSONFieldLessThan handles:
//
//	the response JSON field "PATH" should be less than N
func stepAssertJSONFieldLessThan(ctx *steps.ScenarioContext, args ...string) error {
	path := args[0]
	threshold, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return fmt.Errorf("invalid threshold %q", args[1])
	}
	return assertNumericComparison(ctx, path, threshold, func(got, t float64) bool { return got < t },
		fmt.Sprintf("less than %v", threshold))
}

// stepAssertJSONFieldBetween handles:
//
//	the response JSON field "PATH" should be between N and M
func stepAssertJSONFieldBetween(ctx *steps.ScenarioContext, args ...string) error {
	path := args[0]
	lo, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return fmt.Errorf("invalid lower bound %q", args[1])
	}
	hi, err := strconv.ParseFloat(args[2], 64)
	if err != nil {
		return fmt.Errorf("invalid upper bound %q", args[2])
	}
	return assertNumericComparison(ctx, path, lo, func(got, _ float64) bool { return got >= lo && got <= hi },
		fmt.Sprintf("between %v and %v", lo, hi))
}

func assertNumericComparison(ctx *steps.ScenarioContext, path string, threshold float64, pass func(float64, float64) bool, desc string) error {
	obj, err := parseBody(ctx)
	if err != nil {
		return err
	}
	val, ok := jsonPath(obj, path)
	if !ok {
		e := fmt.Errorf("JSON field %q not found in response", path)
		return softOrHard(ctx, e)
	}
	gotF, numErr := toFloat64(val)
	if numErr != nil {
		e := fmt.Errorf("JSON field %q is not numeric: %w", path, numErr)
		return softOrHard(ctx, e)
	}
	if !pass(gotF, threshold) {
		e := fmt.Errorf("JSON field %q: expected value %v to be %s", path, gotF, desc)
		return softOrHard(ctx, e)
	}
	return nil
}

func toFloat64(v interface{}) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case json.Number:
		return n.Float64()
	default:
		return 0, fmt.Errorf("not a number (%T)", v)
	}
}

// stepAssertJSONArrayHasElementWhere handles:
//
//	the response JSON array "PATH" should contain an element where "FIELD" equals "VALUE"
func stepAssertJSONArrayHasElementWhere(ctx *steps.ScenarioContext, args ...string) error {
	arrayPath, field, wantVal := args[0], args[1], args[2]

	obj, err := parseBody(ctx)
	if err != nil {
		return err
	}
	raw, ok := jsonPath(obj, arrayPath)
	if !ok {
		e := fmt.Errorf("JSON field %q not found in response", arrayPath)
		return softOrHard(ctx, e)
	}
	arr, ok := raw.([]interface{})
	if !ok {
		e := fmt.Errorf("JSON field %q is not an array", arrayPath)
		return softOrHard(ctx, e)
	}
	for _, elem := range arr {
		if val, found := jsonPath(elem, field); found {
			if fmt.Sprintf("%v", val) == wantVal {
				return nil
			}
		}
	}
	e := fmt.Errorf("JSON array %q has no element where %q equals %q", arrayPath, field, wantVal)
	return softOrHard(ctx, e)
}

// stepAssertJSONIncludesFields handles a DataTable assertion:
//
//	the response JSON should include fields:
//	  | field   | value   |
//	  | status  | active  |
//	  | version | 2       |
func stepAssertJSONIncludesFields(ctx *steps.ScenarioContext, _ ...string) error {
	if ctx.CurrentStep == nil || ctx.CurrentStep.DataTable == nil {
		return fmt.Errorf("step requires a DataTable")
	}
	obj, err := parseBody(ctx)
	if err != nil {
		return err
	}

	var errs []error
	for _, row := range ctx.CurrentStep.DataTable.Rows {
		if len(row) < 2 {
			continue
		}
		field, want := strings.TrimSpace(row[0]), strings.TrimSpace(row[1])
		// Skip header row
		if field == "field" || field == "key" || field == "path" {
			continue
		}
		val, found := jsonPath(obj, field)
		if !found {
			errs = append(errs, fmt.Errorf("JSON field %q not found", field))
			continue
		}
		got := fmt.Sprintf("%v", val)
		if got != want {
			errs = append(errs, fmt.Errorf("JSON field %q: expected %q but got %q", field, want, got))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	combined := fmt.Errorf("JSON field assertions failed:\n  %s", strings.Join(msgs, "\n  "))
	return softOrHard(ctx, combined)
}

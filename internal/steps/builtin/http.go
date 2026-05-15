package builtin

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bcp-technology-ug/lobster/internal/steps"
)

func registerHTTPSteps(r *steps.Registry) error {
	defs := []struct {
		pattern string
		handler steps.StepHandler
	}{
		// HTTP request steps
		{
			`I send a (GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS) request to "([^"]+)"`,
			stepSendRequest,
		},
		{
			`I send a (GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS) request to "([^"]+)" with body:`,
			stepSendRequestWithDocStringBody,
		},
		{
			`I send a (GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS) request to "([^"]+)" with JSON body "([^"]+)"`,
			stepSendRequestWithInlineBody,
		},
		{
			`I set the request header "([^"]+)" to "([^"]+)"`,
			stepSetRequestHeader,
		},
		{
			`I set the base URL to "([^"]+)"`,
			stepSetBaseURL,
		},
		// Response assertion steps
		{
			`the response status should be (\d+)`,
			stepAssertStatusCode,
		},
		{
			`the response status should not be (\d+)`,
			stepAssertStatusCodeNot,
		},
		{
			`the response body should contain "([^"]+)"`,
			stepAssertBodyContains,
		},
		{
			`the response body should not contain "([^"]+)"`,
			stepAssertBodyNotContains,
		},
		{
			`the response header "([^"]+)" should equal "([^"]+)"`,
			stepAssertResponseHeader,
		},
		{
			`the response body should be valid JSON`,
			stepAssertBodyIsJSON,
		},
		{
			`I store the response body in variable "([^"]+)"`,
			stepStoreBodyInVar,
		},
		// Auth helpers
		{
			`I set the bearer token "([^"]+)"`,
			stepSetBearerToken,
		},
		{
			`I set the basic auth username "([^"]+)" and password "([^"]+)"`,
			stepSetBasicAuth,
		},
		// Form data
		{
			`I send a (GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS) request to "([^"]+)" with form data:`,
			stepSendRequestWithFormData,
		},
		// Regex body assertions
		{
			`the response body should match "([^"]+)"`,
			stepAssertBodyMatches,
		},
		{
			`the response body should not match "([^"]+)"`,
			stepAssertBodyNotMatches,
		},
		// Regex header assertion
		{
			`the response header "([^"]+)" should match "([^"]+)"`,
			stepAssertResponseHeaderMatches,
		},
		// Response time assertion
		{
			`the response time should be less than (\d+)ms`,
			stepAssertResponseTimeLessThan,
		},
		// Redirect control
		{
			`I follow redirects`,
			stepFollowRedirects,
		},
		{
			`I do not follow redirects`,
			stepDoNotFollowRedirects,
		},
	}
	for _, d := range defs {
		if err := r.Register(d.pattern, d.handler, srcHTTP); err != nil {
			return err
		}
	}
	return nil
}

// stepSendRequest handles: I send a METHOD request to "PATH"
func stepSendRequest(ctx *steps.ScenarioContext, args ...string) error {
	method := strings.ToUpper(args[0])
	path := args[1]
	return doRequest(ctx, method, path, nil, "")
}

// stepSendRequestWithDocStringBody handles the DocString body variant.
func stepSendRequestWithDocStringBody(ctx *steps.ScenarioContext, args ...string) error {
	method := strings.ToUpper(args[0])
	path := args[1]
	var body []byte
	var contentType string
	if ctx.CurrentStep != nil && ctx.CurrentStep.DocString != nil {
		body = []byte(ctx.CurrentStep.DocString.Content)
		contentType = ctx.CurrentStep.DocString.MediaType
	}
	if contentType == "" {
		contentType = "application/json"
	}
	return doRequest(ctx, method, path, body, contentType)
}

// stepSendRequestWithInlineBody handles: I send a METHOD request to "PATH" with JSON body "BODY"
func stepSendRequestWithInlineBody(ctx *steps.ScenarioContext, args ...string) error {
	method := strings.ToUpper(args[0])
	path := args[1]
	body := []byte(args[2])
	return doRequest(ctx, method, path, body, "application/json")
}

// stepSetRequestHeader stores a header to be sent with the next request.
// It modifies DefaultHeaders so subsequent requests also include it.
func stepSetRequestHeader(ctx *steps.ScenarioContext, args ...string) error {
	key := args[0]
	value := args[1]
	if ctx.DefaultHeaders == nil {
		ctx.DefaultHeaders = make(map[string]string)
	}
	ctx.DefaultHeaders[key] = value
	return nil
}

// stepSetBaseURL overrides the HTTP base URL for the scenario.
func stepSetBaseURL(ctx *steps.ScenarioContext, args ...string) error {
	ctx.BaseURL = args[0]
	return nil
}

func stepAssertStatusCode(ctx *steps.ScenarioContext, args ...string) error {
	var want int
	if _, err := fmt.Sscanf(args[0], "%d", &want); err != nil {
		return fmt.Errorf("invalid status code %q", args[0])
	}
	if ctx.LastResponse == nil {
		return fmt.Errorf("no HTTP response received")
	}
	if ctx.LastResponse.StatusCode != want {
		err := fmt.Errorf("expected status %d but got %d", want, ctx.LastResponse.StatusCode)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(err)
			return nil
		}
		return err
	}
	return nil
}

func stepAssertStatusCodeNot(ctx *steps.ScenarioContext, args ...string) error {
	var notWant int
	if _, err := fmt.Sscanf(args[0], "%d", &notWant); err != nil {
		return fmt.Errorf("invalid status code %q", args[0])
	}
	if ctx.LastResponse == nil {
		return fmt.Errorf("no HTTP response received")
	}
	if ctx.LastResponse.StatusCode == notWant {
		err := fmt.Errorf("expected status not to be %d", notWant)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(err)
			return nil
		}
		return err
	}
	return nil
}

func stepAssertBodyContains(ctx *steps.ScenarioContext, args ...string) error {
	needle := args[0]
	if !strings.Contains(string(ctx.LastBody), needle) {
		err := fmt.Errorf("response body does not contain %q", needle)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(err)
			return nil
		}
		return err
	}
	return nil
}

func stepAssertBodyNotContains(ctx *steps.ScenarioContext, args ...string) error {
	needle := args[0]
	if strings.Contains(string(ctx.LastBody), needle) {
		err := fmt.Errorf("response body unexpectedly contains %q", needle)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(err)
			return nil
		}
		return err
	}
	return nil
}

func stepAssertResponseHeader(ctx *steps.ScenarioContext, args ...string) error {
	key := args[0]
	want := args[1]
	if ctx.LastResponse == nil {
		return fmt.Errorf("no HTTP response received")
	}
	got := ctx.LastResponse.Header.Get(key)
	if got != want {
		err := fmt.Errorf("response header %q: expected %q but got %q", key, want, got)
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(err)
			return nil
		}
		return err
	}
	return nil
}

func stepAssertBodyIsJSON(ctx *steps.ScenarioContext, _ ...string) error {
	if !json.Valid(ctx.LastBody) {
		err := fmt.Errorf("response body is not valid JSON")
		if ctx.SoftAssertMode {
			ctx.AddAssertionError(err)
			return nil
		}
		return err
	}
	return nil
}

func stepStoreBodyInVar(ctx *steps.ScenarioContext, args ...string) error {
	varName := args[0]
	ctx.Variables[varName] = string(ctx.LastBody)
	return nil
}

// stepSetBearerToken handles: I set the bearer token "TOKEN"
func stepSetBearerToken(ctx *steps.ScenarioContext, args ...string) error {
	if ctx.DefaultHeaders == nil {
		ctx.DefaultHeaders = make(map[string]string)
	}
	ctx.DefaultHeaders["Authorization"] = "Bearer " + args[0]
	return nil
}

// stepSetBasicAuth handles: I set the basic auth username "USER" and password "PASS"
func stepSetBasicAuth(ctx *steps.ScenarioContext, args ...string) error {
	if ctx.DefaultHeaders == nil {
		ctx.DefaultHeaders = make(map[string]string)
	}
	creds := base64.StdEncoding.EncodeToString([]byte(args[0] + ":" + args[1]))
	ctx.DefaultHeaders["Authorization"] = "Basic " + creds
	return nil
}

// stepSendRequestWithFormData handles:
//
//	I send a METHOD request to "PATH" with form data: (DataTable)
//
// The DataTable must be a two-column key/value table (header row optional).
func stepSendRequestWithFormData(ctx *steps.ScenarioContext, args ...string) error {
	method := strings.ToUpper(args[0])
	path := args[1]

	form := url.Values{}
	if ctx.CurrentStep != nil && ctx.CurrentStep.DataTable != nil {
		for _, row := range ctx.CurrentStep.DataTable.Rows {
			if len(row) < 2 {
				continue
			}
			// Skip header row if it looks like a header (key == "key", "field", "param", etc.)
			if row[0] == "key" || row[0] == "field" || row[0] == "param" || row[0] == "name" {
				continue
			}
			form.Set(row[0], row[1])
		}
	}
	return doRequest(ctx, method, path, []byte(form.Encode()), "application/x-www-form-urlencoded")
}

func stepAssertBodyMatches(ctx *steps.ScenarioContext, args ...string) error {
	re, err := regexp.Compile(args[0])
	if err != nil {
		return fmt.Errorf("invalid regex %q: %w", args[0], err)
	}
	if !re.Match(ctx.LastBody) {
		e := fmt.Errorf("response body does not match regex %q", args[0])
		return softOrHard(ctx, e)
	}
	return nil
}

func stepAssertBodyNotMatches(ctx *steps.ScenarioContext, args ...string) error {
	re, err := regexp.Compile(args[0])
	if err != nil {
		return fmt.Errorf("invalid regex %q: %w", args[0], err)
	}
	if re.Match(ctx.LastBody) {
		e := fmt.Errorf("response body unexpectedly matches regex %q", args[0])
		return softOrHard(ctx, e)
	}
	return nil
}

func stepAssertResponseHeaderMatches(ctx *steps.ScenarioContext, args ...string) error {
	key := args[0]
	pattern := args[1]
	if ctx.LastResponse == nil {
		return fmt.Errorf("no HTTP response received")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex %q: %w", pattern, err)
	}
	got := ctx.LastResponse.Header.Get(key)
	if !re.MatchString(got) {
		e := fmt.Errorf("response header %q (%q) does not match regex %q", key, got, pattern)
		return softOrHard(ctx, e)
	}
	return nil
}

// stepAssertResponseTimeLessThan handles: the response time should be less than Nms
//
// Requires that the last request recorded a response time in
// ctx.Variables["__last_response_time_ms"].
func stepAssertResponseTimeLessThan(ctx *steps.ScenarioContext, args ...string) error {
	limitMs, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid millisecond value %q", args[0])
	}
	raw := ctx.Variables[varLastResponseTimeMs]
	if raw == "" {
		return fmt.Errorf("no response time recorded; make an HTTP request first")
	}
	got, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fmt.Errorf("corrupt response time value %q", raw)
	}
	if got >= limitMs {
		e := fmt.Errorf("response time %dms is not less than %dms", got, limitMs)
		return softOrHard(ctx, e)
	}
	return nil
}

// stepFollowRedirects restores the default redirect-following behaviour.
func stepFollowRedirects(ctx *steps.ScenarioContext, _ ...string) error {
	if ctx.HTTPClient == nil {
		ctx.HTTPClient = http.DefaultClient
	}
	ctx.HTTPClient = &http.Client{
		Timeout:       ctx.HTTPClient.Timeout,
		Transport:     ctx.HTTPClient.Transport,
		CheckRedirect: nil, // default: follow up to 10 redirects
	}
	return nil
}

// stepDoNotFollowRedirects configures the HTTP client to return on any redirect.
func stepDoNotFollowRedirects(ctx *steps.ScenarioContext, _ ...string) error {
	base := ctx.HTTPClient
	if base == nil {
		base = http.DefaultClient
	}
	ctx.HTTPClient = &http.Client{
		Timeout:   base.Timeout,
		Transport: base.Transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return nil
}

// doRequest performs an HTTP request and stores the response on ctx.
func doRequest(ctx *steps.ScenarioContext, method, path string, body []byte, contentType string) error {
	reqURL := buildURL(ctx.BaseURL, path)
	client := ctx.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("build request %s %s: %w", method, reqURL, err)
	}
	for k, v := range ctx.DefaultHeaders {
		req.Header.Set(k, v)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return fmt.Errorf("send %s %s: %w", method, reqURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	ctx.LastRequest = req
	ctx.LastResponse = resp
	ctx.LastBody = respBody
	ctx.Variables[varLastResponseTimeMs] = strconv.FormatInt(elapsed.Milliseconds(), 10)
	return nil
}

// varLastResponseTimeMs is stored in Variables after each HTTP request so the
// response-time assertion step can read it.
const varLastResponseTimeMs = "__last_response_time_ms"

// buildURL joins a base URL and a path, avoiding double slashes.
func buildURL(base, path string) string {
	if base == "" {
		return path
	}
	base = strings.TrimRight(base, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

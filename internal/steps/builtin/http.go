package builtin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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

// doRequest performs an HTTP request and stores the response on ctx.
func doRequest(ctx *steps.ScenarioContext, method, path string, body []byte, contentType string) error {
	url := buildURL(ctx.BaseURL, path)
	client := ctx.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("build request %s %s: %w", method, url, err)
	}
	for k, v := range ctx.DefaultHeaders {
		req.Header.Set(k, v)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send %s %s: %w", method, url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	ctx.LastRequest = req
	ctx.LastResponse = resp
	ctx.LastBody = respBody
	return nil
}

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

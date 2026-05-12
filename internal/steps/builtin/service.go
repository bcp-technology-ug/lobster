package builtin

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bcp-technology/lobster/internal/steps"
)

func registerServiceSteps(r *steps.Registry) error {
	defs := []struct {
		pattern string
		handler steps.StepHandler
	}{
		{
			`the service "([^"]+)" is running`,
			stepServiceIsRunning,
		},
		{
			`the service "([^"]+)" is running at "([^"]+)"`,
			stepServiceIsRunningAt,
		},
		{
			`I wait up to (\d+)s? for the service "([^"]+)" to be running`,
			stepWaitForService,
		},
	}
	for _, d := range defs {
		if err := r.Register(d.pattern, d.handler, srcService); err != nil {
			return err
		}
	}
	return nil
}

// stepServiceIsRunning checks that the service named `name` responds on its
// registered URL. It uses the base URL from context with /<name>/health as the
// health endpoint path.
func stepServiceIsRunning(ctx *steps.ScenarioContext, args ...string) error {
	name := args[0]
	url := buildURL(ctx.BaseURL, "/"+name+"/health")
	return checkServiceURL(ctx, url)
}

// stepServiceIsRunningAt checks that the service at the explicit URL is healthy.
func stepServiceIsRunningAt(ctx *steps.ScenarioContext, args ...string) error {
	url := args[1]
	return checkServiceURL(ctx, url)
}

// stepWaitForService polls until the service responds or the timeout expires.
func stepWaitForService(ctx *steps.ScenarioContext, args ...string) error {
	var timeoutSecs int
	if _, err := fmt.Sscanf(args[0], "%d", &timeoutSecs); err != nil {
		return fmt.Errorf("invalid timeout %q", args[0])
	}
	name := args[1]
	url := buildURL(ctx.BaseURL, "/"+name+"/health")

	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)
	for {
		if err := checkServiceURL(ctx, url); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("service %q did not become available within %ds", name, timeoutSecs)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// checkServiceURL performs a GET to url and returns an error if the status is
// not 2xx.
func checkServiceURL(ctx *steps.ScenarioContext, url string) error {
	client := ctx.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Get(url) //nolint:noctx // service readiness probe
	if err != nil {
		return fmt.Errorf("service unreachable at %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("service at %s responded with status %d", url, resp.StatusCode)
	}
	return nil
}

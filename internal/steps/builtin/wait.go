package builtin

import (
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/bcp-technology-ug/lobster/internal/steps"
)

const srcWait = "builtin:wait"

func registerWaitSteps(r *steps.Registry) error {
	defs := []struct {
		pattern string
		handler steps.StepHandler
	}{
		{
			`I wait (\d+) seconds?`,
			stepWaitSeconds,
		},
		{
			`I wait (\d+) milliseconds?`,
			stepWaitMilliseconds,
		},
		{
			`I poll "([^"]+)" every (\d+)s? until the status is (\d+) for up to (\d+)s?`,
			stepPollUntilStatus,
		},
		{
			`I retry up to (\d+) times every (\d+)s? until the command "([^"]+)" exits 0`,
			stepRetryCommandUntilZero,
		},
	}
	for _, d := range defs {
		if err := r.Register(d.pattern, d.handler, srcWait); err != nil {
			return err
		}
	}
	return nil
}

// stepWaitSeconds handles: I wait N second(s)
func stepWaitSeconds(_ *steps.ScenarioContext, args ...string) error {
	n, err := strconv.Atoi(args[0])
	if err != nil || n < 0 {
		return fmt.Errorf("invalid duration %q", args[0])
	}
	time.Sleep(time.Duration(n) * time.Second)
	return nil
}

// stepWaitMilliseconds handles: I wait N millisecond(s)
func stepWaitMilliseconds(_ *steps.ScenarioContext, args ...string) error {
	n, err := strconv.Atoi(args[0])
	if err != nil || n < 0 {
		return fmt.Errorf("invalid duration %q", args[0])
	}
	time.Sleep(time.Duration(n) * time.Millisecond)
	return nil
}

// stepPollUntilStatus handles:
//
//	I poll "URL" every Ns until the status is CODE for up to Ms
func stepPollUntilStatus(ctx *steps.ScenarioContext, args ...string) error {
	url := args[0]
	intervalSecs, err := strconv.Atoi(args[1])
	if err != nil || intervalSecs <= 0 {
		return fmt.Errorf("invalid interval %q", args[1])
	}
	wantCode, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Errorf("invalid status code %q", args[2])
	}
	timeoutSecs, err := strconv.Atoi(args[3])
	if err != nil || timeoutSecs <= 0 {
		return fmt.Errorf("invalid timeout %q", args[3])
	}

	client := ctx.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	interval := time.Duration(intervalSecs) * time.Second
	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)

	for {
		resp, reqErr := client.Get(url) //nolint:noctx // polling probe
		if reqErr == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == wantCode {
				return nil
			}
		}
		if time.Now().After(deadline) {
			if reqErr != nil {
				return fmt.Errorf("URL %q did not return status %d within %ds: %w", url, wantCode, timeoutSecs, reqErr)
			}
			return fmt.Errorf("URL %q did not return status %d within %ds (last status: %d)",
				url, wantCode, timeoutSecs, resp.StatusCode)
		}
		time.Sleep(interval)
	}
}

// stepRetryCommandUntilZero handles:
//
//	I retry up to N times every Ds until the command "CMD" exits 0
//
// The last stdout/stderr are captured into the standard shell variables.
func stepRetryCommandUntilZero(ctx *steps.ScenarioContext, args ...string) error {
	maxRetries, err := strconv.Atoi(args[0])
	if err != nil || maxRetries <= 0 {
		return fmt.Errorf("invalid retry count %q", args[0])
	}
	intervalSecs, err := strconv.Atoi(args[1])
	if err != nil || intervalSecs <= 0 {
		return fmt.Errorf("invalid interval %q", args[1])
	}
	cmdStr := unescapeArg(args[2])
	interval := time.Duration(intervalSecs) * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		cmd := exec.Command("sh", "-c", cmdStr) //nolint:gosec,noctx // deliberate shell step
		if err := runAndCapture(ctx, cmd); err != nil {
			return err
		}
		exitCode, _ := strconv.Atoi(ctx.Variables[varShellExitCode])
		if exitCode == 0 {
			return nil
		}
		if attempt < maxRetries {
			time.Sleep(interval)
		}
	}

	return fmt.Errorf("command %q did not exit 0 after %d attempts (last exit code: %s, stderr: %s)",
		cmdStr, maxRetries,
		ctx.Variables[varShellExitCode],
		strings.TrimSpace(ctx.Variables[varShellStderr]),
	)
}

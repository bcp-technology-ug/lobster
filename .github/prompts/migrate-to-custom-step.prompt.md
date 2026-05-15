---
mode: agent
description: Identify when to extract a custom lobster step and scaffold the Go code
---

You are helping decide whether a set of repeated Gherkin steps should become a custom step, and if so, scaffolding the Go code.

## Step 1: Identify the repetition

Look for scenarios where 3 or more steps always appear together in the same order, e.g.:

```gherkin
# Repeated across many scenarios:
When I send a POST request to "/auth/login" with body:
  """json
  {"email": "${EMAIL}", "password": "${PASS}"}
  """
Then the response status should be 200
And I store JSON field "$.token" from the response in variable "TOKEN"
And I set the bearer token "${TOKEN}"
```

## Step 2: Decide: is a custom step worth it?

A custom step is worth extracting when:
- The sequence appears in 3+ scenarios across 2+ feature files
- The sequence has non-trivial logic (retries, data transformation, conditional branching)
- The domain concept has a clear, business-readable name (e.g. "I am logged in as a user with role `<role>`")

A custom step is NOT worth it when:
- The sequence only appears once or twice
- It can be handled with a `Background:` block (Background runs before every scenario in a feature)
- It's just convenience — use a `Background:` first

## Step 3: Design the step pattern

Good custom step patterns:
- Business-readable: `I am logged in as "${USER}" with role "${ROLE}"`
- Single responsibility: one clear action or assertion
- Consistent style with built-in steps (imperative, `Given/When/Then` compatible)
- Use `([^"]+)` for string captures, `(\d+)` for numbers

## Step 4: Scaffold the Go file

Create a new file, e.g. `internal/steps/custom/auth.go`:

```go
package custom

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strings"

    "github.com/bcp-technology-ug/lobster/internal/steps"
)

const srcCustomAuth = "custom:auth"

// Register wires custom auth steps into the registry.
func Register(r *steps.Registry) error {
    defs := []struct {
        pattern string
        handler steps.StepHandler
    }{
        {
            `I am logged in as "([^"]+)" with role "([^"]+)"`,
            stepLoginAs,
        },
    }
    for _, d := range defs {
        if err := r.Register(d.pattern, d.handler, srcCustomAuth); err != nil {
            return err
        }
    }
    return nil
}

func stepLoginAs(ctx *steps.ScenarioContext, args ...string) error {
    email, role := args[0], args[1]
    // Build login request body.
    body, _ := json.Marshal(map[string]string{
        "email": email,
        "password": ctx.Variables["DEFAULT_PASSWORD"],
        "role": role,
    })
    req, err := http.NewRequestWithContext(ctx.Context(), http.MethodPost,
        ctx.BaseURL+"/auth/login", strings.NewReader(string(body)))
    if err != nil {
        return fmt.Errorf("login: build request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")
    resp, err := ctx.HTTPClient.Do(req)
    if err != nil {
        return fmt.Errorf("login: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("login: expected 200, got %d", resp.StatusCode)
    }
    var result struct {
        Token string `json:"token"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return fmt.Errorf("login: decode response: %w", err)
    }
    ctx.DefaultHeaders["Authorization"] = "Bearer " + result.Token
    ctx.Variables["TOKEN"] = result.Token
    return nil
}
```

## Step 5: Wire the custom step into the runner

In `cmd/lobster/main.go` (or wherever `builtin.Register` is called), add:

```go
import "github.com/bcp-technology-ug/lobster/internal/steps/custom"

// After builtin.Register(reg):
if err := custom.Register(reg); err != nil {
    log.Fatal("custom steps:", err)
}
```

## Step 6: Verify

```sh
lobster steps --filter custom    # your new steps should appear
lobster validate --features 'features/**/*.feature'
```

## Step 7: Update the feature files

Replace the repeated step sequences with the new custom step:

```gherkin
Background:
  Given I am logged in as "${TEST_USER}" with role "admin"
```

Then validate again:
```sh
lobster validate --features 'features/**/*.feature'
lobster lint     --features 'features/**/*.feature'
```

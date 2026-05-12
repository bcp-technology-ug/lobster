# Step Definitions

Step definitions connect natural-language Gherkin steps to executable behavior.

## Built-in step library

Lobster ships a built-in step library covering HTTP, shell, filesystem, and service readiness
checks. All built-in steps are registered automatically; no extra configuration is required.

### HTTP steps (`builtin:http`)

| Gherkin pattern | Description |
|---|---|
| `I send a METHOD request to "PATH"` | Sends an HTTP request (no body). METHOD is one of GET POST PUT PATCH DELETE HEAD OPTIONS. |
| `I send a METHOD request to "PATH" with body:` | Sends a request using the step's DocString as the body. The DocString media type sets `Content-Type` (defaults to `application/json`). |
| `I send a METHOD request to "PATH" with JSON body "BODY"` | Sends a request with an inline JSON body. |
| `I set the request header "NAME" to "VALUE"` | Adds a default header applied to all subsequent requests in the scenario. |
| `I set the base URL to "URL"` | Overrides the base URL for subsequent requests in the scenario. |
| `the response status should be CODE` | Asserts the last response has an exact HTTP status code. |
| `the response status should not be CODE` | Asserts the last response does not have the given status code. |
| `the response body should contain "TEXT"` | Asserts the response body string contains TEXT. |
| `the response body should not contain "TEXT"` | Asserts the response body does not contain TEXT. |
| `the response header "NAME" should equal "VALUE"` | Asserts a response header has an exact value. |
| `the response body should be valid JSON` | Asserts the response body can be parsed as JSON. |
| `I store the response body in variable "NAME"` | Stores the full response body string into a scenario variable. |

Example:

```gherkin
Scenario: Health endpoint
  Given I set the base URL to "http://localhost:8080"
  When I send a GET request to "/health"
  Then the response status should be 200
  And the response body should contain "ok"
```

### Shell steps (`builtin:shell`)

Arguments support backslash escapes (`\"` for a literal quote, `\\` for a backslash).
After running a command the captured outputs are available as scenario variables
`__shell_stdout`, `__shell_stderr`, and `__shell_exit_code`.

| Gherkin pattern | Description |
|---|---|
| `I run the command "CMD"` | Executes CMD via `sh -c`. Scenario variables are injected into the subprocess environment. |
| `I run lobster "ARGS"` | Runs the `lobster` binary (or `$LOBSTER_BIN`) with shell-quoted ARGS. |
| `the exit code should be N` | Asserts the last command exited with code N. |
| `the exit code should not be N` | Asserts the last command did not exit with code N. |
| `the output should contain "TEXT"` | Asserts stdout contains TEXT. |
| `the output should not contain "TEXT"` | Asserts stdout does not contain TEXT. |
| `the stderr should contain "TEXT"` | Asserts stderr contains TEXT. |
| `the stderr should not contain "TEXT"` | Asserts stderr does not contain TEXT. |
| `the output should be valid JSON` | Asserts stdout is valid JSON. |
| `I store the output in variable "NAME"` | Stores stdout into a scenario variable. |

Example:

```gherkin
Scenario: CLI help flag
  When I run lobster "--help"
  Then the exit code should be 0
  And the output should contain "lobster"
```

### Filesystem steps (`builtin:fs`)

| Gherkin pattern | Description |
|---|---|
| `I am in a new temporary directory` | Creates a temp directory and changes into it. Cleaned up in AfterScenario. |
| `the file "PATH" should exist` | Asserts a file exists at PATH. |
| `the file "PATH" should not exist` | Asserts no file exists at PATH. |
| `the directory "PATH" should exist` | Asserts a directory exists at PATH. |
| `the directory "PATH" should not exist` | Asserts no directory exists at PATH. |
| `the file "PATH" should contain "TEXT"` | Asserts a file's contents contain TEXT. |
| `I create the file "PATH" with content:` | Creates a file at PATH with the step's DocString as content. |
| `I create the file "PATH" containing "CONTENT"` | Creates a file at PATH with inline CONTENT. |

### Service readiness steps (`builtin:service`)

| Gherkin pattern | Description |
|---|---|
| `the service "NAME" is running` | GET `{base_url}/{NAME}/health` and assert 2xx. |
| `the service "NAME" is running at "URL"` | GET the explicit URL and assert 2xx. |
| `I wait up to Ns for the service "NAME" to be running` | Polls `{base_url}/{NAME}/health` up to N seconds until 2xx. |

Example:

```gherkin
Scenario: Wait for API readiness
  Given I wait up to 30s for the service "api" to be running
  When I send a GET request to "/api/users"
  Then the response status should be 200
```

### Variable interpolation

Scenario variables can be referenced in step arguments using `${VAR_NAME}` syntax
(planned for v0.2). In v0.1, variables set by steps such as
`I store the output in variable "NAME"` or `I store the response body in variable "NAME"`
are available in subsequent step handlers via `ScenarioContext.Variables` and are
also injected as environment variables when running shell commands.

Gherkin coverage in v0.1:

- Full `Background` support (runs before each scenario in a feature)
- `Scenario Outline` with `Examples` expansion support

Hook support in v0.1:

- `BeforeSuite`
- `AfterSuite`
- `BeforeScenario`
- `AfterScenario`

Variable scope model in v0.1:

- Scenario-scoped variables
- Suite-scoped shared variables

## Matching behavior

Step matching should be:

- explicit and deterministic
- fail-fast on ambiguous matches
- clear when no match is found

Diagnostics should include scenario, step text, and candidate matches when useful.

Runtime undefined-step behavior:

- During execution, undefined steps are collected and summarized.
- The run fails at the end with a consolidated undefined-step report.

Assertion behavior:

- Default mode is hard-fail assertions.
- Optional soft-assert mode can collect multiple assertion failures before failing the scenario.

Retry policy in v0.1:

- No automatic retries are performed for failing scenarios or steps.
- Reliability patterns should be expressed explicitly via wait/retry step definitions.

HTTP defaults in v0.1:

- Configurable base URL
- Default headers
- Per-request override support in step arguments

## Extension model

Lobster supports custom step packages so teams can model domain-specific behavior without changing lobster core.

In v0.1, custom steps are added through static extension registration (compiled into the lobster binary or distribution build).

High-level extension workflow:

1. Implement the required step registrar contract
2. Register patterns and handlers
3. Wire registrar into the build's extension registry
4. Validate custom steps with lobster validate

Conceptual interface:

```go
type StepRegistrar interface {
    Register(registry Registry) error
}
```

Example extension config:

```yaml
steps:
  registries:
    - builtin-core
    - custom-payments
    - custom-identity
```

Future runtime plugins:

- Runtime-loaded plugins are planned for a future release through an RPC-based plugin system.
- Go `.so` plugin loading is intentionally avoided for portability and reproducibility reasons.
- Future plugins must advertise a compatible `steps.api_version` and be rejected when the version contract does not match.
- Static registration remains the v0.1 default and runtime plugin mismatches must fail closed.

Remote execution behavior:

- local mode executes step handlers in-process.
- daemon mode executes step handlers on the daemon host and streams progress to clients.
- sync and async run models share the same step semantics and diagnostics structure.

Contract direction:

- Step execution capabilities are defined in proto contracts and served over gRPC.
- HTTP access is provided through gRPC-Gateway over the same canonical contract.

## Authoring guidelines

- Keep step text intent-focused and readable
- Avoid embedding implementation details into step phrases
- Make handlers idempotent where possible
- Emit actionable error messages with context
- Keep shared state explicit and scoped

## Testing custom steps

Recommended strategy:

- unit test parsing/matching for step patterns
- integration test handlers against representative services
- add sample feature files that exercise extension behavior

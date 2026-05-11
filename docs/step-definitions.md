# Step Definitions

Step definitions connect natural-language Gherkin steps to executable behavior.

## Built-in step library

Lobster provides a built-in set of reusable steps for common E2E workflows.

Planned categories:

- Service state checks
- HTTP request and response assertions
- JSONPath and body assertions
- Retry and wait primitives for eventually consistent flows
- Basic auth/token flow helpers
- Data Table-driven steps for structured inputs

Built-in auth modes in v0.1:

- Bearer token
- Basic auth
- API key header
- mTLS client certificate authentication
- OAuth device/code flow helpers

Example built-in usage:

```gherkin
Scenario: Health endpoint
  Given the service "api" is running
  When I send a GET request to "/health"
  Then the response status should be 200
  And the response body should contain "ok"
```

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

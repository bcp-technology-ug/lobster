# Hooks

Hooks are callbacks that run at defined points in the test lifecycle. They are
registered programmatically and are the primary mechanism for custom setup,
teardown, and cross-cutting concerns such as authentication or observability.

## Lifecycle points

| Hook | When it runs | Called per |
|---|---|---|
| `BeforeSuite` | Before any scenario executes | Run (once) |
| `AfterSuite` | After all scenarios complete, even on failure | Run (once) |
| `BeforeScenario` | Before each individual scenario | Scenario |
| `AfterScenario` | After each individual scenario | Scenario |

## Error semantics

- **BeforeSuite** — if any hook returns an error the entire run fails immediately;
  remaining `BeforeSuite` hooks are not called.
- **AfterSuite** — all hooks are called regardless of earlier errors. The first
  error encountered is returned; subsequent errors are not surfaced.
- **BeforeScenario** — if a hook returns an error the scenario is marked failed
  and its steps are skipped; remaining `BeforeScenario` hooks are not called.
- **AfterScenario** — all hooks are called regardless of earlier errors. The
  first error encountered is returned as the scenario result.

## Registering hooks

Hooks are registered on a `*steps.HookRegistry`. Custom step packages receive
the registry during their `Register` call and attach hooks at that point.

```go
import "github.com/bcp-technology/lobster/internal/steps"

func Register(r *steps.Registry, h *steps.HookRegistry) error {
    h.BeforeSuite(func(ctx context.Context) error {
        // One-time setup: seed database, start auxiliary services, etc.
        return nil
    })

    h.AfterSuite(func(ctx context.Context) error {
        // One-time teardown: flush telemetry, close connections, etc.
        return nil
    })

    h.BeforeScenario(func(sc *steps.ScenarioContext) error {
        // Per-scenario setup: reset shared state, provision fixtures, etc.
        sc.Variables["__api_token"] = fetchToken()
        return nil
    })

    h.AfterScenario(func(sc *steps.ScenarioContext) error {
        // Per-scenario teardown: clean up created resources, reset DB, etc.
        return nil
    })

    return nil
}
```

## Accessing scenario state

`BeforeScenario` and `AfterScenario` receive a `*steps.ScenarioContext` which
carries scenario-scoped state:

| Field | Type | Description |
|---|---|---|
| `Variables` | `map[string]string` | Scenario-scoped key/value store, also injected as env vars in shell steps |
| `BaseURL` | `string` | HTTP base URL used by HTTP steps |
| `DefaultHeaders` | `map[string]string` | Headers sent with every HTTP request |
| `HTTPClient` | `*http.Client` | Override to use a custom transport (e.g. mTLS) |
| `SoftAssertMode` | `bool` | When true, assertion steps collect failures instead of returning immediately |

## Ordering

Hooks are executed in the order they were registered. When multiple step
packages are loaded, hooks from earlier packages run before those from later
packages within the same lifecycle point.

## Integration adapter hooks

The `integrations` package uses `BeforeSuite`/`AfterSuite` to start and stop
Docker Compose stacks and `BeforeScenario`/`AfterScenario` to apply per-scenario
environment resets. See [integrations.md](integrations.md) for details.

## Testing hooks

Recommended approach:

- Unit-test hook callbacks directly by constructing a `ScenarioContext` and
  calling the function.
- Add feature files under `tests/features/` that exercise the observable side
  effects of your hook (e.g. a file written in `BeforeScenario` that a
  filesystem step can assert on).
- Use `BeforeScenario`/`AfterScenario` to implement test isolation rather than
  sharing mutable global state between scenarios.

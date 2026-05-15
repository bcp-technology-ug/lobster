# CLAUDE.md — Lobster Project

Lobster is a CLI-first BDD end-to-end test runner. Read `AGENTS.md` first for the full guide.

## Allowed tools in this project

When working on lobster tests or the lobster codebase, you may:
- Run `lobster steps`, `lobster validate`, `lobster lint`, `lobster plan`, `lobster doctor`
- Run `lobster run` only when explicitly asked to execute tests
- Run `go build ./...` and `go test ./...` for the lobster source code
- Edit `.feature` files, `lobster.yaml`, and Go source files under `internal/`

## Critical rules

1. **Always run `lobster steps --format markdown` before writing feature files.** This is the only source of truth for available step patterns. Never guess.

2. **Validate before running.** The sequence is always: write → `lobster validate` → `lobster lint` → `lobster plan` → `lobster run`.

3. **Step pattern fidelity.** Step text must match a registered pattern character-for-character (lobster uses regexp matching). Mismatches produce `ErrUndefined` at runtime.

4. **Variable interpolation.** Use `${VAR_NAME}` syntax inside any quoted string argument. Variables are set with `I set variable "NAME" to "VALUE"` or captured from responses.

5. **DocString syntax.** Steps ending in `:` require a DocString body:
   ```gherkin
   When I send a POST request to "/api" with body:
     """json
     {"key": "value"}
     """
   ```

## Project structure

```
features/          ← .feature files live here
lobster.yaml       ← project config
docs/              ← reference documentation
internal/          ← Go source (CLI, runner, steps, parser)
```

## Memory guidance

When you encounter a new feature in lobster or discover how a specific step works, note it in your session memory. If a step pattern fails to match, check `lobster steps` output — the pattern may have changed.

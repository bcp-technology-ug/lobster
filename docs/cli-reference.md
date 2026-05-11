# CLI Reference

This document describes the planned lobster CLI commands, flags, and examples.

## Runtime model (v0.1)

- Scenario execution is serial for deterministic results.
- Infrastructure orchestration is SDK-driven and compose-compatible.
- Local mode executes fully in the CLI process.
- Daemon mode executes through a remote API endpoint.
- Output targets CLI and CI readability first.

Contract note:

- Even in local mode, the CLI should use the same generated proto request and response types as daemon mode.
- Local execution is an in-process implementation of the same service contract, not a separate ad hoc API.

## Command overview

- lobster init: scaffold a new lobster test project
- lobster plan: compute execution plan without running steps
- lobster validate: parse and validate feature files
- lobster lint: enforce style and quality checks for feature files
- lobster run: execute end-to-end test scenarios against the configured stack
- lobster config: inspect and validate active configuration

Note:

- Daemon lifecycle is handled by the separate `lobsterd` binary.

## Contract and generation workflow

Lobster follows a contract-first workflow aligned with company backend standards.

Recommended project-level commands:

- make generate: regenerate proto and sqlc artifacts
- make proto: regenerate proto-derived artifacts only
- make sqlc: regenerate sqlc repositories only
- make lint-proto: run buf lint checks
- make break-proto: run buf breaking checks against main

These commands are expected to keep generated artifacts and compatibility checks in sync with source contracts.

Why this matters:

- it keeps local mode and daemon mode aligned on the same shape for requests, responses, and validation
- it makes contract drift visible at generation time and in buf breaking checks, rather than only at runtime

If the repository does not yet expose these make targets, run equivalent direct commands in CI and local workflows (for example, buf generate, sqlc generate, buf lint, and buf breaking).

## lobster init

Scaffold project files.

```bash
lobster init [path]
```

Common flags:

- --template string: starter template name
- --force: overwrite existing scaffolding files

Example:

```bash
lobster init ./tests/e2e
```

## lobster validate

Validate Gherkin files and step resolution.

```bash
lobster validate [flags]
```

Common flags:

- --features string: feature file glob
- --strict: fail on warnings
- --format string: output format (tty|json)

Example:

```bash
lobster validate --features "features/**/*.feature" --strict
```

Contract validation note:

- `lobster validate` checks feature syntax, step resolution, and local execution-level validity.
- It does not replace proto linting and breaking checks.
- It does not execute buf/sqlc generation checks.
- Run project-level contract checks as part of the same validation workflow.

## lobster lint

Lint feature files for consistency and readability.

```bash
lobster lint [flags]
```

Common checks:

- Scenario naming conventions
- Duplicate or ambiguous steps
- Missing Given/When/Then structure
- Overly broad assertions

Strict mode behavior:

- `--strict` promotes selected warning-class lint rules to errors.
- The exact selected rule set is project-defined and configured by policy.

Example:

```bash
lobster lint --features "features/**/*.feature"
```

## lobster run

Run scenarios with full orchestration.

```bash
lobster run [flags]
```

Key flags:

- --features string: feature file glob
- --scenario-regex string: run only scenarios matching regex
- --changed-only: run scenarios mapped to changed files in git diff
- --workspace string: monorepo workspace selector (see docs/configuration.md workspace database isolation)
- --compose stringArray: one or more compose files
- --compose-profile stringArray: compose profiles to enable
- --executor-mode string: local|daemon
- --executor-addr string: daemon gRPC address (example: dns:///lobsterd:9443)
- --run-mode string: sync|async
- --env stringArray: KEY=VALUE overrides
- --matrix stringArray: run across multiple named config profiles
- --cache: enable reusable run cache
- --no-cache: bypass cache for deterministic cold execution
- --from-plan string: execute from a previously saved plan artifact
- -v: info-level runtime logs
- -vv: debug-level runtime logs
- -vvv: trace-level runtime logs
- --ci: non-interactive mode with stable log output
- --keep-stack: skip teardown on completion
- --report-json string: write JSON report
- --report-junit string: write JUnit XML report
- --report-verbose: include step-level details in report outputs
- --soft-assert: collect assertion failures and fail scenario at end
- --otel-endpoint string: OpenTelemetry collector endpoint
- --otel-service-name string: service name used for OTel traces
- --auth-token string: bearer token for daemon auth (prefer env in automation)
- --tls-ca-file string: trusted CA bundle for daemon connection
- --tls-cert-file string: client cert for mTLS
- --tls-key-file string: client key for mTLS
- --tags string: tag expression filter
- --fail-fast: stop after first failing scenario
- --timeout duration: global run timeout
- --step-timeout duration: default timeout for each step

Example:

```bash
lobster run \
  --compose docker-compose.yml \
  --compose docker-compose.test.yml \
  --tags "@smoke and not @wip" \
  --fail-fast \
  --ci \
  --report-junit reports/junit.xml \
  --report-json reports/results.json
```

Tag expression style:

- Uses Cucumber-style boolean expressions, for example: `@smoke and not @wip`.

Selective execution options:

- feature path/glob selection
- scenario name regex selection
- changed-files-based selection from git diff

Remote execution options:

- `--executor-mode local` keeps execution in-process (default).
- `--executor-mode daemon --executor-addr ...` routes execution to remote daemon.
- `--run-mode sync` streams progress and exits on run completion.
- `--run-mode async` returns a run identifier and exits.

Workspace behavior:

- v0.1 supports monorepo workspace discovery and workspace-scoped runs.

Execution output model (v0.1):

- Primary stream is hierarchical: Feature -> Scenario -> Step.
- Default local mode is structured CLI output.
- Optional TUI mode can be enabled explicitly.

Undefined step behavior:

- During `run`, undefined steps are collected across execution and reported together.
- Run exits non-zero after reporting all undefined steps.

Async run behavior:

- `run --run-mode async` prints `run_id` on successful submission.
- Use `lobster run watch --run-id <id>` to stream events.
- Use `lobster run status --run-id <id>` to inspect current state.
- Use `lobster run cancel --run-id <id>` to request cancellation.

### Async run management

When `--run-mode async` is used, run lifecycle is managed with run subcommands.

#### lobster run watch

Stream events for an existing async run.

```bash
lobster run watch --run-id <id>
```

Common flags:

- --run-id string: run identifier returned by async submission

#### lobster run status

Inspect current state for an existing async run.

```bash
lobster run status --run-id <id>
```

Common flags:

- --run-id string: run identifier returned by async submission

#### lobster run cancel

Request graceful cancellation for an existing async run.

```bash
lobster run cancel --run-id <id>
```

Common flags:

- --run-id string: run identifier returned by async submission

Step data support in v0.1:

- Gherkin Data Tables are supported as structured step input.

HTTP behavior defaults in v0.1:

- Base URL and default headers are configurable.
- Individual request steps can override defaults.

Failure diagnostics behavior:

- On failure, lobster captures relevant diagnostics when available:
- recent container logs for relevant services
- resolved effective configuration snapshot
- expanded scenario execution trace
- container/service state dump for failed services

Immediate failure output includes:

- failed step text with error reason
- relevant request/response excerpt
- linked service/container log snippet

Redaction behavior:

- Logs and reports redact sensitive values (tokens, passwords, secret headers) by default.
- An explicit allowlist-based override can be configured for non-sensitive fields.

## lobsterd

`lobsterd` is the dedicated long-running daemon binary for remote API and Wish hosting.

Typical usage:

```bash
lobsterd start \
  --listen :9443 \
  --http-listen :8080 \
  --db-path .lobster/lobster.db \
  --migrations-mode auto
```

Common flags:

- --listen string: daemon listen address
- --http-listen string: gRPC-Gateway listen address
- --db-path string: SQLite database path
- --migrations-mode string: auto|external|disabled
- --tls-cert-file string
- --tls-key-file string
- --tls-client-ca-file string
- --auth-mode string: token|none

## lobster config

Inspect resolved configuration.

```bash
lobster config [flags]
```

Common flags:

- --print: print effective config
- --validate: validate config schema
- --format string: tty|json

Example:

```bash
lobster config --print --format json
```

## Exit codes

Suggested run semantics:

- 0: all scenarios passed
- 1: one or more scenarios failed
- 2: validation or configuration error
- 3: infrastructure startup or orchestration error
- 4: internal runtime error

Daemon and remote execution semantics:

- API auth or TLS failure should return exit code `2` (config/runtime contract error).
- Remote orchestration failure should return exit code `3`.

Contract pipeline failure semantics:

- Proto lint or breaking-check failures should fail CI before run execution.
- Generation drift (source changed but generated artifacts stale) should fail CI validation.

Report detail levels:

- Default: scenario-level fields (name, status, duration, error summary)
- Verbose mode: scenario-level plus step-level execution details

Telemetry:

- v0.1 supports basic OpenTelemetry trace export when endpoint settings are provided.

## lobster plan

Generate an execution plan without running step handlers or mutating target systems.

```bash
lobster plan [flags]
```

Purpose:

- Resolve feature files, tag expressions, and scenario selection
- Expand Scenario Outline examples
- Resolve matrix/profile expansion
- Show orchestrator and integration actions that would be executed

Common flags:

- --features string
- --tags string
- --scenario-regex string
- --matrix stringArray
- --workspace string (see docs/configuration.md workspace database isolation)
- --compose-profile stringArray
- --cache
- --no-cache
- --out string (save plan artifact)
- --format string (tty|json)

Example:

```bash
lobster plan --workspace payments --features "features/**/*.feature" --tags "@smoke and not @quarantine" --matrix ci,ci-hardened --out plans/smoke.lobsterplan.json
```

Output path behavior:

- `--out` explicitly overrides the configured plan blob directory.
- Without `--out`, the default output location is derived from `persistence.plans.blob_dir`.

Apply workflow:

```bash
lobster run --from-plan plans/smoke.lobsterplan.json
```

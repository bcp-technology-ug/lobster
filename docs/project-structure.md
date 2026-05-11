# Project Structure

This document defines the recommended Go repository structure for lobster and the ownership boundaries of each package.

## Top-level layout

```text
lobster/
  cmd/
    lobster/
      main.go
    lobsterd/
      main.go
  proto/
    lobster/
      v1/
        common.proto
        run.proto
        plan.proto
        stack.proto
        admin.proto
  gen/
    go/
      lobster/v1/
    openapi/
      lobster/v1/
    ts/
      lobster/v1/
    sqlc/
      run/
      plan/
      stack/
      integrations/
  sql/
    run/
    plan/
    stack/
    integrations/
  migrations/
  internal/
    cli/
    config/
    parser/
    runner/
    orchestration/
    api/
    daemon/
    store/
    steps/
    integrations/
    reports/
    tui/
  pkg/
    plugin-sdk/
  buf.yaml
  buf.gen.yaml
  sqlc.yaml
  tests/
    unit/
    integration/
    e2e/
    fixtures/
```

## Package responsibilities

### cmd/lobster

- Process entrypoint
- Dependency wiring and startup
- Root command execution for local/client workflows

### cmd/lobsterd

- Long-running daemon entrypoint
- Remote API and Wish host process startup
- Daemon runtime lifecycle and shutdown orchestration

### internal/cli

- Cobra command tree and flags
- User input validation at CLI boundary
- Command to service-layer dispatch

### internal/config

- Viper loading and config merging
- Schema binding and validation
- Effective configuration rendering (`config --print`)

### internal/parser

- Gherkin file parsing to internal models
- Validation and linting rules
- Parser diagnostics

### internal/runner

- Coordinates full test execution flow
- Calls orchestrator, step registry, adapters, and reporters
- Produces run-level summary and status
- Enforces data lifecycle policy (seed/reset) and quarantine routing behavior

### internal/orchestration

- Docker SDK lifecycle management
- Service startup, health wait, teardown
- Runtime stack diagnostics and container logs
- Profile-driven migration execution modes (`auto`, `external`, `disabled`)

### internal/api

- Server-side transport wiring for gRPC and gRPC-Gateway
- Request/response translation between transport and runner services
- Authentication and capability handlers

### internal/daemon

- Daemon process lifecycle and startup
- Run service orchestration for sync and async execution paths
- Single-writer persistence coordination in daemon mode

### internal/store

- Repository implementations backed by SQLite
- sqlc-generated query integration
- Migration and schema-version checks

### internal/steps

- Step matcher and registry
- Built-in step implementations
- v0.1 static step-extension registration

### internal/integrations

- External system adapters
- Keycloak adapter implementation
- Adapter lifecycle hooks (setup/reset/teardown)

### internal/reports

- Console summary output
- JUnit XML generation
- JSON report generation

### internal/tui

- Reserved presentation layer for Bubbletea/Lipgloss/Bubbles
- Must remain independent from core execution ownership

### pkg/plugin-sdk

- Public extension contracts for future external authors
- Stable interface surface kept small and versioned
- Explicit API version compatibility checks against configured `steps.api_version`
- Any future runtime plugin system must fail closed on version mismatches

### proto/lobster/v1

- Canonical proto contract source files for Lobster API services
- Versioned package namespaces (for example, lobster.v1.run)
- Canonical proto source files used by buf generation
- HTTP mapping and validation annotations are authored here

### gen

- All generated artifacts (Go transport, gRPC and gRPC-Gateway stubs, OpenAPI, TypeScript, sqlc)
- Never hand-edit generated files; regenerate from source contracts

Scope note:

- OpenAPI generation is required in CI.
- TypeScript client generation is deferred until backend API stabilization.

### sql

- Hand-authored SQL query files used by sqlc
- Organized by capability surface (run, plan, stack)

### migrations

- Versioned schema migration files with up/down pairs
- Source-of-truth schema evolution history

### tests

- unit: pure Go package tests (`go test`)
- integration: Lobster-driven subsystem suites against controlled dependencies
- e2e: Lobster-driven full-workflow suites against composed fixture stacks
- fixtures: feature files, compose files, test data, golden reports

## Dependency direction rules

- cli depends on config and runner; never the reverse.
- runner depends on interfaces and result contracts, not concrete infrastructure types.
- parser has no dependency on orchestration or reporting.
- reports consumes run results only.
- integrations and orchestration are infrastructure adapters behind interfaces.

## v0.1 and v0.2 extension strategy

- v0.1: static extension registries keep behavior deterministic and portable.
- v0.2 target: RPC runtime plugin loading for out-of-process extensions.
- v0.2 plugin loading must preserve explicit version negotiation and fail closed when the version contract does not match.

## API and persistence strategy

- Proto-first contracts define remote execution capabilities.
- SQLite + sqlc provide structured persistence for run and plan state.
- In daemon mode, API services own persistence writes.

## Generation and ownership rules

- Developers author proto files, SQL queries, and business logic.
- Transport and repository implementations are generated.
- Manual edits to generated files are prohibited.

## Why this structure

This structure keeps core domain logic easy to test, prevents tight coupling between CLI and execution internals, and allows incremental growth without large refactors.

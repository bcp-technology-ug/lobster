# Testing Strategy

This document defines how Lobster is tested and what each test layer is responsible for.

## Principles

- keep fast feedback for core logic with Go unit tests
- validate real behavior with Lobster-driven integration and end-to-end suites
- dogfood continuously: the newest Lobster binaries must test Lobster itself

## Test pyramid

## Unit tests (Go)

Use Go unit tests for package-level behavior:

- parser and matcher logic
- config resolution and validation helpers
- runner decision logic and error mapping
- repository adapters and utility functions

Command:

```bash
go test ./...
```

## Integration tests (Lobster-driven)

Integration suites are authored as Lobster feature files and executed with Lobster.

Scope:

- interactions across multiple internal components
- compose lifecycle and readiness handling
- API contract behavior at service boundaries

Typical location:

- `tests/integration/`

Execution:

```bash
lobster run --features "tests/integration/**/*.feature"
```

## End-to-end tests (Lobster-driven)

E2E suites represent user-visible and operator-visible full workflows.

Scope:

- full stack orchestration
- realistic failure diagnostics paths
- report generation and CI artifact behavior

Typical location:

- `tests/e2e/`

Execution:

```bash
lobster run --features "tests/e2e/**/*.feature" --ci
```

## Dogfooding policy

Lobster uses self-hosted testing for integration and E2E quality gates.

Required policy:

- integration and E2E suites are executed by Lobster itself
- CI must run suites with binaries built from the current commit
- pull requests fail if the candidate binaries cannot pass Lobster-driven suites

Recommended CI flow:

1. build candidate binaries (`lobster`, `lobsterd`) from current commit
2. run Go unit tests
3. run Lobster-driven integration suites with candidate `lobster`
4. run Lobster-driven E2E suites with candidate `lobster`

Example candidate-binary flow:

```bash
go build -o bin/lobster ./cmd/lobster
go build -o bin/lobsterd ./cmd/lobsterd

./bin/lobster validate --features "tests/**/*.feature"
./bin/lobster run --ci --features "tests/integration/**/*.feature"
./bin/lobster run --ci --features "tests/e2e/**/*.feature"
```

## Tags and suite routing

Recommended tag policy:

- `@integration` for integration-level scenarios
- `@e2e` for full end-to-end scenarios
- `@quarantine` for non-blocking flaky scenarios

Routing examples:

```bash
./bin/lobster run --ci --tags "@integration and not @quarantine"
./bin/lobster run --ci --tags "@e2e and not @quarantine"
```

## Ownership and review

- behavior changes require updates to relevant feature suites
- new capabilities should include at least one Lobster-driven integration or E2E scenario
- generated reports (JSON/JUnit) remain standard CI artifacts for test visibility

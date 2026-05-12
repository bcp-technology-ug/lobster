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

## Integration and end-to-end tests (Lobster-driven)

Integration and E2E suites are authored as Lobster feature files under `tests/features/`
and executed with Lobster itself.

Integration scope:

- interactions across multiple internal components
- compose lifecycle and readiness handling
- API contract behavior at service boundaries
- parity between local execution and daemon-backed execution for the same request/response shapes

E2E scope:

- full stack orchestration
- realistic failure diagnostics paths
- report generation and CI artifact behavior

Location:

- `tests/features/`

Suites are tagged to distinguish scope: `@integration` for cross-component scenarios,
`@docker` and `@daemon` for daemon and compose-stack scenarios.

Execution:

```bash
# All suites
lobster run --features "tests/features/**/*.feature"

# Integration only
lobster run --features "tests/features/**/*.feature" --tags @integration

# Daemon/docker only
lobster run --features "tests/features/**/*.feature" --tags "@docker or @daemon" --ci
```

## Dogfooding policy

Lobster uses self-hosted testing for integration and E2E quality gates.

Required policy:

- suites are executed by Lobster itself
- CI must run suites with binaries built from the current commit
- pull requests fail if the candidate binaries cannot pass Lobster-driven suites

Recommended CI flow:

1. build candidate binaries (`lobster`, `lobsterd`) from current commit
2. run Go unit tests
3. run Lobster-driven feature suites with candidate `lobster`

Contract coverage note:

- add at least one integration scenario that exercises the same logical operation through local mode and daemon mode when both are available
- prefer tests that construct generated proto request types so contract drift is caught by compile errors and behavior checks

Example candidate-binary flow:

```bash
go build -o bin/lobster ./cmd/lobster
go build -o bin/lobsterd ./cmd/lobsterd

./bin/lobster validate --features "tests/features/**/*.feature"
./bin/lobster run --ci --features "tests/features/**/*.feature"
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

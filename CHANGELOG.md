# Changelog

All notable changes to lobster are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Lobster uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

---

## [0.1.0] — 2026-05-13

Initial public release.

### Added

- `lobster` CLI with `init`, `validate`, `lint`, `plan`, `run`, `config`, `runs`, `plans`, `stack`, `integrations`, and `admin` commands
- `lobsterd` daemon with gRPC and gRPC-Gateway HTTP/JSON API
- Local in-process execution mode and remote daemon execution mode (`--executor-mode daemon`)
- Proto-first API contracts; full gRPC + HTTP/JSON gateway
- Docker Compose lifecycle management via Docker SDK — health-aware startup, automatic teardown
- `Background`, `Scenario Outline`, and Data Table support in Gherkin feature execution
- Built-in HTTP step definitions: request dispatch, response assertions, header management, JSON body handling
- Built-in service readiness steps with configurable polling and timeouts
- `lobster plan` — Terraform-style deterministic execution plan before running
- `lobster run --from-plan` — apply a saved plan artifact
- SQLite persistence for run history, scenario results, and step-level detail via sqlc-generated queries
- `--run-mode async` for fire-and-forget run submission; `run watch`, `run status`, `run cancel` sub-commands
- Console, JUnit XML, and JSON report output
- Hierarchical output verbosity (`-v`, `-vv`, `-vvv`) and `--ci` mode
- Soft-assert mode (`--soft-assert`) — collect all assertion failures instead of stopping on the first
- Quarantine-tag workflow (`@quarantine`) for isolating flaky tests in CI
- Monorepo workspace discovery and workspace-scoped execution
- Configurable SQLite migration modes: `auto`, `external`, `disabled`
- Optional OpenTelemetry trace export via OTLP HTTP
- Basic Keycloak integration adapter for realm setup, user provisioning, and token acquisition
- Charm Wish SSH surface for remote `lobsterd` interaction
- Dogfooding: lobster's own integration and CLI test suite is executed by lobster itself
- MIT licence; no telemetry, no accounts, no limits

[Unreleased]: https://github.com/bcp-technology-ug/lobster/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/bcp-technology-ug/lobster/releases/tag/v0.1.0

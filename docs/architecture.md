# Lobster Architecture

This document describes the high-level architecture of lobster and how core components interact.

## v0.1 architecture stance

Lobster v0.1 optimizes for deterministic behavior, clear boundaries, and testability.

- Serial scenario execution (parallelism deferred)
- Docker SDK-backed orchestration from day one
- Built-in static step and integration extension registries (runtime plugins deferred)
- CI-first outputs: console summary, JUnit XML, JSON
- Hook support in v0.1: `BeforeSuite`, `AfterSuite`, `BeforeScenario`, `AfterScenario`
- Optional soft-assert execution mode in v0.1
- Basic OpenTelemetry trace export in v0.1
- Default test-data policy: per-scenario reset with idempotent seed behavior
- Migration handling is profile-configurable (`auto`, `external`, `disabled`)
- Flaky policy uses explicit quarantine tags and separate CI routing (no auto-quarantine)
- Structured CLI-first UX with hierarchical progress stream and explicit verbosity levels
- Terraform-style execution planning via `lobster plan`
- Monorepo workspace discovery support in v0.1
- Configurable execution cache with deterministic bypass controls
- Plan artifact workflow (`plan` output consumed by `run --from-plan`)
- Optional daemon execution mode for remote hosts while preserving CLI-first behavior
- Proto-first service contracts for gRPC and gRPC-Gateway surfaces
- SQLite + sqlc persistence for run history and structured execution state
- Codegen-first engineering standard: generated transport and repository layers

## Goals

- CLI-first workflow for local and CI usage
- Optional remote execution on stronger infrastructure through a daemon API
- Infrastructure orchestration through Docker Compose
- Clear separation between parsing, orchestration, execution, and reporting
- Extensible step and integration model

## Engineering standards alignment

Lobster follows shared BCP backend standards:

- Generate everything that can be generated (transport, gateway, repository code).
- Keep business logic in service and runner layers, not in generated code.
- Treat proto and SQL as source-of-truth contracts.
- Enforce proto lint and breaking-change checks in CI.
- Never hand-edit files under generated output directories.

## Execution modes

Lobster supports two runtime modes with one behavior model:

1. local mode: CLI process executes orchestration and run flow in-process.
2. daemon mode: CLI/Wish/HTTP clients call a remote daemon that owns orchestration and execution.

Design rules:

- CLI remains the primary workflow entrypoint in both modes.
- Wish is an additional client surface, not a separate backend.
- Sync streaming and async job runs are both first-class server capabilities.
- Local mode still uses the same proto-generated request and response types as daemon mode.
- Local execution must go through the same service interface boundary as remote execution, with a local in-process implementation behind that interface.
- Validation rules live in proto and are enforced before business logic in both modes.

## Primary components

- Command layer: Cobra command tree (`init`, `validate`, `lint`, `run`, `config`)
- Configuration layer: Viper-backed configuration resolution
- Workspace resolver: monorepo discovery and workspace-scoped targeting
- Parser and linter: Gherkin parsing, syntax validation, and quality checks
- Orchestration engine: Docker SDK lifecycle management with Compose-style behavior and health waiting
- Step engine: step matching and handler execution
- Integration adapters: service-specific setup helpers (for example, Keycloak)
- Presentation layer: structured CLI output in v0.1; richer Bubbletea/Lipgloss/Bubbles TUI planned for future versions
- Reporters: console, JSON, and JUnit XML outputs
- Planner: deterministic plan generation and artifact serialization
- Service API: proto-first contract boundary for Run/Plan/Stack/Admin services
- Persistence: SQLite repositories via sqlc-generated query layer

## Package layout (planned)

```text
cmd/
   lobster/
      main.go
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
tests/
   unit/
   integration/
   e2e/
   fixtures/
```

See docs/project-structure.md for package ownership and dependency direction.

## Component flow

```text
                +--------------------------+
                |      Cobra Commands      |
                | init validate lint run   |
                +------------+-------------+
                             |
                             v
                +--------------------------+
                |   Config Resolver        |
                |   (Viper + env + flags)  |
                +------------+-------------+
                             |
                             v
   +-------------------------+-------------------------+
   |                                                   |
   v                                                   v
+----------------------+                    +----------------------+
| Gherkin Parser/Lint  |                    | Compose Orchestrator |
| features + quality   |                    | up/wait/down/logs    |
+----------+-----------+                    +----------+-----------+
           |                                           |
           +-------------------+-----------------------+
                               v
                    +-----------------------+
                    |    Step Engine        |
                    | match + execute       |
                    +-----------+-----------+
                                |
                +---------------+----------------+
                |                                |
                v                                v
     +----------------------+         +----------------------+
     | Integration Adapters |         | Reporters            |
     | keycloak, custom...  |         | tty, json, junitxml  |
     +----------------------+         +----------------------+

Daemon-mode boundary:

- Client surfaces (CLI/Wish/HTTP) call the same server capabilities.
- Server owns orchestration, execution, and persistence writes.
```

## Command layer

Cobra defines the command tree and command-level flags. Commands are designed to be scriptable and deterministic for CI.

Expected v0.1 user-facing commands:

- `lobster plan`
- `lobster validate`
- `lobster lint`
- `lobster config`
- `lobster run`

Daemon binary:

- `lobsterd` (long-running remote daemon and Wish host)

## Configuration layer

Viper resolves configuration from multiple sources with predictable precedence:

1. Command-line flags
2. Environment variables
3. Config file (`lobster.yaml`)
4. Defaults

## Orchestration engine

The orchestration engine is responsible for:

- Reading one or more Docker Compose files
- Starting required services using Docker APIs
- Waiting for readiness and health checks
- Providing cleanup hooks after run completion

Design note:

- v0.1 uses SDK-backed orchestration for stronger programmatic control and diagnostics.

## Step engine

The step engine:

- Parses scenario steps
- Matches steps to registered handlers
- Executes handlers with context
- Captures assertions and diagnostics

Execution controls in v0.1:

- `--fail-fast` stops execution after the first failed scenario.
- No automatic scenario or step retries; explicit wait/retry steps are the primary mechanism.
- Timeout model includes both global run timeout and per-step timeout.
- Supports matrix-style multi-environment runs from one invocation.

Handlers can come from built-in libraries or statically registered extensions.

Extension strategy:

- v0.1: static extension registries (compiled-in extensions)
- v0.2 target: runtime RPC plugin loading

Service API strategy:

- Canonical contracts are defined in proto files.
- gRPC and HTTP gateway surfaces are generated from the same contract.
- Capability negotiation allows clients to adapt to optional server features.
- Proto contracts include explicit HTTP annotations for gateway generation.
- Validation rules are defined in proto and enforced by shared validation middleware.
- CLI code should build requests from generated proto types only; it should not define separate local-only DTOs.
- Local mode should call the same service contracts as daemon mode, which keeps API drift visible at compile time and in contract tests.

See docs/api-reference.md for transport contract details.

## Integration adapters

Adapters provide setup and interaction helpers for external systems.

Key adapter concepts:

- setup: establish required entities before tests
- reset: clear state between scenarios or suites
- teardown: remove transient resources after run

Data lifecycle defaults:

- v0.1 default is reset-per-scenario plus idempotent seeding.
- Lighter reset patterns can be opted in explicitly by tag/policy.

## Persistence architecture

Persistence baseline:

- SQLite is the primary embedded datastore.
- sqlc-generated queries provide typed data access.
- File reports (JSON/JUnit) remain supported for CI compatibility.

Access policy:

- local mode: CLI may read/write SQLite directly.
- daemon mode: daemon is the single writer; clients access state only through API.

Contract policy:

- local execution is still bound to the same service contract as daemon execution
- local implementations may write directly only after request validation passes through the shared proto contract

See docs/persistence.md for schema ownership, migration policy, retention, and compatibility rules.

## Output and reporting

Lobster supports two runtime experiences over the project lifecycle:

- Standard CLI output for scripts and logs (v0.1)
- Interactive TUI for local development insights (planned)

CLI UX defaults in v0.1:

- Feature -> Scenario -> Step hierarchical progress output
- Verbosity controls: `-v` (info), `-vv` (debug), `-vvv` (trace)
- Inline failure details include step error, request/response excerpt, and related service log snippet

Report outputs target automation and observability:

- Exit codes for pipeline gating
- JSON reports for machine processing
- JUnit XML for CI test dashboards

## Testing approach

Lobster uses a layered testing model:

- Go unit tests for package-level correctness (`go test`)
- Lobster-driven integration suites for subsystem behavior
- Lobster-driven E2E suites for full workflow verification

Dogfooding policy:

- The newest candidate Lobster binaries must run Lobster integration and E2E suites in CI.
- This ensures Lobster continuously validates its own runtime behavior and contract compatibility.

See docs/testing.md for detailed test-layer ownership and CI execution flow.

## Dependency boundaries

- `internal/runner` depends on interfaces, not concrete adapter implementations.
- `internal/parser` is side-effect free and independent of orchestration/reporting.
- `internal/cli` wires dependencies and does not implement core domain logic.
- `internal/reports` consumes execution results only.
- `internal/tui` is a presentation concern and must not own execution flow.

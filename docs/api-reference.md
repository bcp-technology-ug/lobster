# API Reference

This document defines the canonical Lobster API contract.

Source of truth policy:

- Protocol Buffers are the canonical contract.
- gRPC service interfaces are generated from proto definitions.
- HTTP endpoints are generated via gRPC-Gateway from the same proto definitions.
- CLI, Wish, and HTTP clients all target the same server capabilities.

## Goals

- Preserve Lobster as CLI-first while enabling remote execution on a stronger host.
- Keep one behavior model across local and remote execution modes.
- Support both synchronous streaming runs and asynchronous job workflows.
- Keep transport contracts versioned, with stricter backward-compatibility expectations after v1.0.

## Company standard alignment

Lobster follows the same contract-first standards used in other BCP backends:

- Developers author proto contracts and business logic; transport handlers are generated.
- gRPC and HTTP/JSON gateway behavior comes from proto annotations.
- Request validation rules are declared in proto and enforced uniformly at runtime.
- Breaking API changes are controlled by buf breaking checks in CI.

## Proto package and file conventions

Canonical layout:

- `proto/lobster/v1/common.proto` for shared messages
- `proto/lobster/v1/run.proto` for run lifecycle APIs
- `proto/lobster/v1/plan.proto` for planning APIs
- `proto/lobster/v1/stack.proto` for stack lifecycle APIs
- `proto/lobster/v1/admin.proto` for health/capabilities APIs

Naming conventions:

- package names follow `lobster.v1.<surface>`
- one service per proto surface file (for example, `RunService`)
- request and response types follow `{Action}{Entity}Request` and `{Action}{Entity}Response`
- all fields use snake_case in proto definitions

## HTTP mapping standards

RPC methods should use explicit `google.api.http` annotations so gateway routes are generated with no hand-written controllers.

Recommended URL prefix:

- `/api/v1/...`

Recommended operation mapping:

- list/get style reads: GET
- create/submit actions: POST
- updates: PATCH
- cancels/removals: DELETE or POST custom action when semantically clearer

Custom non-CRUD actions should use colon suffix style for clarity (for example, `:cancel`, `:retry`).

## Validation standards

Validation rules are declared in proto fields via `buf.validate` annotations.

Runtime behavior:

- validation is applied before service business logic
- validation failures return structured field-level details
- service code must not duplicate basic shape validation already defined in proto
- local-mode service implementations must apply the same validation rules as the gRPC server entrypoint

## Pagination defaults

List APIs should use cursor-based pagination defaults:

- `page_size` default: 20
- `page_size` maximum: 100
- `page_token`: opaque continuation token

List responses should return items plus next_page_token metadata.

## Generation and CI policy

Required buf policies:

- lint rules: STANDARD
- breaking detection: FILE against main branch
- dependencies include googleapis and protovalidate definitions as needed

Required generation outputs:

- `gen/go/lobster/v1/*.pb.go`
- `gen/go/lobster/v1/*_grpc.pb.go`
- `gen/go/lobster/v1/*.pb.gw.go`
- `gen/openapi/lobster/v1/*.yaml`
- optional TypeScript client output in `gen/ts/lobster/v1/`

Generated code must never be hand-edited.

Current scope decision:

- TypeScript client generation is deferred until backend API stabilization.
- OpenAPI generation remains required in CI.

## Execution surfaces

Lobster exposes one logical API through multiple client surfaces:

- CLI client over gRPC
- Wish client over gRPC
- HTTP client via gRPC-Gateway

All surfaces map to the same server-side behavior.

## Local execution parity

Local CLI execution is not a separate contract. It must still use the same generated proto messages, request validation, and service boundaries as daemon-backed execution.

Required rules:

- CLI code constructs requests from generated `lobster.v1` types only.
- Local mode calls the same service contracts that the daemon exposes, using an in-process implementation behind that interface.
- Proto validation is enforced before business logic in both local and remote paths.
- buf breaking checks keep the proto contract stable across releases.

## Versioning

- Proto packages must be versioned (example: `lobster.v1.run`).
- Before v1.0, breaking changes may be introduced in minor releases when they are clearly documented.
- After v1.0, breaking changes require a new major proto package.
- Additive fields and RPCs are allowed in the same major package.
- Deprecated fields and RPCs must remain supported for at least one minor release before removal.

Breaking-change cadence rule:

- once Lobster reaches v1.0, breaking contract changes trigger an immediate new major package rather than batching breaks on a schedule.

## Service model

The server exposes four service groups.

1. RunService: start and observe execution.
2. PlanService: compute and inspect execution plans.
3. StackService: manage compose-backed runtime stack behavior.
4. AdminService: health, readiness, and capability reporting.

### RunService

Synchronous workflow:

- `RunSync` starts a run and streams progress events until completion.
- Used by CLI and Wish for immediate interactive feedback.

Asynchronous workflow:

- `RunAsync` creates a run job and returns `run_id`.
- `GetRun` returns current run state.
- `StreamRunEvents` streams progress for a `run_id`.
- `CancelRun` requests cancellation.
- `ListRuns` returns historical run metadata.

Cancellation terminal-state rule:

- `CANCELLED` when graceful stop and cleanup succeed.
- `FAILED` only when cancellation processing fails.

### PlanService

- `Plan` computes a deterministic execution plan.
- `GetPlan` fetches a stored plan artifact.
- `ListPlans` lists recent plans and metadata.

### StackService

- `EnsureStack` prepares compose services and waits for readiness.
- `GetStackStatus` returns current status and health summary.
- `TeardownStack` tears down services according to policy.
- `GetStackLogs` streams service logs for diagnostics.

### AdminService

- `GetHealth` returns liveness/readiness.
- `GetCapabilities` reports server features and versioned API support.
- `GetConfigSummary` returns sanitized effective runtime configuration.

## Authentication and transport security

Baseline security profile:

- mTLS is required for production deployments.
- Bearer token authentication is required in addition to mTLS.
- Token validation may be static-key, JWKS, or custom validator policy.

Default daemon validator mode:

- JWKS URL validation is the default production token-validation mode.

Local development profile:

- TLS and token checks may be relaxed only in explicitly local development mode.
- Local-mode exceptions must be opt-in and clearly logged.
- Default local auth mode remains token-based unless explicitly overridden by local profile policy.

## Error model

gRPC error rules:

- Validation failures return `InvalidArgument`.
- Authentication failures return `Unauthenticated`.
- Authorization failures return `PermissionDenied`.
- Not found returns `NotFound`.
- Conflicts and lock contention return `Aborted`.
- Internal failures return `Internal`.

HTTP mapping is provided by gRPC-Gateway from the same status codes.

HTTP response policy:

- default behavior uses standard gRPC-Gateway error mapping.
- no custom HTTP error envelope is required in the initial contract.

## Event model

Run progress events should include:

- stable run and scenario identifiers
- lifecycle stage transitions
- step-level status updates
- structured diagnostics payloads
- final summary status

Event ordering must be stable per run.

Ordering guarantee:

- per-run total ordering is guaranteed.
- global ordering across different runs is not guaranteed.

## Idempotency and retries

- `RunAsync` should support idempotency keys.
- `CancelRun` should be idempotent.
- Read-only RPCs must be side-effect free.

Idempotency retention default:

- async idempotency keys are retained for 24 hours by default.

## Capability negotiation

Clients should query `GetCapabilities` and adapt behavior when:

- optional APIs are unavailable
- server policy disables a feature
- client/server version mismatch requires fallback behavior

## Non-goals

- This document does not define internal Go package shapes.
- This document does not define SQL schema details.

See [persistence.md](persistence.md) for storage behavior and schema policy.

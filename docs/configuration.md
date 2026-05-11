# Configuration

Lobster uses Viper to resolve configuration from flags, environment variables, and config files.

## Resolution precedence

Highest to lowest:

1. CLI flags
2. Environment variables
3. Config file (lobster.yaml)
4. Defaults

## Config file

Default filename: lobster.yaml

Example:

```yaml
project: payments-e2e
workspace:
  discovery:
    enabled: true
    roots:
      - services/*
  selected: payments
features:
  paths:
    - features/**/*.feature
compose:
  files:
    - docker-compose.yml
    - docker-compose.test.yml
  project_name: payments-e2e
  migrations:
    mode: auto
  profiles:
    - e2e
  wait:
    strategy: healthcheck
    timeout: 120s
execution:
  mode: local
  executor_address: ""
  run_mode: sync
  parallel_scenarios: 1
  fail_fast: false
  soft_assert: false
  keep_stack: false
  timeout: 20m
  step_timeout: 30s
transport:
  security:
    allow_insecure_local: false
  tls:
    enabled: false
    ca_file: ""
    cert_file: ""
    key_file: ""
  auth:
    mode: token
    token_env: LOBSTER_AUTH_TOKEN
    jwks_url: ""
persistence:
  sqlite:
    path: .lobster/lobster.db
    journal_mode: WAL
    busy_timeout: 5s
    synchronous: NORMAL
  plans:
    blob_dir: .lobster/plans
  retention:
    max_runs: 500
    max_age: 2160h
matrix:
  enabled: false
  profiles: []
cache:
  enabled: true
  dir: .lobster/cache
variables:
  suite:
    TENANT: local
http:
  base_url: http://api:8080
  default_headers:
    Accept: application/json
    Content-Type: application/json
hooks:
  before_suite: []
  after_suite: []
  before_scenario: []
  after_scenario: []
data:
  seed:
    strategy: idempotent
  reset:
    mode: per_scenario
quarantine:
  enabled: true
  tag: "@quarantine"
  blocking_in_main_ci: false
reports:
  json: reports/results.json
  junit: reports/junit.xml
  verbose: false
redaction:
  enabled: true
  allowlist: []
telemetry:
  otel:
    enabled: false
    endpoint: http://otel-collector:4317
    service_name: lobster
integrations:
  keycloak:
    enabled: true
    base_url: http://keycloak:8080
    admin_user: admin
    admin_password_env: KEYCLOAK_ADMIN_PASSWORD
steps:
  api_version: v1
  registries:
    - builtin-core
    - keycloak-core
env:
  API_BASE_URL: http://api:8080
```

Notes:

- `parallel_scenarios` remains `1` in v0.1 for deterministic execution.
- Runtime plugin file loading is not part of v0.1 configuration surface.
- `execution.mode` supports `local` and `daemon`.
- `execution.run_mode` supports `sync` and `async` run submission behavior.
- Default global run timeout is 20m.
- Default step timeout is 30s.
- Automatic retries are not enabled in v0.1; use explicit wait/retry steps in feature flows.
- Timeout policy uses both global run timeout and per-step timeout.
- Variable scope includes scenario-scoped and suite-scoped values.
- Reports support scenario-level default output with optional step-level verbose mode.
- Redaction is enabled by default, with explicit allowlist overrides.
- Matrix mode can run one suite across multiple named profiles in one invocation.
- Basic OpenTelemetry trace export is available in v0.1 when configured.
- Migration behavior is configurable per profile: `auto`, `external`, or `disabled`.
- v0.1 default data policy is per-scenario reset with idempotent seeding.
- Flaky tests should be tagged (default `@quarantine`) and routed to separate non-blocking jobs.
- Extension compatibility uses SemVer for lobster plus explicit `steps.api_version` contract checks.
- Monorepo workspace discovery is supported and can be constrained via configured roots.
- Compose profile selection is supported alongside compose file selection.
- Cache is configurable and can be bypassed per run with `--no-cache`.
- SQLite is the canonical datastore for run and plan metadata; JSON/JUnit reports remain file outputs.
- Plan blob payloads are stored in local filesystem under `.lobster/plans` by default, with blob references persisted in SQLite.
- In daemon mode, state access is API-mediated and daemon-owned for writes.
- Default retention baseline is 500 runs and 90 days.
- In monorepos, use one SQLite DB per selected workspace path.
- `transport.security.allow_insecure_local` must remain false by default and be enabled explicitly only for local development.

## Workspace database isolation

When `--workspace` is used, database isolation follows this rule:

- if `persistence.sqlite.path` is explicitly set per profile, that explicit path is used.
- otherwise, Lobster derives an effective path from workspace selection:
  - `.lobster/workspaces/<workspace>/lobster.db`

Example:

- `--workspace payments` -> `.lobster/workspaces/payments/lobster.db`
- `--workspace identity` -> `.lobster/workspaces/identity/lobster.db`

This keeps one SQLite database per selected workspace path in monorepo usage.

## Local authentication defaults

Default local behavior:

- `transport.auth.mode` remains `token` unless explicitly overridden.
- when `transport.security.allow_insecure_local: true`, TLS and strict remote security checks may be relaxed for local development only.
- insecure-local mode must never be enabled in CI or production profiles.

See docs/persistence.md for full migration, compatibility, and retention behavior.

## Environment variables

Environment variable prefix: LOBSTER_

Examples:

- LOBSTER_PROJECT
- LOBSTER_FEATURES_PATHS
- LOBSTER_COMPOSE_FILES
- LOBSTER_EXECUTION_FAIL_FAST
- LOBSTER_EXECUTION_MODE
- LOBSTER_EXECUTION_EXECUTOR_ADDRESS
- LOBSTER_EXECUTION_RUN_MODE
- LOBSTER_REPORTS_JUNIT
- LOBSTER_PERSISTENCE_SQLITE_PATH
- LOBSTER_AUTH_TOKEN

For secrets, prefer environment variables over plain text in lobster.yaml.

For local development, plain-text secrets in config may be allowed by project policy, but CI environments should still use environment variables or secret managers.

Example:

```bash
export KEYCLOAK_ADMIN_PASSWORD="super-secret"
export LOBSTER_REPORTS_JUNIT="reports/junit.xml"
```

## Recommended profiles

### Local development

- fail_fast: false
- keep_stack: true while debugging
- use verbose console logs for step-level troubleshooting

### CI pipeline

- fail_fast: true for fast feedback
- keep_stack: false
- ci mode enabled
- report paths set for artifact collection
- keep `parallel_scenarios: 1` for reproducible runs in v0.1

## Validation

Validate config before running tests:

```bash
lobster config --validate
```

Print effective config:

```bash
lobster config --print --format json
```

For ready-to-use profile templates, see docs/config-profiles.md.

## Terminology note

Use explicit profile naming in docs and config discussions:

- config profile: lobster execution behavior profile (local, ci, debug)
- compose profile: docker compose service-profile selection
- matrix profile: profile entry used in one matrix run expansion

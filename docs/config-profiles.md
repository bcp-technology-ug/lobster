# Configuration Profiles

This document provides recommended profile templates for local development, CI, and debugging.

Terminology:

- config profile: lobster execution behavior profile in this document
- compose profile: docker compose service-selection profile
- matrix profile: profile identifier expanded by one matrix run

## local profile

Use this profile for day-to-day development and fast iteration.

```yaml
project: lobster-local
features:
  paths:
    - features/**/*.feature
compose:
  files:
    - docker-compose.yml
execution:
  mode: local
  executor_address: ""
  run_mode: sync
  parallel_scenarios: 1
  fail_fast: false
  soft_assert: false
  keep_stack: true
  timeout: 30m
  step_timeout: 45s
variables:
  suite:
    ENV: local
http:
  base_url: http://api:8080
  default_headers:
    Accept: application/json
reports:
  json: reports/local-results.json
  junit: reports/local-junit.xml
  verbose: false
redaction:
  enabled: true
  allowlist: []
telemetry:
  otel:
    enabled: false
    endpoint: http://otel-collector:4317
    service_name: lobster-local
transport:
  security:
    allow_insecure_local: true # Development only; do not enable in CI or production
  tls:
    enabled: false
  auth:
    mode: token
    token_env: LOBSTER_AUTH_TOKEN
    jwks_url: ""
persistence:
  sqlite:
    path: .lobster/lobster.db
  retention:
    max_runs: 500
    max_age: 2160h
```

## ci profile

Use this profile for deterministic pipeline execution.

```yaml
project: lobster-ci
features:
  paths:
    - features/**/*.feature
compose:
  files:
    - docker-compose.yml
    - docker-compose.ci.yml
execution:
  mode: daemon
  executor_address: dns:///lobsterd.internal:9443
  run_mode: sync
  parallel_scenarios: 1
  fail_fast: true
  soft_assert: false
  keep_stack: false
  timeout: 20m
  step_timeout: 30s
matrix:
  enabled: true
  profiles:
    - ci
    - ci-hardened
variables:
  suite:
    ENV: ci
reports:
  json: reports/ci-results.json
  junit: reports/ci-junit.xml
  verbose: false
redaction:
  enabled: true
  allowlist: []
telemetry:
  otel:
    enabled: true
    endpoint: http://otel-collector:4317
    service_name: lobster-ci
transport:
  security:
    allow_insecure_local: false
  tls:
    enabled: true
    ca_file: /etc/ssl/certs/ca-bundle.crt
  auth:
    mode: token
    token_env: LOBSTER_AUTH_TOKEN
    jwks_url: https://auth.internal/.well-known/jwks.json
persistence:
  sqlite:
    path: /var/lib/lobster/lobster.db
  retention:
    max_runs: 500
    max_age: 2160h
```

## debug profile

Use this profile when investigating difficult failures.

```yaml
project: lobster-debug
features:
  paths:
    - features/**/*.feature
compose:
  files:
    - docker-compose.yml
    - docker-compose.debug.yml
execution:
  mode: daemon
  executor_address: dns:///lobsterd.internal:9443
  run_mode: sync
  parallel_scenarios: 1
  fail_fast: false
  soft_assert: true
  keep_stack: true
  timeout: 45m
  step_timeout: 90s
variables:
  suite:
    ENV: debug
reports:
  json: reports/debug-results.json
  junit: reports/debug-junit.xml
  verbose: true
redaction:
  enabled: true
  allowlist: []
telemetry:
  otel:
    enabled: true
    endpoint: http://otel-collector:4317
    service_name: lobster-debug
transport:
  security:
    allow_insecure_local: false
  tls:
    enabled: true
    ca_file: /etc/ssl/certs/ca-bundle.crt
  auth:
    mode: token
    token_env: LOBSTER_AUTH_TOKEN
    jwks_url: https://auth.internal/.well-known/jwks.json
persistence:
  sqlite:
    path: /var/lib/lobster/lobster.db
  retention:
    max_runs: 1000 # Increased retention for deep failure investigations
    max_age: 4320h # Increased retention for deep failure investigations
```

## profile usage

Use profile-specific files by selecting them in your command or workflow policy.

Example patterns:

- local runs use local profile values with stack retention
- CI runs use ci profile values with strict fail-fast behavior
- debug runs use verbose reports and longer timeouts
- debug runs also use increased retention (1000 runs / 4320h) to preserve investigation history

## policy notes

- Redaction stays enabled by default in all profiles.
- Plain-text secrets may be tolerated for local-only workflows, but CI should prefer environment-variable or secret-manager injection.
- Keep `parallel_scenarios: 1` in v0.1 for deterministic behavior.
- In daemon mode, persistence writes are daemon-owned; clients should not write SQLite directly.
- Default retention baseline is 500 runs and 90 days unless a profile explicitly overrides it.

# Getting Started

This guide walks through installing lobster, creating your first project, and running your first end-to-end BDD test.

## Prerequisites

- Go 1.22 or newer
- Docker with Compose v2
- A project that can be started with Docker Compose
- Optional: network path and credentials for a remote Lobster daemon host

## Install lobster

```bash
go install github.com/bcp-technology-ug/lobster@latest
```

Verify installation:

```bash
lobster --version
```

## Initialize a new test project

```bash
lobster init my-system-tests
cd my-system-tests
```

Expected starter layout:

```text
my-system-tests/
  lobster.yaml
  docker-compose.yml
  features/
    smoke/
      health.feature
```

## Create a feature file

Create features/smoke/health.feature:

```gherkin
Feature: Service health
  Scenario: API health endpoint responds
    Given the service "api" is running
    When I send a GET request to "/health"
    Then the response status should be 200
    And the response body should contain "ok"
```

## Validate and lint

Run syntax and quality checks before execution:

```bash
lobster plan
lobster validate
lobster lint
lobster config --validate
```

## Plan before execution

Generate and save a plan artifact:

```bash
lobster plan --workspace my-service --out plans/smoke.lobsterplan.json
```

Note:

- `--out` overrides the default plan blob directory (`.lobster/plans`).
- If `--out` is omitted, Lobster writes the plan to the configured default blob directory.

Execute directly from the plan:

```bash
lobster run --from-plan plans/smoke.lobsterplan.json
```

## Run tests against your full stack

```bash
lobster run
```

## Run against a remote daemon host

Start daemon on the target host:

```bash
lobsterd start --listen :9443 --http-listen :8080 --db-path /var/lib/lobster/lobster.db
```

Submit a synchronous remote run from your client:

```bash
lobster run \
  --executor-mode daemon \
  --executor-addr dns:///lobsterd.internal:9443 \
  --run-mode sync
```

Submit an asynchronous remote run:

```bash
lobster run --executor-mode daemon --executor-addr dns:///lobsterd.internal:9443 --run-mode async
```

Security note:

- For production daemon usage, include auth and TLS flags such as `--auth-token`, `--tls-ca-file`, `--tls-cert-file`, and `--tls-key-file`.
- See docs/api-reference.md for production security policy (mTLS plus bearer token).

What run does:

1. Loads lobster configuration
2. Starts Docker Compose services
3. Waits for service readiness
4. Executes all matching scenarios
5. Produces summary and report files
6. Tears down stack according to policy

## Use in CI mode

For non-interactive pipeline runs:

```bash
lobster run --ci --report-junit reports/junit.xml --report-json reports/results.json
```

## Next steps

- Read docs/configuration.md to tune environment and execution behavior
- Read docs/step-definitions.md to use built-in steps and extend the static registry
- Read docs/integrations.md to configure adapters like Keycloak
- Read docs/api-reference.md for transport contracts
- Read docs/persistence.md for SQLite and sqlc behavior
- Read docs/testing.md for unit, integration, E2E, and dogfooding policy

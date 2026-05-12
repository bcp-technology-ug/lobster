# CI/CD Usage

Lobster is designed for non-interactive, pipeline-first execution.

## CI goals

- deterministic orchestration and test execution
- machine-readable reports for dashboards and quality gates
- stable exit codes for pipeline decisions

## Generic CI command

Use staged execution so failures appear in the right phase.

Contract generation stage:

```bash
buf generate
sqlc generate
```

Candidate build stage (dogfooding baseline):

```bash
go build -o bin/lobster ./cmd/lobster
go build -o bin/lobsterd ./cmd/lobsterd
```

Contract validation stage:

```bash
buf lint
buf breaking --against '.git#branch=main'
```

Lobster validation stage:

```bash
./bin/lobster config --validate
./bin/lobster lint
```

Execution stage:

```bash
./bin/lobster run \
  --ci \
  --report-junit reports/junit.xml \
  --report-json reports/results.json
```

Dogfooding stage:

```bash
./bin/lobster run --ci --features "tests/integration/**/*.feature"
./bin/lobster run --ci --features "tests/e2e/**/*.feature"
```

Remote daemon CI pattern:

```bash
buf generate
sqlc generate
buf lint
buf breaking --against '.git#branch=main'
lobster config --validate
lobster lint
lobster run \
  --ci \
  --executor-mode daemon \
  --executor-addr dns:///lobsterd.internal:9443 \
  --run-mode sync \
  --report-junit reports/junit.xml \
  --report-json reports/results.json
```

Production security note for daemon CI:

- Add `--auth-token`, `--tls-ca-file`, `--tls-cert-file`, and `--tls-key-file` for mTLS plus bearer-authenticated connections.
- See docs/api-reference.md for baseline daemon security requirements.

v0.1 execution defaults for CI:

- Serial scenario execution for deterministic results
- Structured console logs for pipeline readability
- Explicit non-zero exit codes for gating
- No automatic retries; feature/authored wait logic should handle eventual consistency
- Global run timeout and per-step timeout both enforced
- Sensitive values redacted by default in logs and reports

Recommended failure artifacts:

- last N lines of relevant container logs
- resolved effective config snapshot
- expanded scenario execution trace
- failed service/container state dump

For daemon mode, include daemon health summary and API capability snapshot in failure artifacts.

Report detail modes:

- Default CI mode should publish scenario-level reports.
- Step-level verbose reporting can be enabled when debugging flaky failures.

Optional CI expansions in v0.1:

- matrix-style profile runs (for example: staging and production-like test profiles)
- basic OpenTelemetry trace export to a configured collector

Required security checks in v0.1 CI:

- `go vet`
- `staticcheck`
- `govulncheck`
- dependency license scan
- container image scan for test fixtures

Required contract checks in CI:

- generation commands must run and leave no unstaged generated diffs
- proto lint must pass
- proto breaking checks must pass or be explicitly versioned
- sqlc generation must reflect current SQL and migration sources
- OpenAPI generation must be present and current
- generated artifacts must be committed and validated for drift

Required dogfooding checks in CI:

- integration suites must be executed by the candidate `./bin/lobster` binary
- E2E suites must be executed by the candidate `./bin/lobster` binary
- failures in self-hosted suites block merge
- at least one cross-mode contract suite must cover the same logical flow in local mode and daemon mode when both are available
- contract parity failures between local and daemon execution paths must block merge
- cross-mode parity is a required blocking gate, not a best-effort recommendation

Remote security baseline in CI:

- use mTLS for daemon connections
- use bearer token auth for API calls
- prefer short-lived credentials via CI secret store

Flaky-test policy in v0.1:

- Do not auto-quarantine tests.
- Use explicit `@quarantine` tagging.
- Run quarantined scenarios in a separate non-blocking CI job.

Migration policy in CI:

- Migration behavior should be profile-configured (`auto`, `external`, `disabled`) to match each application stack.

## GitHub Actions example

```yaml
name: e2e
on:
  push:
  pull_request:

jobs:
  lobster-e2e:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install lobster
        run: go install github.com/bcp-technology-ug/lobster@latest

      - name: Run lobster
        run: |
          lobster validate
          lobster run --ci --report-junit reports/junit.xml --report-json reports/results.json

      - name: Upload reports
        uses: actions/upload-artifact@v4
        with:
          name: lobster-reports
          path: reports/
```

## GitLab CI example

```yaml
stages:
  - test

lobster_e2e:
  stage: test
  image: golang:1.22
  services:
    - docker:dind
  variables:
    DOCKER_HOST: tcp://docker:2375
    DOCKER_TLS_CERTDIR: ""
  script:
    - go install github.com/bcp-technology-ug/lobster@latest
    - lobster validate
    - lobster run --ci --report-junit reports/junit.xml --report-json reports/results.json
  artifacts:
    when: always
    paths:
      - reports/
```

## Exit code behavior

Suggested semantics:

- 0: all tests passed
- 1: one or more scenarios failed
- 2: validation or config failure
- 3: infrastructure startup failure
- 4: unexpected internal runtime error

## CI best practices

- Pin image versions for reproducibility
- Keep feature suites deterministic and isolated
- Collect artifacts on both success and failure
- Fail fast on startup/validation errors
- Use tag filters to split slow suites across jobs
- Keep `@quarantine` scenarios out of main blocking quality gates
- Fail fast on contract drift between source files and generated outputs

Release and compatibility notes:

- Maintain `stable` and `nightly` release channels.
- Before v1.0, minor versions may include breaking changes; communicate clearly in release notes and changelogs.
- After v1.0, breaking changes must move to a new major version.
- Before v1.0, deprecations should still be called out in release notes and changelogs when practical.
- After v1.0, deprecations should produce runtime warnings and document target removal versions.

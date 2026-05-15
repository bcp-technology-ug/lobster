# Lobster — AI Agent Guide

Lobster is a CLI-first, BDD end-to-end test runner that executes Gherkin `.feature` files against real infrastructure (Docker Compose stacks, live APIs). Tests are declarative — you write `Given/When/Then` scenarios using built-in step patterns; no custom Go code is needed for the common case.

---

## Bootstrap: Do This First

Before writing any feature file, run this command to get the complete list of available steps:

```sh
lobster steps --format markdown
```

**Only use step patterns returned by that command.** Never invent or guess step text — lobster will fail with `ErrUndefined` at runtime.

---

## The 6-Step Workflow

```sh
# 1. Discover what steps are available
lobster steps --format markdown

# 2. Write .feature files using only those patterns
#    (see Feature File Rules below)

# 3. Validate syntax
lobster validate --features 'features/**/*.feature'

# 4. Check quality / style
lobster lint --features 'features/**/*.feature'

# 5. Dry-run — see which scenarios will execute
lobster plan --features 'features/**/*.feature' --tags @mytag

# 6. Execute
lobster run --features 'features/**/*.feature' --tags @mytag
```

Always run steps 3 and 4 after writing or modifying feature files. Never skip `validate` — it catches undefined steps before they fail at runtime.

---

## Step Categories

| Category | Filter | What it covers |
|---|---|---|
| HTTP | `--filter http` | REST requests, response assertions, auth headers |
| Shell | `--filter shell` | CLI commands, exit codes, stdout/stderr assertions |
| Filesystem | `--filter fs` | File/dir create, read, assert, temp directory lifecycle |
| Service | `--filter service` | Wait-for-service-ready, TCP port health checks |
| gRPC | `--filter grpc` | gRPC health protocol checks |
| Variables | `--filter vars` | Set/read/assert named variables, UUID generation |
| Wait/Retry | `--filter wait` | Sleep, poll until condition, retry with backoff |
| JSON Assert | `--filter assert` | JSONPath field assertions on the last HTTP response |

---

## Feature File Rules

```gherkin
@tag1 @tag2
Feature: Short description
  Optional longer description.

  Background:
    Given I am in a new temporary directory   # runs before every scenario

  Scenario: Concrete example name
    Given <setup>
    When  <action>
    Then  <assertion>

  Scenario Outline: Parameterised test <param>
    Given I set the base URL to "<url>"
    When  I send a GET request to "/status"
    Then  the response status should be <code>

    Examples:
      | url                    | code |
      | http://staging.example | 200  |
      | http://prod.example    | 200  |
```

**Rules:**
- One `Feature:` per file.
- Step text must match a pattern from `lobster steps` exactly (case-sensitive, same spacing).
- DocString steps end with `:` — the body is indented `""" ` blocks on the next lines.
- Variable interpolation: use `${VAR_NAME}` anywhere inside quoted step arguments.
- Tag scenarios with `@smoke`, `@integration`, `@e2e`, `@quarantine` as appropriate.

---

## Common Patterns

### HTTP API test
```gherkin
Feature: User API

  Background:
    Given I set the base URL to "http://localhost:8080"
    And I set the bearer token "${AUTH_TOKEN}"

  Scenario: Create user
    When I send a POST request to "/api/users" with body:
      """json
      {"name": "Alice", "email": "alice@example.com"}
      """
    Then the response status should be 201
    And the response JSON field "$.id" should exist
    And I store the response body in variable "CREATE_RESPONSE"
```

### CLI tool test
```gherkin
Feature: lobster init command

  Scenario: Initialises a project
    Given I am in a new temporary directory
    When I run the command "lobster init --project my-app --no-interactive"
    Then the exit code should be 0
    And the file "lobster.yaml" should exist
    And the file "lobster.yaml" should contain "my-app"
```

### Service readiness
```gherkin
Background:
  Given I wait up to 30s for the service "api" to be running
  And I wait up to 30s for TCP port "5432" on "localhost" to be open
```

### Variable chaining
```gherkin
Scenario: Authenticate then fetch profile
  Given I set variable "USER_EMAIL" to "alice@example.com"
  When I send a POST request to "/auth/login" with body:
    """json
    {"email": "${USER_EMAIL}", "password": "secret"}
    """
  Then the response status should be 200
  And I store JSON field "$.token" from the response in variable "TOKEN"
  When I set the bearer token "${TOKEN}"
  And I send a GET request to "/me"
  Then the response status should be 200
```

---

## lobster.yaml Key Fields

```yaml
project: my-app
features:
  paths:
    - features/**/*.feature
compose:
  files:
    - docker-compose.yaml
  migrations:
    mode: auto        # auto|external|disabled
http:
  base_url: http://localhost:8080
variables:
  AUTH_TOKEN: ""      # override per environment via --env AUTH_TOKEN=xxx
execution:
  fail_fast: false
  soft_assert: false
  timeouts:
    scenario: 60s
    step: 10s
```

---

## Useful Flags

```sh
# Run only scenarios matching a tag expression
lobster run --features 'features/**/*.feature' --tags '@smoke and not @slow'

# Run a specific scenario by name regex
lobster run --features 'features/**/*.feature' --scenario-regex "Create user"

# Pass env variables to the run
lobster run --features 'features/**/*.feature' --env AUTH_TOKEN=abc123 --env BASE_URL=https://staging.example.com

# Keep the Docker Compose stack running after the run (for debugging)
lobster run --features 'features/**/*.feature' --keep-stack

# Generate reports
lobster run --features 'features/**/*.feature' --report-junit results.xml --report-json results.json

# Soft-assert mode: collect all failures per scenario instead of stopping at first
lobster run --features 'features/**/*.feature' --soft-assert

# Verbose output
lobster run --features 'features/**/*.feature' -vv

# Check environment health before running
lobster doctor
```

---

## Exit Codes

| Code | Meaning |
|---|---|
| 0 | All scenarios passed |
| 1 | One or more scenarios failed |
| 2 | Configuration or validation error |
| 3 | Docker Compose / orchestration error |
| 4 | Internal runtime error |

---

## Do Not

- Do not invent step patterns. If a step does not exist in `lobster steps` output, use a different approach or ask the user to add a custom step.
- Do not add custom Go code to make a single test work — lobster's built-in steps cover the vast majority of E2E testing needs.
- Do not run `lobster run` without first running `lobster validate` — always validate first.
- Do not use `--keep-stack` in CI — it leaks containers.

---

## Getting Help

```sh
lobster --help
lobster steps --help
lobster run --help
lobster doctor
```

Full docs: `docs/` directory or https://github.com/bcp-technology-ug/lobster

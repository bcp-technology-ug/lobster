---
applyTo: "**/*.feature"
---
# Lobster — VS Code Copilot Instructions

Lobster is a CLI-first BDD end-to-end test runner. Feature files use Gherkin syntax and execute against real infrastructure via Docker Compose.

## Before writing any `.feature` file

Run in terminal to get the full step catalog:
```sh
lobster steps --format markdown
```

**Use only patterns from that output.** Copilot should suggest completions from the step catalog, not invent new patterns.

## Recommended workflow

```sh
lobster steps --format markdown        # discover steps
lobster validate --features 'features/**/*.feature'  # after writing
lobster lint     --features 'features/**/*.feature'  # style check
lobster plan     --features 'features/**/*.feature'  # dry run
lobster run      --features 'features/**/*.feature'  # execute
```

## Feature file conventions

- One `Feature:` block per `.feature` file
- Steps are `Given` (setup), `When` (action), `Then` (assertion); use `And`/`But` for continuation
- Tag scenarios with `@smoke`, `@e2e`, `@integration`, `@quarantine`
- Variable interpolation: `${VAR_NAME}` inside any quoted string argument
- DocString steps end with `:` followed by an indented `"""` block
- DataTable steps end with `:` followed by a `| col | col |` table

## Key step patterns (summary — run `lobster steps` for the full list)

```
# HTTP
I send a (GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS) request to "<path>"
I send a (GET|...) request to "<path>" with body:          ← DocString
I set the request header "<name>" to "<value>"
I set the bearer token "<token>"
I set the base URL to "<url>"
the response status should be <code>
the response JSON field "<jsonpath>" should equal "<value>"
I store the response body in variable "<name>"
I store JSON field "<jsonpath>" from the response in variable "<name>"

# Shell
I run the command "<cmd>"
the exit code should be <n>
the output should contain "<text>"
I store the output in variable "<name>"

# Filesystem
I am in a new temporary directory
I create the file "<path>" with content:                   ← DocString
the file "<path>" should exist
the file "<path>" should contain "<text>"

# Service
I wait up to <n>s for the service "<name>" to be running
I wait up to <n>s for URL "<url>" to be reachable
the TCP port "<port>" on "<host>" should be open

# Variables
I set variable "<name>" to "<value>"
I set variable "<name>" to a random UUID
the variable "<name>" should equal "<value>"

# Wait
I wait <n> second(s)
I poll "<url>" every <n>s until the status is <code> for up to <m>s
```

## lobster.yaml minimum config

```yaml
project: my-app
features:
  paths:
    - features/**/*.feature
http:
  base_url: http://localhost:8080
```

## Common mistakes to avoid

- Do NOT invent step text — only use patterns from `lobster steps` output
- Do NOT skip `lobster validate` — run it every time after editing feature files
- Do NOT use `--keep-stack` in CI pipelines
- Tag with `@quarantine` to isolate flaky tests rather than deleting them

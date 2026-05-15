---
mode: agent
description: Debug a failing lobster scenario — diagnose the error and propose a fix
---

You are helping diagnose a failing lobster test scenario.

## Step 1: Get the failure details

Ask the user to provide (or read from the current context):
1. The full error output from `lobster run`
2. The `.feature` file containing the failing scenario
3. The `lobster.yaml` config

## Step 2: Check for undefined steps

If the error contains `ErrUndefined` or "undefined step":

```sh
lobster steps --format markdown
```

Compare the failing step text to the output. Common causes:
- Extra/missing spaces in step text
- Wrong capitalisation (step matching is case-sensitive)
- A pattern changed between lobster versions
- The step belongs to a category not registered in `lobster.yaml` `steps.registries`

## Step 3: Validate and lint the feature file

```sh
lobster validate --features 'features/**/*.feature'
lobster lint     --features 'features/**/*.feature'
```

Fix all errors before attempting to run again.

## Step 4: Check environment

```sh
lobster doctor
```

Verify:
- Docker / Docker Compose is running
- Required services are reachable
- `lobster.yaml` is valid

## Step 5: Isolate the failing scenario

Run only the failing scenario using a tag or name regex:

```sh
lobster run --features 'path/to/failing.feature' --scenario-regex "Failing scenario name" --keep-stack -vv
```

The `--keep-stack` flag keeps the Docker Compose stack running after the failure so you can inspect state.

## Step 6: Inspect and fix

Common failure patterns:

| Symptom | Likely cause | Fix |
|---|---|---|
| `ErrUndefined` | Step text doesn't match any pattern | Check `lobster steps` output, fix step text |
| `status 401` when expecting 200 | Missing auth header | Add `I set the bearer token "${TOKEN}"` in Background |
| `status 404` | Wrong path or service not running | Check base URL and service health |
| Exit code 3 | Docker Compose stack failed to start | Run `lobster doctor`, check compose file |
| Variable `${X}` not expanded | Variable not set before use | Add `I set variable "X" to "..."` in Background |
| Flaky timing | Service not ready when test starts | Use `I wait up to Ns for the service "name" to be running` |

## Step 7: Show the fix

Present the corrected `.feature` file and explain what changed.

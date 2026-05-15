---
mode: agent
description: Scaffold a new Gherkin feature file for a lobster test project
---

You are helping write a new Gherkin feature file for a lobster BDD test project.

## Step 1: Discover available steps

Run this command first and read the full output before writing a single line of Gherkin:

```sh
lobster steps --format markdown
```

Only use step patterns that appear in that output.

## Step 2: Ask clarifying questions (if not already known)

Before writing, confirm:
1. What feature/capability are we testing? (one sentence)
2. What HTTP service(s) or CLI tools are involved?
3. What is the base URL? (e.g. `http://localhost:8080`)
4. Does this test require authentication? If so, what kind (bearer token, basic auth)?
5. What tags should this scenario have? (`@smoke`, `@e2e`, `@integration`)
6. Are there any variables that should come from the environment (e.g. `${AUTH_TOKEN}`, `${BASE_URL}`)?

## Step 3: Write the feature file

Follow these rules:
- One `Feature:` block per file
- Use `Background:` for setup steps shared across all scenarios (base URL, auth, service readiness)
- Steps: `Given` = setup, `When` = action, `Then` = assertion; `And`/`But` for continuation
- All quoted arguments support `${VAR_NAME}` interpolation
- DocString steps end with `:` and have an indented `"""` block
- Place the file in the `features/` directory

## Step 4: Validate, lint, plan

After writing, run in sequence:

```sh
lobster validate --features 'features/**/*.feature'
lobster lint     --features 'features/**/*.feature'
lobster plan     --features 'features/**/*.feature'
```

Fix any errors reported by validate before proceeding. Address lint warnings.

## Step 5: Show the user

Present the complete `.feature` file content and confirm the plan output looks correct.

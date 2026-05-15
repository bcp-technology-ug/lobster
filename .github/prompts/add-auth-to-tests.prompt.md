---
mode: agent
description: Add authentication steps to existing lobster feature files
---

You are helping add authentication to existing lobster feature files.

## Step 1: Identify the auth mechanism

Ask the user which auth type is needed:
- **Bearer token** (OAuth2 / JWT) — most common for REST APIs
- **Basic auth** (username + password)
- **API key header** (e.g. `X-API-Key`)
- **Keycloak** (lobster has a built-in Keycloak integration adapter)

## Step 2: Check available auth steps

```sh
lobster steps --filter http --format markdown
```

Key auth-related patterns:
```
I set the bearer token "([^"]+)"
I set the basic auth username "([^"]+)" and password "([^"]+)"
I set the request header "([^"]+)" to "([^"]+)"
```

## Step 3: Bearer token pattern

If the service issues tokens via an endpoint, chain the token acquisition into the Background:

```gherkin
Feature: Protected resource

  Background:
    Given I set the base URL to "${BASE_URL}"
    And I wait up to 30s for URL "${BASE_URL}/healthz" to be reachable
    # Acquire token
    When I send a POST request to "/auth/token" with body:
      """json
      {"client_id": "${CLIENT_ID}", "client_secret": "${CLIENT_SECRET}"}
      """
    Then the response status should be 200
    And I store JSON field "$.access_token" from the response in variable "TOKEN"
    And I set the bearer token "${TOKEN}"

  Scenario: Access protected resource
    When I send a GET request to "/api/protected"
    Then the response status should be 200
```

Supply credentials via `lobster.yaml` variables or `--env` flags — never hardcode secrets in feature files.

## Step 4: Basic auth pattern

```gherkin
Background:
  Given I set the base URL to "${BASE_URL}"
  And I set the basic auth username "${API_USER}" and password "${API_PASS}"
```

## Step 5: API key header pattern

```gherkin
Background:
  Given I set the base URL to "${BASE_URL}"
  And I set the request header "X-API-Key" to "${API_KEY}"
```

## Step 6: Update lobster.yaml for credentials

Add variables to `lobster.yaml` (leave values empty, override at runtime):

```yaml
variables:
  BASE_URL: "http://localhost:8080"
  CLIENT_ID: ""
  CLIENT_SECRET: ""
  API_KEY: ""
```

Run with real values:
```sh
lobster run --features 'features/**/*.feature' \
  --env CLIENT_ID=my-client \
  --env CLIENT_SECRET=secret \
  --env API_KEY=abc123
```

## Step 7: Validate

```sh
lobster validate --features 'features/**/*.feature'
lobster lint     --features 'features/**/*.feature'
```

## Step 8: Show the updated feature file

Present the complete updated feature file with auth steps integrated.

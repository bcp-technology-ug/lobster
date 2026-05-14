@docker @covers:AdminService.GetHealth @covers:AdminService.GetCapabilities
Feature: lobsterd Docker Compose — daemon startup and health
  As a developer
  I want docker compose to start lobsterd and expose a healthy daemon
  So that I can verify the image builds and the process boots correctly

  # ── Prerequisites ─────────────────────────────────────────────────────────
  # Requires Docker and docker compose to be available on the host.
  # Run from the repo root: lobster run --features tests/features/docker-*.feature --tags "@docker" --ci

  Background:
    Given I am in a new temporary directory

  # ── Image build ───────────────────────────────────────────────────────────

  Scenario: docker compose build succeeds without errors
    When I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml build --no-cache 2>&1"
    Then the exit code should be 0
    And the output should not contain "ERROR"

  # ── Daemon startup ────────────────────────────────────────────────────────

  Scenario: docker compose up starts the lobsterd container
    When I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml up -d 2>&1"
    Then the exit code should be 0
    And the output should contain "lobsterd"

  Scenario: lobsterd container reaches running state
    Given I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml up -d 2>&1"
    When I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml ps --format json 2>&1"
    Then the exit code should be 0
    And the output should contain "running"

  # ── Health endpoint ───────────────────────────────────────────────────────

  Scenario: /healthz responds with HTTP 200 after startup
    Given I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml up -d 2>&1"
    And I set the base URL to "http://localhost:8080"
    When I wait up to 30s for the service "healthz" to be running
    Then the response status should be 200

  Scenario: /healthz response body is valid JSON
    Given I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml up -d 2>&1"
    And I set the base URL to "http://localhost:8080"
    When I wait up to 30s for the service "healthz" to be running
    And I send a GET request to "/healthz"
    Then the response status should be 200
    And the response body should be valid JSON

  Scenario: /healthz response contains expected fields
    Given I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml up -d 2>&1"
    And I set the base URL to "http://localhost:8080"
    When I wait up to 30s for the service "healthz" to be running
    And I send a GET request to "/healthz"
    Then the response body should contain "lobsterd"
    And the response body should contain "ready"
    And the response body should contain "version"

  # ── Teardown ─────────────────────────────────────────────────────────────

  Scenario: docker compose down stops the daemon cleanly
    Given I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml up -d 2>&1"
    When I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml down 2>&1"
    Then the exit code should be 0

  Scenario: docker compose down with -v removes the data volume
    Given I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml up -d 2>&1"
    When I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml down -v 2>&1"
    Then the exit code should be 0
    And the output should contain "lobsterd-data"

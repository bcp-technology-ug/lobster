@docker
Feature: lobsterd Docker Compose — persistence and restart resilience
  As a developer
  I want the SQLite database to survive container restarts
  So that run history is not lost between daemon restarts

  Background:
    Given I am in a new temporary directory

  # ── Data persistence across restart ──────────────────────────────────────

  Scenario: Run history is preserved after daemon restart
    Given I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml up -d 2>&1"
    And I set the base URL to "http://localhost:8080"
    And I wait up to 30s for the service "healthz" to be running
    And I create the file "features/persist.feature" with content:
      """
      Feature: Persistence test
        Scenario: Submitted before restart
          Given I am in a new temporary directory
          When I run the command "echo persist-ok"
          Then the exit code should be 0
      """
    And I run lobster "run --features features/persist.feature --run-mode async --executor-mode daemon --executor-addr localhost:9443 --insecure-local --ci"
    And I store the output in variable "PERSIST_RUN_ID"
    And I run the command "sleep 3"
    # Restart the daemon; the named volume keeps the SQLite database.
    And I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml restart lobsterd 2>&1"
    And I wait up to 30s for the service "healthz" to be running
    When I run lobster "run status --run-id ${PERSIST_RUN_ID} --executor-mode daemon --executor-addr localhost:9443 --insecure-local"
    Then the exit code should be 0

  # ── Volume wipe ───────────────────────────────────────────────────────────

  Scenario: docker compose down -v removes the data volume
    Given I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml up -d 2>&1"
    And I wait up to 30s for the service "healthz" to be running
    When I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml down -v 2>&1"
    Then the exit code should be 0
    And the output should contain "lobsterd-data"

  # ── Fresh start after volume wipe ─────────────────────────────────────────

  Scenario: Daemon starts cleanly on an empty volume
    Given I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml down -v 2>&1"
    When I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml up -d 2>&1"
    And I set the base URL to "http://localhost:8080"
    And I wait up to 30s for the service "healthz" to be running
    Then I send a GET request to "/healthz"
    And the response status should be 200

  # ── Migration auto-mode ───────────────────────────────────────────────────

  Scenario: Daemon runs migrations automatically on a brand-new database
    Given I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml down -v 2>&1"
    And I run the command "docker compose -f ${LOBSTER_ROOT}/config/docker-compose.yml up -d 2>&1"
    And I set the base URL to "http://localhost:8080"
    When I wait up to 30s for the service "healthz" to be running
    And I send a GET request to "/healthz"
    Then the response body should contain "ready"

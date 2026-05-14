@daemon @covers:cli:admin:health @covers:cli:admin:capabilities @covers:cli:admin:config @covers:AdminService.GetHealth @covers:AdminService.GetCapabilities @covers:AdminService.GetConfigSummary
Feature: lobster admin command
  As an operator
  I want to query daemon health, capabilities, and config from the CLI
  So that I can verify daemon state without HTTP tooling

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  # ── admin health ──────────────────────────────────────────────────────────

  Scenario: lobster admin health --help exits 0
    When I run lobster "admin health --help"
    Then the exit code should be 0

  Scenario: lobster admin health against live daemon exits 0
    When I run lobster "admin health --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: lobster admin health output contains version
    When I run lobster "admin health --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0
    And the output should contain "Version"

  Scenario: lobster admin health --format json exits 0
    When I run lobster "admin health --format json --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: lobster admin health --format json output is valid JSON
    When I run lobster "admin health --format json --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: lobster admin health against unreachable daemon exits non-zero
    When I run lobster "admin health --executor-addr localhost:1"
    Then the exit code should not be 0

  # ── admin capabilities ────────────────────────────────────────────────────

  Scenario: lobster admin capabilities --help exits 0
    When I run lobster "admin capabilities --help"
    Then the exit code should be 0

  Scenario: lobster admin capabilities against live daemon exits 0
    When I run lobster "admin capabilities --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: lobster admin capabilities output contains api
    When I run lobster "admin capabilities --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0
    And the output should contain "API"

  Scenario: lobster admin capabilities --format json is valid JSON
    When I run lobster "admin capabilities --format json --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: lobster admin capabilities against unreachable daemon exits non-zero
    When I run lobster "admin capabilities --executor-addr localhost:1"
    Then the exit code should not be 0

  # ── admin config ──────────────────────────────────────────────────────────

  Scenario: lobster admin config --help exits 0
    When I run lobster "admin config --help"
    Then the exit code should be 0

  Scenario: lobster admin config against live daemon exits 0
    When I run lobster "admin config --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: lobster admin config --format json is valid JSON
    When I run lobster "admin config --format json --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: lobster admin config against unreachable daemon exits non-zero
    When I run lobster "admin config --executor-addr localhost:1"
    Then the exit code should not be 0

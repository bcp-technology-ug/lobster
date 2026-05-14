@daemon @covers:cli:runs
Feature: lobster runs command
  As a developer
  I want to list and inspect stored runs via the lobster runs subcommands
  So that I can review past test execution history

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  Scenario: lobster runs --help exits 0
    When I run lobster "runs --help"
    Then the exit code should be 0

  Scenario: lobster runs --help shows usage information
    When I run lobster "runs --help"
    Then the exit code should be 0
    And the output should contain "runs"

  Scenario: lobster runs list --help exits 0
    When I run lobster "runs list --help"
    Then the exit code should be 0

  Scenario: lobster runs list against daemon exits 0
    When I run lobster "runs list --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: lobster runs list --format json exits 0
    When I run lobster "runs list --format json --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: lobster runs list --format json output is valid JSON
    When I run lobster "runs list --format json --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: lobster runs list --workspace filter is accepted
    When I run lobster "runs list --workspace test-runs-ws --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: lobster runs list --limit 5 is accepted
    When I run lobster "runs list --limit 5 --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: lobster runs list with unreachable daemon addr exits non-zero
    When I run lobster "runs list --executor-addr localhost:1"
    Then the exit code should not be 0

  Scenario: lobster runs get --help exits 0
    When I run lobster "runs get --help"
    Then the exit code should be 0

  Scenario: lobster runs cancel --help exits 0
    When I run lobster "runs cancel --help"
    Then the exit code should be 0

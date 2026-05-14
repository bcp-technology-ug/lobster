@daemon @covers:cli:plans
Feature: lobster plans command
  As a developer
  I want to list and inspect stored plans via the lobster plans subcommands
  So that I can review past execution plans

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  Scenario: lobster plans --help exits 0
    When I run lobster "plans --help"
    Then the exit code should be 0

  Scenario: lobster plans --help shows usage
    When I run lobster "plans --help"
    Then the output should contain "plans"

  Scenario: lobster plans list --help exits 0
    When I run lobster "plans list --help"
    Then the exit code should be 0

  Scenario: lobster plans list against daemon exits 0
    When I run lobster "plans list --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: lobster plans list --format json exits 0
    When I run lobster "plans list --format json --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: lobster plans list --format json output is valid JSON
    When I run lobster "plans list --format json --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: lobster plans list --workspace filter is accepted
    When I run lobster "plans list --workspace test-plans-cmd-ws --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: lobster plans list --limit 10 is accepted
    When I run lobster "plans list --limit 10 --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: lobster plans list with unreachable daemon exits non-zero
    When I run lobster "plans list --executor-addr localhost:1"
    Then the exit code should not be 0

  Scenario: lobster plans get --help exits 0
    When I run lobster "plans get --help"
    Then the exit code should be 0

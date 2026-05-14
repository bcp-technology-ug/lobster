@daemon @covers:cli:integrations @covers:IntegrationService.ListIntegrationAdapters @covers:IntegrationService.GetIntegrationAdapter @covers:IntegrationService.SetIntegrationAdapterState @covers:IntegrationService.ValidateIntegrationAdapter
Feature: lobster integrations command
  As a developer
  I want to manage integration adapters via the lobster integrations subcommands
  So that adapter state can be controlled from the CLI

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  Scenario: lobster integrations --help exits 0
    When I run lobster "integrations --help"
    Then the exit code should be 0

  Scenario: lobster integrations --help shows subcommands
    When I run lobster "integrations --help"
    Then the output should contain "integrations"

  Scenario: lobster integrations list --help exits 0
    When I run lobster "integrations list --help"
    Then the exit code should be 0

  Scenario: lobster integrations list against daemon exits 0
    When I run lobster "integrations list --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: lobster integrations list returns JSON with adapters field
    When I run lobster "integrations list --format json --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: lobster integrations get --help exits 0
    When I run lobster "integrations get --help"
    Then the exit code should be 0

  Scenario: lobster integrations get with unknown adapter exits non-zero
    When I run lobster "integrations get nonexistent-adapter --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should not be 0

  Scenario: lobster integrations enable --help exits 0
    When I run lobster "integrations enable --help"
    Then the exit code should be 0

  Scenario: lobster integrations enable with unknown adapter exits non-zero
    When I run lobster "integrations enable nonexistent-adapter --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should not be 0

  Scenario: lobster integrations disable --help exits 0
    When I run lobster "integrations disable --help"
    Then the exit code should be 0

  Scenario: lobster integrations disable with unknown adapter exits non-zero
    When I run lobster "integrations disable nonexistent-adapter --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should not be 0

  Scenario: lobster integrations validate --help exits 0
    When I run lobster "integrations validate --help"
    Then the exit code should be 0

  Scenario: lobster integrations validate with unknown adapter exits non-zero
    When I run lobster "integrations validate nonexistent-adapter --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should not be 0

  Scenario: lobster integrations list with unreachable daemon exits non-zero
    When I run lobster "integrations list --executor-addr localhost:1"
    Then the exit code should not be 0

  Scenario: lobster integrations get with unreachable daemon exits non-zero
    When I run lobster "integrations get some-adapter --executor-addr localhost:1"
    Then the exit code should not be 0

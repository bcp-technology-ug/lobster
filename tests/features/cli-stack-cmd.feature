@daemon @covers:cli:stack
Feature: lobster stack command
  As a developer
  I want to manage Docker Compose stacks via the lobster stack subcommands
  So that infrastructure can be provisioned and inspected independently of runs

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  Scenario: lobster stack --help exits 0
    When I run lobster "stack --help"
    Then the exit code should be 0

  Scenario: lobster stack --help shows subcommands
    When I run lobster "stack --help"
    Then the output should contain "stack"

  Scenario: lobster stack status --help exits 0
    When I run lobster "stack status --help"
    Then the exit code should be 0

  Scenario: lobster stack status with workspace exits non-zero for missing stack
    When I run lobster "stack status --workspace test-stack-cmd --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should not be 0

  Scenario: lobster stack status with workspace error reaches daemon
    When I run lobster "stack status --workspace test-stack-cmd-json --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should not be 0

  Scenario: lobster stack ensure --help exits 0
    When I run lobster "stack up --help"
    Then the exit code should be 0

  Scenario: lobster stack teardown --help exits 0
    When I run lobster "stack down --help"
    Then the exit code should be 0

  Scenario: lobster stack with unreachable daemon exits non-zero
    When I run lobster "stack status --executor-addr localhost:1 --workspace test-ws"
    Then the exit code should not be 0

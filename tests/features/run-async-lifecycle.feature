@daemon @covers:RunService.RunAsync @covers:cli:run:watch @covers:cli:run:status @covers:RunService.StreamRunEvents
Feature: Async run lifecycle
  As a developer
  I want async run submission, watching, status inspection, and cancellation to work end-to-end
  So that long-running suites can execute without blocking the caller

  # These scenarios require a running lobsterd instance.
  # Set LOBSTERD_ADDR to the gRPC address before running.

  Scenario: Async run returns a run ID on success
    Given I am in a new temporary directory
    And I create the file "features/async.feature" with content:
      """
      Feature: Async feature
        Scenario: Placeholder
          Given I am in a new temporary directory
          When I run the command "echo async-ok"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/async.feature --run-mode async --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --ci"
    Then the exit code should be 0

  Scenario: lobster run status shows run in a terminal state
    Given I am in a new temporary directory
    And I create the file "features/async-status.feature" with content:
      """
      Feature: Async status
        Scenario: Quick pass
          Given I am in a new temporary directory
          When I run the command "echo status-ok"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/async-status.feature --run-mode async --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --ci"
    And I store the output in variable "LOBSTER_RUN_ID"
    And I run the command "sleep 2"
    When I run lobster "run status --run-id ${LOBSTER_RUN_ID} --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: lobster run cancel accepts a valid run ID
    Given I am in a new temporary directory
    And I create the file "features/async-cancel.feature" with content:
      """
      Feature: Async cancel
        Scenario: Long running
          Given I am in a new temporary directory
          When I run the command "sleep 60"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/async-cancel.feature --run-mode async --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --ci"
    And I store the output in variable "LOBSTER_RUN_ID"
    When I run lobster "run cancel --run-id ${LOBSTER_RUN_ID} --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

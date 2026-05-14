@daemon @covers:RunService.RunAsync @covers:cli:run:watch @covers:cli:run:status @covers:RunService.StreamRunEvents @covers:RunService.CancelRun
Feature: run async advanced lifecycle
  As a developer
  I want detailed control over async run lifecycle events
  So that CI systems can fully track and control long-running tests

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  Scenario: run async outputs run_id to stderr
    Given I generate a unique workspace id
    And I create the file "features/async1.feature" with content:
      """
      Feature: Async 1
        Scenario: Async pass
          When I run the command "echo async1"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/async1.feature --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --run-mode async"
    Then the exit code should be 0
    And the output should not be empty

  Scenario: run watch streams until run reaches terminal state
    Given I generate a unique workspace id
    And I create the file "features/async-watch.feature" with content:
      """
      Feature: Async Watch
        Scenario: Watch pass
          When I run the command "echo watch"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/async-watch.feature --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --run-mode async"
    Then the exit code should be 0
    And I store the output in variable "WATCH_RUN_ID"
    When I run lobster "run watch --run-id ${WATCH_RUN_ID} --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: run status shows current state of run
    Given I generate a unique workspace id
    And I create the file "features/async-status.feature" with content:
      """
      Feature: Async Status
        Scenario: Status pass
          When I run the command "echo status"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/async-status.feature --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --run-mode async"
    Then the exit code should be 0
    And I store the output in variable "STATUS_RUN_ID"
    When I run lobster "run status --run-id ${STATUS_RUN_ID} --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0
    And the output should contain "Run ID"

  Scenario: run status shows run id in output
    Given I generate a unique workspace id
    And I create the file "features/async-status-json.feature" with content:
      """
      Feature: Async Status JSON
        Scenario: Status JSON pass
          When I run the command "echo status-json"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/async-status-json.feature --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --run-mode async"
    Then the exit code should be 0
    And I store the output in variable "STATUS_RUN_ID2"
    When I run lobster "run status --run-id ${STATUS_RUN_ID2} --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0
    And the output should contain "Run ID"

  Scenario: run cancel on in-flight run succeeds
    Given I generate a unique workspace id
    And I create the file "features/cancellable.feature" with content:
      """
      Feature: Cancellable
        Scenario: Long running
          When I run the command "sleep 30"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/cancellable.feature --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --run-mode async"
    Then the exit code should be 0
    And I store the output in variable "CANCEL_RUN_ID"
    When I run lobster "run cancel --run-id ${CANCEL_RUN_ID} --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: run cancel with valid run id exits 0
    Given I generate a unique workspace id
    And I create the file "features/cancel-reason.feature" with content:
      """
      Feature: Cancel reason
        Scenario: Long running 2
          When I run the command "sleep 30"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/cancel-reason.feature --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --run-mode async"
    Then the exit code should be 0
    And I store the output in variable "CANCEL_REASON_RUN_ID"
    When I run lobster "run cancel --run-id ${CANCEL_REASON_RUN_ID} --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: run cancel with unknown run_id exits non-zero
    When I run lobster "run cancel --run-id nonexistent-run-xyz --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should not be 0

  Scenario: run watch with unknown run_id exits non-zero
    When I run lobster "run watch --run-id nonexistent-run-xyz --executor-mode daemon --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should not be 0

  Scenario: run status with unknown run_id exits non-zero
    When I run lobster "run status --run-id nonexistent-run-xyz --executor-mode daemon --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should not be 0

  Scenario: run cancel with unreachable daemon exits non-zero
    When I run lobster "run cancel --run-id some-run --executor-mode daemon --executor-addr localhost:1"
    Then the exit code should not be 0

  Scenario: run watch with unreachable daemon exits non-zero
    When I run lobster "run watch --run-id some-run --executor-mode daemon --executor-addr localhost:1"
    Then the exit code should not be 0

@daemon
Feature: lobsterd — async run lifecycle
  As a developer
  I want to submit async runs to the daemon and use run subcommands to manage them
  So that long-running suites can execute without blocking the caller

  # Requires a running lobsterd. Set LOBSTERD_ADDR (gRPC) and LOBSTERD_HTTP_URL (HTTP).
  # These are injected by `make test-all` via --env flags.

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  # ── Submit ────────────────────────────────────────────────────────────────

  Scenario: Async submission returns a non-empty run_id
    Given I create the file "features/async.feature" with content:
      """
      Feature: Async lifecycle
        Scenario: Pass
          Given I am in a new temporary directory
          When I run the command "echo async-ok"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/async.feature --run-mode async --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --ci"
    Then the exit code should be 0
    And I store the output in variable "ASYNC_RUN_ID"
    And the stderr should contain "Run submitted"

  # ── Status ────────────────────────────────────────────────────────────────

  Scenario: run status shows state for a submitted run
    Given I create the file "features/status-check.feature" with content:
      """
      Feature: Status check
        Scenario: Pass
          Given I am in a new temporary directory
          When I run the command "echo status-ok"
          Then the exit code should be 0
      """
    And I run lobster "run --features features/status-check.feature --run-mode async --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --ci"
    And I store the output in variable "ASYNC_RUN_ID"
    And I run the command "sleep 3"
    When I run lobster "run status --run-id ${ASYNC_RUN_ID} --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  # ── Watch ─────────────────────────────────────────────────────────────────

  Scenario: run watch streams events for a completed run
    Given I create the file "features/watch.feature" with content:
      """
      Feature: Watch test
        Scenario: Watch pass
          Given I am in a new temporary directory
          When I run the command "echo watch-ok"
          Then the exit code should be 0
      """
    And I run lobster "run --features features/watch.feature --run-mode async --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --ci"
    And I store the output in variable "ASYNC_RUN_ID"
    And I run the command "sleep 3"
    When I run lobster "run watch --run-id ${ASYNC_RUN_ID} --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  # ── Cancel ────────────────────────────────────────────────────────────────

  Scenario: run cancel accepts a valid run_id and exits 0
    Given I create the file "features/cancel.feature" with content:
      """
      Feature: Cancel test
        Scenario: Long running
          Given I am in a new temporary directory
          When I run the command "sleep 60"
          Then the exit code should be 0
      """
    And I run lobster "run --features features/cancel.feature --run-mode async --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --ci"
    And I store the output in variable "ASYNC_RUN_ID"
    When I run lobster "run cancel --run-id ${ASYNC_RUN_ID} --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0

  Scenario: run cancel with unknown run_id exits non-zero
    When I run lobster "run cancel --run-id 00000000-0000-0000-0000-000000000000 --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should not be 0

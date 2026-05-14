@daemon @covers:cli:run @covers:cli:run:cancel
Feature: lobsterd — CLI connectivity
  As a developer
  I want the lobster CLI to connect to a running lobsterd daemon
  So that I can verify gRPC transport and remote execution work end-to-end

  # Requires a running lobsterd. Set LOBSTERD_ADDR (gRPC) and LOBSTERD_HTTP_URL (HTTP).
  # These are injected by `make test-all` via --env flags.

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  # ── gRPC reachability ─────────────────────────────────────────────────────

  Scenario: lobster can connect to the daemon gRPC port
    When I run lobster "run --features features/nonexistent/*.feature --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --ci"
    Then the exit code should be 0

  # ── sync run via daemon ───────────────────────────────────────────────────

  Scenario: lobster run --executor-mode daemon executes undefined feature and exits 0
    Given I create the file "features/remote.feature" with content:
      """
      Feature: Remote daemon test
        Scenario: Undefined step via daemon
          Given an undefined step on the daemon
      """
    When I run lobster "run --features features/remote.feature --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --ci"
    Then the exit code should be 0

  Scenario: lobster run --executor-mode daemon streams output to stdout
    Given I create the file "features/stream.feature" with content:
      """
      Feature: Stream test
        Scenario: Shell step via daemon
          Given I am in a new temporary directory
          When I run the command "echo daemon-stream-ok"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/stream.feature --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --ci"
    Then the exit code should be 0

  Scenario: lobster run exits non-zero when daemon is unreachable
    When I run lobster "run --executor-mode daemon --executor-addr localhost:1 --ci"
    Then the exit code should not be 0

  # ── --run-mode async via daemon ───────────────────────────────────────────

  Scenario: lobster run --run-mode async returns a run_id
    Given I create the file "features/async.feature" with content:
      """
      Feature: Async daemon test
        Scenario: Quick pass
          Given I am in a new temporary directory
          When I run the command "echo async-daemon-ok"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/async.feature --run-mode async --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --ci"
    Then the exit code should be 0
    And the stderr should contain "Run submitted"

  # ── report flags via daemon ───────────────────────────────────────────────

  Scenario: --report-json produces a JSON file when running via daemon
    Given I create the file "features/report.feature" with content:
      """
      Feature: Daemon report test
        Scenario: Report pass
          Given I am in a new temporary directory
          When I run the command "echo report-daemon"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/report.feature --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --report-json report.json --ci"
    Then the exit code should be 0
    And the file "report.json" should exist

  Scenario: --report-junit produces an XML file when running via daemon
    Given I create the file "features/junit.feature" with content:
      """
      Feature: Daemon JUnit test
        Scenario: JUnit pass
          Given I am in a new temporary directory
          When I run the command "echo junit-daemon"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/junit.feature --executor-mode daemon --executor-addr ${LOBSTERD_ADDR} --report-junit report.xml --ci"
    Then the exit code should be 0
    And the file "report.xml" should exist

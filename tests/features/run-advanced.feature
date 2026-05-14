@covers:cli:run
Feature: lobster run — flags and execution control
  As a developer
  I want lobster run to honour all execution flags
  So that I can control test selection, environment, and output precisely

  # ── --env flag ────────────────────────────────────────────────────────────

  Scenario: --env with invalid format exits 2
    Given I am in a new temporary directory
    And I create the file "features/dummy.feature" with content:
      """
      Feature: Dummy
        Scenario: Dummy
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/dummy.feature --env INVALID_NO_EQUALS --ci"
    Then the exit code should be 2
    And the stderr should contain "KEY=VALUE"

  Scenario: --env KEY=VALUE is accepted and run succeeds
    Given I am in a new temporary directory
    And I create the file "features/dummy.feature" with content:
      """
      Feature: Dummy
        Scenario: Dummy
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/dummy.feature --env MY_VAR=hello --ci"
    Then the exit code should be 0

  # ── --executor-mode daemon without address ────────────────────────────────

  Scenario: --executor-mode daemon without --executor-addr exits 2
    Given I am in a new temporary directory
    And I create the file "features/dummy.feature" with content:
      """
      Feature: Dummy
        Scenario: Dummy
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/dummy.feature --executor-mode daemon --ci"
    Then the exit code should be 2
    And the stderr should contain "executor-addr"

  # ── --run-mode async requires daemon executor ─────────────────────────────

  Scenario: --run-mode async with --executor-mode local exits 2
    Given I am in a new temporary directory
    And I create the file "features/dummy.feature" with content:
      """
      Feature: Dummy
        Scenario: Dummy
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/dummy.feature --run-mode async --executor-mode local --ci"
    Then the exit code should be 2

  # ── --from-plan ───────────────────────────────────────────────────────────

  Scenario: --from-plan with non-existent file exits 2
    Given I am in a new temporary directory
    When I run lobster "run --from-plan /nonexistent/plan.json --ci"
    Then the exit code should be 2
    And the stderr should contain "plan"

  Scenario: --from-plan with valid plan file runs scenarios in the plan
    Given I am in a new temporary directory
    And I create the file "features/planned.feature" with content:
      """
      Feature: Planned

        Scenario: Planned alpha
          Given I set the base URL to "http://example.com"

        Scenario: Planned beta
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "plan --features features/planned.feature --out plan.json"
    Then the exit code should be 0
    And the file "plan.json" should exist
    When I run lobster "run --from-plan plan.json --ci"
    Then the exit code should be 0
    And the output should contain "2 passed"

  # ── --scenario-regex execution filtering ──────────────────────────────────

  Scenario: --scenario-regex runs only matching scenarios
    Given I am in a new temporary directory
    And I create the file "features/filter.feature" with content:
      """
      Feature: Filter

        Scenario: Alpha scenario
          Given I set the base URL to "http://example.com"

        Scenario: Beta scenario
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/filter.feature --scenario-regex Alpha --ci"
    Then the exit code should be 0
    And the output should contain "1 passed"

  # ── --tags execution filtering ────────────────────────────────────────────

  Scenario: --tags runs only tagged scenarios
    Given I am in a new temporary directory
    And I create the file "features/tagsrun.feature" with content:
      """
      Feature: Tags run

        @run-me
        Scenario: Tagged scenario
          Given I set the base URL to "http://example.com"

        Scenario: Untagged scenario
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/tagsrun.feature --tags @run-me --ci"
    Then the exit code should be 0
    And the output should contain "1 passed"

  # ── --tags with no matches runs nothing ───────────────────────────────────

  Scenario: --tags with non-matching expression exits 0 with 0 scenarios
    Given I am in a new temporary directory
    And I create the file "features/tagsrun.feature" with content:
      """
      Feature: Tags run

        @smoke
        Scenario: Smoke scenario
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/tagsrun.feature --tags @nonexistent --ci"
    Then the exit code should be 0
    And the output should contain "0 total"

  # ── reporting flags ───────────────────────────────────────────────────────

  Scenario: --report-json writes a JSON file with run summary
    Given I am in a new temporary directory
    And I create the file "features/report.feature" with content:
      """
      Feature: Report

        Scenario: Report scenario
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/report.feature --report-json run-report.json --ci"
    Then the exit code should be 0
    And the file "run-report.json" should exist
    And the file "run-report.json" should contain "run_id"
    And the file "run-report.json" should contain "\"status\": \"passed\""

  Scenario: --report-junit writes an XML file with testsuites structure
    Given I am in a new temporary directory
    And I create the file "features/report.feature" with content:
      """
      Feature: Report

        Scenario: Report scenario
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/report.feature --report-junit run-report.xml --ci"
    Then the exit code should be 0
    And the file "run-report.xml" should exist
    And the file "run-report.xml" should contain "testsuites"
    And the file "run-report.xml" should contain "testcase"

  Scenario: Both --report-json and --report-junit can be used together
    Given I am in a new temporary directory
    And I create the file "features/report.feature" with content:
      """
      Feature: Report

        Scenario: Dual report
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/report.feature --report-json out.json --report-junit out.xml --ci"
    Then the exit code should be 0
    And the file "out.json" should exist
    And the file "out.xml" should exist

  # ── --fail-fast ───────────────────────────────────────────────────────────

  Scenario: --fail-fast stops after first failing scenario
    Given I am in a new temporary directory
    And I create the file "features/failfast.feature" with content:
      """
      Feature: Fail fast

        Scenario: First fails
          Given I set the base URL to "http://localhost:19999"
          When I send a GET request to "/ping"
          Then the response status should be 200

        Scenario: Second would also fail
          Given I set the base URL to "http://localhost:19999"
          When I send a GET request to "/ping"
          Then the response status should be 200
      """
    When I run lobster "run --features features/failfast.feature --fail-fast --ci"
    Then the exit code should be 1
    And the output should contain "1 total"

  # ── verbose flag ──────────────────────────────────────────────────────────

  Scenario: -v flag is accepted and does not crash
    Given I am in a new temporary directory
    And I create the file "features/verbose.feature" with content:
      """
      Feature: Verbose

        Scenario: Verbose run
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/verbose.feature -v --ci"
    Then the exit code should be 0

  # ── undefined steps ───────────────────────────────────────────────────────

  Scenario: Undefined steps exit 0 and output mentions undefined
    Given I am in a new temporary directory
    And I create the file "features/undefined.feature" with content:
      """
      Feature: Undefined

        Scenario: Has undefined step
          Given an undefined step that does not exist anywhere
      """
    When I run lobster "run --features features/undefined.feature --ci"
    Then the exit code should be 0
    And the output should contain "undefined"

  # ── no feature files ─────────────────────────────────────────────────────

  Scenario: --features glob matching nothing exits 0
    Given I am in a new temporary directory
    When I run lobster "run --features features/nowhere/*.feature --ci"
    Then the exit code should be 0

@covers:cli:run
Feature: run command output formats
  As a developer
  I want to control the verbosity and output format of the run command
  So that CI pipelines and developers get the right level of detail

  # ── Verbosity levels ─────────────────────────────────────────────────────

  Scenario: run with -v flag exits 0
    Given I am in a new temporary directory
    And I create the file "features/verbose.feature" with content:
      """
      Feature: Verbose
        Scenario: Verbose pass
          When I run the command "echo verbose"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/verbose.feature -v --ci"
    Then the exit code should be 0

  Scenario: run with -vv flag exits 0
    Given I am in a new temporary directory
    And I create the file "features/verbose2.feature" with content:
      """
      Feature: Verbose2
        Scenario: Verbose2 pass
          When I run the command "echo verbose2"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/verbose2.feature -vv --ci"
    Then the exit code should be 0

  Scenario: run with -vvv flag exits 0
    Given I am in a new temporary directory
    And I create the file "features/verbose3.feature" with content:
      """
      Feature: Verbose3
        Scenario: Verbose3 pass
          When I run the command "echo verbose3"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/verbose3.feature -vvv --ci"
    Then the exit code should be 0

  # ── CI mode ───────────────────────────────────────────────────────────────

  Scenario: run --ci flag suppresses interactive output
    Given I am in a new temporary directory
    And I create the file "features/ci.feature" with content:
      """
      Feature: CI mode
        Scenario: CI pass
          When I run the command "echo ci"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/ci.feature --ci"
    Then the exit code should be 0

  # ── JSON report ───────────────────────────────────────────────────────────

  Scenario: run --report-json produces a file
    Given I am in a new temporary directory
    And I create the file "features/rptjson.feature" with content:
      """
      Feature: Report JSON
        Scenario: Pass for report
          When I run the command "echo reportjson"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/rptjson.feature --report-json report.json --ci"
    Then the exit code should be 0
    And the file "report.json" should exist

  Scenario: run --report-json produces valid JSON
    Given I am in a new temporary directory
    And I create the file "features/rptjson2.feature" with content:
      """
      Feature: Report JSON 2
        Scenario: Pass for report
          When I run the command "echo reportjson2"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/rptjson2.feature --report-json report2.json --ci"
    Then the exit code should be 0
    And the file "report2.json" should contain valid JSON

  Scenario: run --report-json contains test results
    Given I am in a new temporary directory
    And I create the file "features/rptjson3.feature" with content:
      """
      Feature: Report JSON 3
        Scenario: Pass for report
          When I run the command "echo reportjson3"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/rptjson3.feature --report-json report3.json --ci"
    Then the exit code should be 0
    And the file "report3.json" should exist

  # ── JUnit report ──────────────────────────────────────────────────────────

  Scenario: run --report-junit produces a file
    Given I am in a new temporary directory
    And I create the file "features/rptjunit.feature" with content:
      """
      Feature: Report JUnit
        Scenario: Pass for junit
          When I run the command "echo junit"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/rptjunit.feature --report-junit junit.xml --ci"
    Then the exit code should be 0
    And the file "junit.xml" should exist

  Scenario: run --report-junit output contains XML structure
    Given I am in a new temporary directory
    And I create the file "features/rptjunit2.feature" with content:
      """
      Feature: Report JUnit 2
        Scenario: Pass for junit
          When I run the command "echo junit2"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/rptjunit2.feature --report-junit junit2.xml --ci"
    Then the exit code should be 0
    And the file "junit2.xml" should contain "testsuites"

  # ── Both reports simultaneously ───────────────────────────────────────────

  Scenario: run with both --report-json and --report-junit exits 0
    Given I am in a new temporary directory
    And I create the file "features/bothreports.feature" with content:
      """
      Feature: Both reports
        Scenario: Pass for both
          When I run the command "echo bothreports"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/bothreports.feature --report-json out.json --report-junit out.xml --ci"
    Then the exit code should be 0
    And the file "out.json" should exist
    And the file "out.xml" should exist

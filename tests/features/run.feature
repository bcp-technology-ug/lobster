@covers:cli:run
Feature: lobster run command
  As a developer
  I want lobster run to execute scenarios and report results correctly
  So that failures are surfaced clearly and output artifacts are produced

  Scenario: Feature with undefined steps exits 0
    Given I am in a new temporary directory
    And I create the file "features/undefined.feature" with content:
      """
      Feature: Undefined steps
        Scenario: Uses unknown step
          Given an unregistered step that does not exist
          When another unknown step happens
          Then a third unknown step is checked
      """
    When I run lobster "run --features features/undefined.feature --ci"
    Then the exit code should be 0

  Scenario: Feature glob matching no files exits 0
    Given I am in a new temporary directory
    When I run lobster "run --features features/nonexistent/*.feature --ci"
    Then the exit code should be 0

  Scenario: --report-json creates a JSON report file
    Given I am in a new temporary directory
    And I create the file "features/report.feature" with content:
      """
      Feature: Report test
        Scenario: Undefined for report
          Given a step that is not registered anywhere
      """
    When I run lobster "run --features features/report.feature --report-json report.json --ci"
    Then the file "report.json" should exist

  Scenario: --report-junit creates a JUnit XML report file
    Given I am in a new temporary directory
    And I create the file "features/junit.feature" with content:
      """
      Feature: JUnit test
        Scenario: Undefined for junit
          Given a step that is not registered anywhere
      """
    When I run lobster "run --features features/junit.feature --report-junit report.xml --ci"
    Then the file "report.xml" should exist

  Scenario: --run-mode async without daemon exits 2
    Given I am in a new temporary directory
    When I run lobster "run --features features/x.feature --run-mode async --executor-mode local --ci"
    Then the exit code should be 2

  Scenario: --fail-fast stops after first failing scenario
    Given I am in a new temporary directory
    And I create the file "features/failfast.feature" with content:
      """
      Feature: Fail fast
        Scenario: First failure
          Given I set the base URL to "http://localhost:19999"
          When I send a GET request to "/health"
          Then the response status should be 200

        Scenario: Second failure
          Given I set the base URL to "http://localhost:19999"
          When I send a GET request to "/health"
          Then the response status should be 200
      """
    When I run lobster "run --features features/failfast.feature --fail-fast --ci"
    Then the exit code should be 1

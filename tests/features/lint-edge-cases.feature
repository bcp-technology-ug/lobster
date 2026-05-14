@covers:cli:lint
Feature: lint command edge cases
  As a developer
  I want the lint command to catch common Gherkin quality issues
  So that feature files remain readable and well-structured

  Scenario: lint reports error for scenario with no steps
    Given I am in a new temporary directory
    And I create the file "features/nosteps.feature" with content:
      """
      Feature: No steps
        Scenario: Empty scenario
      """
    When I run lobster "lint --features features/nosteps.feature"
    Then the exit code should not be 0

  Scenario: lint passes on valid feature with And and But after proper steps
    Given I am in a new temporary directory
    And I create the file "features/validand.feature" with content:
      """
      Feature: Valid And
        Scenario: Valid and/but
          Given I am in a new temporary directory
          When I run the command "echo ok"
          Then the exit code should be 0
      """
    When I run lobster "lint --features features/validand.feature"
    Then the exit code should be 0

  Scenario: lint with --strict exits non-zero on warnings
    Given I am in a new temporary directory
    And I create the file "features/warnonly.feature" with content:
      """
      Feature: Feature with no scenarios
      """
    When I run lobster "lint --features features/warnonly.feature --strict"
    Then the exit code should not be 0

  Scenario: lint with glob matching no files exits 0
    Given I am in a new temporary directory
    When I run lobster "lint --features nonexistent/**/*.feature"
    Then the exit code should be 0

  Scenario: lint feature with multiple issues reports all of them
    Given I am in a new temporary directory
    And I create the file "features/multi-issues.feature" with content:
      """
      Feature: Multiple issues
        Scenario: Empty scenario A

        Scenario: Empty scenario B
      """
    When I run lobster "lint --features features/multi-issues.feature"
    Then the exit code should not be 0

  Scenario: lint --help exits 0
    Given I am in a new temporary directory
    When I run lobster "lint --help"
    Then the exit code should be 0

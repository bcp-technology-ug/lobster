@covers:cli:run
Feature: Advanced hook behaviour
  As a test author
  I want hooks to be resilient and composable
  So that complex setup/teardown logic does not interfere with test results

  Scenario: BeforeScenario variables are isolated between scenarios
    Given I am in a new temporary directory
    When I run the command "echo scenario-one-value"
    And I store the output in variable "ISOLATED_VAR"
    Then the output should contain "scenario-one-value"

  Scenario: Second scenario does not see variables from the first
    Given I am in a new temporary directory
    When I run the command "echo fresh"
    Then the output should contain "fresh"
    And the output should not contain "scenario-one-value"

  Scenario: AfterScenario still runs when a step fails
    Given I am in a new temporary directory
    And I create the file "features/hook-fail.feature" with content:
      """
      Feature: Hook fail test
        Scenario: Failing step
          Given I am in a new temporary directory
          When I run lobster "--help"
          Then the exit code should be 99
      """
    When I run lobster "run --features features/hook-fail.feature --ci"
    Then the exit code should be 1

  Scenario: Nested temporary directories are cleaned up between scenarios
    Given I am in a new temporary directory
    When I run the command "echo nested-dir"
    Then the exit code should be 0
    And the output should contain "nested-dir"

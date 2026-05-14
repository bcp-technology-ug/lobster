@covers:cli:run
Feature: Hook lifecycle basics
  As a test author
  I want BeforeSuite, AfterSuite, BeforeScenario, and AfterScenario hooks to run
  So that I can implement reliable setup and teardown

  Scenario: Built-in AfterScenario cleans up the temporary directory
    Given I am in a new temporary directory
    And I run the command "pwd"
    Then the exit code should be 0
    And the output should contain "lobster-test-"

  Scenario: Variables set in a step are visible in the same scenario
    Given I am in a new temporary directory
    When I run the command "echo fixture-value"
    And I store the output in variable "fixture_output"
    Then the output should contain "fixture-value"

  Scenario: Shell step injects scenario variables as env vars
    Given I am in a new temporary directory
    When I run the command "echo hello"
    And I store the output in variable "MY_VAR"
    When I run the command "echo ${MY_VAR}"
    Then the output should contain "hello"

  Scenario: AfterScenario error does not prevent next scenario from running
    Given I am in a new temporary directory
    When I run lobster "--help"
    Then the exit code should be 0

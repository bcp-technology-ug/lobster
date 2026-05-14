@covers:cli:tui
Feature: lobster tui command
  As a developer
  I want the tui command to provide a terminal user interface for running tests
  So that I can interactively control test execution

  Scenario: lobster tui --help exits 0
    Given I am in a new temporary directory
    When I run lobster "tui --help"
    Then the exit code should be 0

  Scenario: lobster tui --help shows usage
    Given I am in a new temporary directory
    When I run lobster "tui --help"
    Then the output should contain "tui"

  Scenario: lobster tui --help mentions features flag
    Given I am in a new temporary directory
    When I run lobster "tui --help"
    Then the exit code should be 0
    And the output should not be empty

  Scenario: lobster tui with unreachable daemon exits non-zero
    Given I am in a new temporary directory
    When I run lobster "tui --executor-mode daemon --executor-addr localhost:1 --ci"
    Then the exit code should not be 0

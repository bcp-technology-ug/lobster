@covers:cli:doctor
Feature: lobster doctor command
  As a developer
  I want the doctor command to diagnose configuration and connectivity issues
  So that I can quickly identify what is wrong with my lobster setup

  Scenario: lobster doctor --help exits 0
    Given I am in a new temporary directory
    When I run lobster "doctor --help"
    Then the exit code should be 0

  Scenario: lobster doctor --help shows usage
    Given I am in a new temporary directory
    When I run lobster "doctor --help"
    Then the output should contain "doctor"

  Scenario: lobster doctor exits 0 with valid default config
    Given I am in a new temporary directory
    When I run lobster "doctor"
    Then the exit code should be 0

  Scenario: lobster doctor outputs check summary
    Given I am in a new temporary directory
    When I run lobster "doctor"
    Then the exit code should be 0
    And the output should not be empty

  Scenario: lobster doctor --format json exits 0
    Given I am in a new temporary directory
    When I run lobster "doctor --format json"
    Then the exit code should be 0

  Scenario: lobster doctor --format json output is valid JSON
    Given I am in a new temporary directory
    When I run lobster "doctor --format json"
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: lobster doctor with unreachable daemon reports issue and exits non-zero
    Given I am in a new temporary directory
    When I run lobster "doctor --executor-mode daemon --executor-addr localhost:1"
    Then the exit code should not be 0

  Scenario: lobster doctor in a directory with missing features dir reports it
    Given I am in a new temporary directory
    When I run lobster "doctor"
    Then the exit code should be 0
    And the output should not be empty

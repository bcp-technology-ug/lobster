@covers:cli:config
Feature: config command edge cases
  As a developer
  I want the config command to validate, print, and layer configuration correctly
  So that misconfigurations are caught early

  Scenario: config --help exits 0
    Given I am in a new temporary directory
    When I run lobster "config --help"
    Then the exit code should be 0

  Scenario: config --validate with valid lobster.yaml exits 0
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: test-config-valid
      features: features/
      """
    When I run lobster "config --validate"
    Then the exit code should be 0

  Scenario: config --validate with unknown config key still exits 0
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      not_a_valid_key: true
      """
    When I run lobster "config --validate"
    Then the exit code should be 0

  Scenario: config --print outputs configuration
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: printconfig
      features: features/
      """
    When I run lobster "config --print"
    Then the exit code should be 0
    And the output should not be empty

  Scenario: config --format json with valid config is valid JSON
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: jsonconfig
      features: features/
      """
    When I run lobster "config --print --format json"
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: config with missing lobster.yaml uses defaults and exits 0
    Given I am in a new temporary directory
    When I run lobster "config --print"
    Then the exit code should be 0

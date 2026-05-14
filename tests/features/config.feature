@covers:cli:config
Feature: lobster config command
  As a developer
  I want lobster config to display the effective configuration
  So that I can verify what settings will be used at runtime

  Scenario: Config with lobster.yaml in CWD shows project context
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" containing "project: config-test-project"
    When I run lobster "config"
    Then the exit code should be 0
    And the output should contain "Workspace"

  Scenario: --format json produces valid JSON
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" containing "project: json-config-project"
    When I run lobster "config --format json"
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: Config with no lobster.yaml uses defaults and exits 0
    Given I am in a new temporary directory
    When I run lobster "config"
    Then the exit code should be 0

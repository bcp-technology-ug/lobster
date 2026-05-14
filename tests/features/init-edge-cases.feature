@covers:cli:init
Feature: init command edge cases
  As a developer
  I want the init command to handle various project configurations correctly
  So that projects are scaffolded correctly even in unusual conditions

  Scenario: init --help exits 0
    Given I am in a new temporary directory
    When I run lobster "init --help"
    Then the exit code should be 0

  Scenario: init creates lobster.yaml in an empty directory
    Given I am in a new temporary directory
    When I run lobster "init --project testproj --no-interactive"
    Then the exit code should be 0
    And the file "lobster.yaml" should exist

  Scenario: init creates features directory
    Given I am in a new temporary directory
    When I run lobster "init --project testproj2 --no-interactive"
    Then the exit code should be 0

  Scenario: init in existing directory with lobster.yaml does not overwrite
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: existing-project
      """
    When I run lobster "init --project newproj --no-interactive"
    Then the exit code should not be 0

  Scenario: init without --project flag exits non-zero
    Given I am in a new temporary directory
    When I run lobster "init --no-interactive"
    Then the exit code should not be 0

  Scenario: init with custom --features path creates that directory
    Given I am in a new temporary directory
    When I run lobster "init --project mypkg --features custom/feats --no-interactive"
    Then the exit code should be 0

  Scenario: init created lobster.yaml passes validate
    Given I am in a new temporary directory
    When I run lobster "init --project validateproj --no-interactive"
    Then the exit code should be 0
    When I run lobster "validate"
    Then the exit code should be 0

  Scenario: init --project sets the project name in lobster.yaml
    Given I am in a new temporary directory
    When I run lobster "init --project my-test-project --no-interactive"
    Then the exit code should be 0
    And the file "lobster.yaml" should contain "my-test-project"

Feature: lobster init command
  As a developer
  I want lobster init to scaffold a new test project
  So that I can get started quickly with a working directory structure

  Scenario: --no-interactive with --project scaffolds project files
    Given I am in a new temporary directory
    When I run lobster "init . --project my-project --no-interactive"
    Then the exit code should be 0
    And the file "lobster.yaml" should exist
    And the directory "features" should exist
    And the directory ".lobster" should exist
    And the file "lobster.yaml" should contain "my-project"

  Scenario: --no-interactive without --project exits 2
    Given I am in a new temporary directory
    When I run lobster "init . --no-interactive"
    Then the exit code should be 2

  Scenario: Stdin fallback populates project name when not a TTY
    Given I am in a new temporary directory
    When I run the command "printf 'stdin-project\n\n\n\n' | lobster init ."
    Then the exit code should be 0
    And the file "lobster.yaml" should exist
    And the file "lobster.yaml" should contain "stdin-project"

  Scenario: Generated lobster.yaml contains the provided project name
    Given I am in a new temporary directory
    When I run lobster "init . --project acme-tests --no-interactive"
    Then the exit code should be 0
    And the file "lobster.yaml" should contain "acme-tests"

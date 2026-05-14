@covers:cli:config
Feature: lobster config — show and validate configuration
  As a developer
  I want lobster config to display the active configuration clearly
  So that I can diagnose issues and understand the runtime environment

  # ── default output ────────────────────────────────────────────────────────

  Scenario: config command exits 0 with lobster.yaml present
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" containing "project: config-test"
    When I run lobster "config"
    Then the exit code should be 0

  Scenario: config command exits 0 even without lobster.yaml
    Given I am in a new temporary directory
    When I run lobster "config"
    Then the exit code should be 0

  Scenario: Default config output contains Workspace section
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" containing "project: myproj"
    When I run lobster "config"
    Then the exit code should be 0
    And the output should contain "Workspace"

  Scenario: Default config output contains Persistence section
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" containing "project: myproj"
    When I run lobster "config"
    Then the exit code should be 0
    And the output should contain "Persistence"

  # ── --format json ─────────────────────────────────────────────────────────

  Scenario: --format json outputs valid JSON
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" containing "project: json-config"
    When I run lobster "config --format json"
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: JSON config includes sqlite_path key
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" containing "project: json-config"
    When I run lobster "config --format json"
    Then the exit code should be 0
    And the output should be valid JSON
    And the output should contain "sqlite_path"

  Scenario: JSON config includes migration_mode key
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" containing "project: json-config"
    When I run lobster "config --format json"
    Then the exit code should be 0
    And the output should be valid JSON
    And the output should contain "migration_mode"

  # ── --sqlite-path flag ────────────────────────────────────────────────────

  Scenario: --sqlite-path overrides the database path in JSON output
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" containing "project: custom-db"
    When I run lobster "config --sqlite-path /tmp/custom-test.db --format json"
    Then the exit code should be 0
    And the output should be valid JSON
    And the output should contain "/tmp/custom-test.db"

  # ── --workspace flag ──────────────────────────────────────────────────────

  Scenario: --workspace includes workspace name in the sqlite path
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" containing "project: ws-test"
    When I run lobster "config --workspace myspace"
    Then the exit code should be 0
    And the output should contain "myspace"

  # ── --migration-mode invalid ──────────────────────────────────────────────

  Scenario: --migration-mode with unsupported value exits with error
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" containing "project: invalid-mode"
    When I run lobster "config --migration-mode bogusmode"
    Then the exit code should not be 0
    And the stderr should contain "migration"

  # ── --print flag ─────────────────────────────────────────────────────────

  Scenario: --print produces the same output as the default invocation
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" containing "project: print-test"
    When I run lobster "config --print"
    Then the exit code should be 0
    And the output should contain "Workspace"

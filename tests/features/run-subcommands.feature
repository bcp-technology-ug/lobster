Feature: lobster run subcommands — status, watch, cancel
  As a developer
  I want run subcommands to fail helpfully when required flags are missing
  So that errors are diagnosed immediately rather than silently

  # ── run status ────────────────────────────────────────────────────────────

  Scenario: run status without --run-id exits 2
    Given I am in a new temporary directory
    When I run lobster "run status"
    Then the exit code should be 2
    And the stderr should contain "run-id"

  Scenario: run status with --run-id but without daemon addr exits 3
    Given I am in a new temporary directory
    When I run lobster "run status --run-id abc123-def456"
    Then the exit code should be 3
    And the stderr should contain "executor-addr"

  # ── run watch ────────────────────────────────────────────────────────────

  Scenario: run watch without --run-id exits 2
    Given I am in a new temporary directory
    When I run lobster "run watch"
    Then the exit code should be 2
    And the stderr should contain "run-id"

  Scenario: run watch with --run-id but without daemon addr exits 3
    Given I am in a new temporary directory
    When I run lobster "run watch --run-id abc123-def456"
    Then the exit code should be 3
    And the stderr should contain "executor-addr"

  # ── run cancel ───────────────────────────────────────────────────────────

  Scenario: run cancel without --run-id exits 2
    Given I am in a new temporary directory
    When I run lobster "run cancel"
    Then the exit code should be 2
    And the stderr should contain "run-id"

  Scenario: run cancel with --run-id but without daemon addr exits 3
    Given I am in a new temporary directory
    When I run lobster "run cancel --run-id abc123-def456"
    Then the exit code should be 3
    And the stderr should contain "executor-addr"

  # ── help output ──────────────────────────────────────────────────────────

  Scenario: run status --help exits 0
    Given I am in a new temporary directory
    When I run lobster "run status --help"
    Then the exit code should be 0
    And the output should contain "run-id"

  Scenario: run watch --help exits 0
    Given I am in a new temporary directory
    When I run lobster "run watch --help"
    Then the exit code should be 0
    And the output should contain "run-id"

  Scenario: run cancel --help exits 0
    Given I am in a new temporary directory
    When I run lobster "run cancel --help"
    Then the exit code should be 0
    And the output should contain "run-id"

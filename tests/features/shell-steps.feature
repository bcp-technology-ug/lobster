Feature: Shell step definitions
  As a test author
  I want to use shell step definitions to execute and assert on arbitrary commands
  So that I can test any CLI tool including lobster itself

  # ── basic command execution ───────────────────────────────────────────────

  Scenario: I run the command executes successfully
    Given I am in a new temporary directory
    When I run the command "echo hello"
    Then the exit code should be 0

  Scenario: I run the command captures stdout
    Given I am in a new temporary directory
    When I run the command "echo hello-world"
    Then the exit code should be 0
    And the output should contain "hello-world"

  Scenario: I run the command captures stderr
    Given I am in a new temporary directory
    When I run the command "echo error-msg >&2"
    Then the exit code should be 0
    And the stderr should contain "error-msg"

  Scenario: Failed command captures non-zero exit code
    Given I am in a new temporary directory
    When I run the command "exit 42"
    Then the exit code should be 42

  Scenario: Exit code should not be assertion passes for different codes
    Given I am in a new temporary directory
    When I run the command "exit 0"
    Then the exit code should not be 1

  # ── output assertions ────────────────────────────────────────────────────

  Scenario: Output should not contain passes when text is absent
    Given I am in a new temporary directory
    When I run the command "echo present-text"
    Then the output should not contain "absent-text"

  Scenario: Stderr should not contain passes when text is absent
    Given I am in a new temporary directory
    When I run the command "echo stdout-only"
    Then the stderr should not contain "stdout-only"

  # ── JSON validation step ──────────────────────────────────────────────────

  Scenario: Output should be valid JSON passes for JSON output
    Given I am in a new temporary directory
    When I run the command "echo '{\"key\": \"value\"}'"
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: Output should be valid JSON fails for non-JSON
    Given I am in a new temporary directory
    And I create the file "features/json-fail.feature" with content:
      """
      Feature: JSON fail
        Scenario: Not JSON
          Given I am in a new temporary directory
          When I run the command "echo not-json-output"
          Then the output should be valid JSON
      """
    When I run lobster "run --features features/json-fail.feature --ci"
    Then the exit code should be 1

  # ── variable storage ─────────────────────────────────────────────────────

  Scenario: I store the output in variable makes value available
    Given I am in a new temporary directory
    When I run the command "echo stored-value"
    And I store the output in variable "MY_VAR"
    When I run the command "echo $MY_VAR"
    Then the exit code should be 0
    And the output should contain "stored-value"

  # ── I run lobster step ────────────────────────────────────────────────────

  Scenario: I run lobster invokes the lobster binary
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" containing "project: shell-step-test"
    When I run lobster "--help"
    Then the exit code should be 0
    And the output should contain "lobster"

  Scenario: I run lobster with a subcommand
    Given I am in a new temporary directory
    When I run lobster "version"
    Then the exit code should not be 0

  # ── working directory isolation ──────────────────────────────────────────

  Scenario: Commands execute in the temporary directory
    Given I am in a new temporary directory
    When I run the command "touch marker.txt"
    Then the exit code should be 0
    And the file "marker.txt" should exist

  Scenario: Working directory is restored between scenarios
    Given I am in a new temporary directory
    When I run the command "pwd"
    Then the exit code should be 0
    And the output should contain "lobster-test"

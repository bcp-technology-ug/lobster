Feature: Shell step extensions
  As a test author
  I want regex assertions, environment variables, directory changes and exit code storage
  So that I can assert patterns, control environment, and navigate the file system

  # ── regex stdout assertions ───────────────────────────────────────────────

  Scenario: Output should match passes when stdout matches the regex
    Given I am in a new temporary directory
    When I run the command "echo hello-world-42"
    Then the output should match "hello-\w+-\d+"

  Scenario: Output should match fails when stdout does not match
    Given I am in a new temporary directory
    And I create the file "features/out-match-fail.feature" with content:
      """
      Feature: Output match fail
        Scenario: Output does not match regex
          Given I am in a new temporary directory
          When I run the command "echo hello"
          Then the output should match "^\d+$"
      """
    When I run lobster "run --features features/out-match-fail.feature --ci"
    Then the exit code should be 1

  Scenario: Output should not match passes when stdout does not match the regex
    Given I am in a new temporary directory
    When I run the command "echo hello-world"
    Then the output should not match "^\d+$"

  Scenario: Output should not match fails when stdout matches
    Given I am in a new temporary directory
    And I create the file "features/out-not-match-fail.feature" with content:
      """
      Feature: Output not match fail
        Scenario: Output matches but should not
          Given I am in a new temporary directory
          When I run the command "echo 12345"
          Then the output should not match "^\d+"
      """
    When I run lobster "run --features features/out-not-match-fail.feature --ci"
    Then the exit code should be 1

  # ── regex stderr assertions ───────────────────────────────────────────────

  Scenario: Stderr should match passes when stderr matches the regex
    Given I am in a new temporary directory
    When I run the command "echo error-code-404 >&2"
    Then the stderr should match "error-code-\d{3}"

  Scenario: Stderr should not match passes when stderr does not match
    Given I am in a new temporary directory
    When I run the command "echo some-stderr-text >&2"
    Then the stderr should not match "^\d+$"

  # ── environment variable injection ────────────────────────────────────────

  Scenario: I set environment variable makes the value available in shell commands
    Given I am in a new temporary directory
    When I set environment variable "MY_SERVICE_URL" to "http://example.com"
    And I run the command "echo $MY_SERVICE_URL"
    Then the output should contain "http://example.com"

  Scenario: Multiple environment variables can be set independently
    Given I am in a new temporary directory
    When I set environment variable "APP_ENV" to "test"
    And I set environment variable "LOG_LEVEL" to "debug"
    And I run the command "echo $APP_ENV $LOG_LEVEL"
    Then the output should contain "test"
    And the output should contain "debug"

  Scenario: Environment variable overrides a previously set variable
    Given I am in a new temporary directory
    When I set environment variable "COLOR" to "blue"
    And I set environment variable "COLOR" to "red"
    And I run the command "echo $COLOR"
    Then the output should contain "red"
    And the output should not contain "blue"

  # ── changing directories ──────────────────────────────────────────────────

  Scenario: I change directory to moves the working directory
    Given I am in a new temporary directory
    And I run the command "mkdir -p subdir"
    When I change directory to "subdir"
    And I run the command "pwd"
    Then the output should match "/subdir"

  Scenario: Commands run in the changed directory
    Given I am in a new temporary directory
    And I run the command "mkdir -p workspace"
    And I change directory to "workspace"
    When I run the command "touch sentinel.txt"
    Then the file "sentinel.txt" should exist

  # ── exit code storage ─────────────────────────────────────────────────────

  Scenario: I store the exit code in variable captures the exit code
    Given I am in a new temporary directory
    When I run the command "exit 7"
    And I store the exit code in variable "LAST_CODE"
    Then the variable "LAST_CODE" should equal "7"

  Scenario: Stored exit code can be used in subsequent shell commands
    Given I am in a new temporary directory
    When I run the command "exit 2"
    And I store the exit code in variable "MY_CODE"
    And I run the command "echo code-was-$MY_CODE"
    Then the output should contain "code-was-2"

  Scenario: Exit code zero is correctly stored
    Given I am in a new temporary directory
    When I run the command "echo success"
    And I store the exit code in variable "RC"
    Then the variable "RC" should equal "0"

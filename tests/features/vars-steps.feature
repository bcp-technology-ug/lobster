Feature: Variable step definitions
  As a test author
  I want variable manipulation steps to store, transform and assert on values
  So that I can chain steps together and build dynamic test data

  # ── setting literal values ─────────────────────────────────────────────────

  Scenario: I set variable stores a literal string
    Given I am in a new temporary directory
    When I set variable "GREETING" to "hello"
    Then the variable "GREETING" should equal "hello"

  Scenario: Variable is available as shell environment variable in subsequent commands
    Given I am in a new temporary directory
    When I set variable "MYVAL" to "injected-value"
    And I run the command "echo $MYVAL"
    Then the output should contain "injected-value"

  Scenario: I set variable to a random UUID produces a well-formed UUID v4
    Given I am in a new temporary directory
    When I set variable "REQUEST_ID" to a random UUID
    Then the variable "REQUEST_ID" should not be empty
    And the variable "REQUEST_ID" should match "^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"

  Scenario: Two successive UUID variables are distinct
    Given I am in a new temporary directory
    When I set variable "ID_A" to a random UUID
    And I set variable "ID_B" to a random UUID
    Then the variable "ID_A" should not equal "${ID_B}"

  Scenario: I set variable to the current Unix timestamp produces a positive integer
    Given I am in a new temporary directory
    When I set variable "TS" to the current Unix timestamp
    Then the variable "TS" should not be empty
    And the variable "TS" should match "^\d{10,}$"

  # ── clearing variables ────────────────────────────────────────────────────

  Scenario: I clear variable removes it from the shell environment
    Given I am in a new temporary directory
    When I set variable "CLEARME" to "was-set"
    And I run the command "echo $CLEARME"
    Then the output should contain "was-set"
    When I clear variable "CLEARME"
    And I run the command "echo -n \"$CLEARME\""
    Then the output should not contain "was-set"

  # ── variable assertions ───────────────────────────────────────────────────

  Scenario: Variable should equal passes for matching value
    Given I am in a new temporary directory
    When I set variable "STATUS" to "active"
    Then the variable "STATUS" should equal "active"

  Scenario: Variable should not equal passes for a different value
    Given I am in a new temporary directory
    When I set variable "STATUS" to "active"
    Then the variable "STATUS" should not equal "inactive"

  Scenario: Variable should contain passes when value includes substring
    Given I am in a new temporary directory
    When I set variable "FULL_URL" to "https://example.com/api/v1"
    Then the variable "FULL_URL" should contain "api/v1"

  Scenario: Variable should not contain passes when substring is absent
    Given I am in a new temporary directory
    When I set variable "LABEL" to "production"
    Then the variable "LABEL" should not contain "staging"

  Scenario: Variable should match passes for a matching regex
    Given I am in a new temporary directory
    When I set variable "VERSION" to "v2.4.1"
    Then the variable "VERSION" should match "^v\d+\.\d+\.\d+$"

  Scenario: Variable should not be empty passes for a non-empty value
    Given I am in a new temporary directory
    When I set variable "NONEMPTY" to "something"
    Then the variable "NONEMPTY" should not be empty

  Scenario: Variable should not be empty fails when variable is unset
    Given I am in a new temporary directory
    And I create the file "features/var-empty-fail.feature" with content:
      """
      Feature: Var empty fail
        Scenario: Unset variable fails not-empty check
          Given I am in a new temporary directory
          Then the variable "UNSET_VAR" should not be empty
      """
    When I run lobster "run --features features/var-empty-fail.feature --ci"
    Then the exit code should be 1

  # ── extracting values from HTTP responses ─────────────────────────────────

  @docker
  Scenario: I set variable from JSON field stores a string field
    Given I am in a new temporary directory
    And I set the base URL to "${FIXTURE_URL}"
    When I send a GET request to "/token.json"
    Then the response status should be 200
    And I set variable "TOKEN" from JSON field "token" in the response
    And the variable "TOKEN" should equal "abc123"

  @docker
  Scenario: I store JSON field from the response in variable is an ergonomic alias
    Given I am in a new temporary directory
    And I set the base URL to "${FIXTURE_URL}"
    When I send a GET request to "/count.json"
    Then the response status should be 200
    And I store JSON field "count" from the response in variable "COUNT"
    And the variable "COUNT" should equal "42"

  # ── storing response headers ──────────────────────────────────────────────

  @docker
  Scenario: I store the response header in variable captures the header value
    Given I am in a new temporary directory
    And I set the base URL to "${HTTPBIN_URL}"
    When I send a GET request to "/response-headers?X-Request-Id=req-42"
    Then the response status should be 200
    And I store the response header "X-Request-Id" in variable "REQ_ID"
    And the variable "REQ_ID" should equal "req-42"

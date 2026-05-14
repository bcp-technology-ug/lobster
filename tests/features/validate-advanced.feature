@covers:cli:validate
Feature: lobster validate — advanced scenarios
  As a developer
  I want lobster validate to provide detailed feedback on feature files
  So that I can confidently ensure parse correctness before execution

  # ── strict mode ───────────────────────────────────────────────────────────

  Scenario: --strict on a valid file exits 0 and prints strict indicator
    Given I am in a new temporary directory
    And I create the file "features/valid.feature" with content:
      """
      Feature: All good
        Scenario: Passes
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "validate --features features/valid.feature --strict"
    Then the exit code should be 0
    And the output should contain "strict"

  # ── JSON output format ────────────────────────────────────────────────────

  Scenario: --format json lists status ok for a valid file
    Given I am in a new temporary directory
    And I create the file "features/ok.feature" with content:
      """
      Feature: JSON ok
        Scenario: First
          Given I set the base URL to "http://example.com"
        Scenario: Second
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "validate --features features/ok.feature --format json"
    Then the exit code should be 0
    And the output should be valid JSON
    And the output should contain "\"status\": \"ok\""
    And the output should contain "\"scenarios\": 2"

  Scenario: --format json reports status error for an unparseable file
    Given I am in a new temporary directory
    And I create the file "features/broken.feature" with content:
      """
      this is not gherkin at all
      random text that will fail parsing
      """
    When I run lobster "validate --features features/broken.feature --format json"
    Then the exit code should be 2
    And the output should be valid JSON
    And the output should contain "\"status\": \"error\""

  # ── multiple files ────────────────────────────────────────────────────────

  Scenario: Multiple valid files all reported as passing
    Given I am in a new temporary directory
    And I create the file "features/a.feature" with content:
      """
      Feature: File A
        Scenario: A scenario
          Given I set the base URL to "http://example.com"
      """
    And I create the file "features/b.feature" with content:
      """
      Feature: File B
        Scenario: B scenario
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "validate --features features/a.feature"
    Then the exit code should be 0
    And the output should contain "1 file"

  Scenario: Mix of valid and invalid files exits 2
    Given I am in a new temporary directory
    And I create the file "features/good.feature" with content:
      """
      Feature: Good
        Scenario: Passes
          Given I set the base URL to "http://example.com"
      """
    And I create the file "features/bad.feature" with content:
      """
      not gherkin
      """
    When I run lobster "validate --features features/good.feature"
    Then the exit code should be 0
    When I run lobster "validate --features features/bad.feature"
    Then the exit code should be 2

  # ── no files matched ─────────────────────────────────────────────────────

  Scenario: No files matched shows warning and exits 0
    Given I am in a new temporary directory
    When I run lobster "validate --features features/nowhere/*.feature"
    Then the exit code should be 0
    And the output should contain "No feature files found"

  # ── file counts ───────────────────────────────────────────────────────────

  Scenario: Output contains scenario and step counts
    Given I am in a new temporary directory
    And I create the file "features/counts.feature" with content:
      """
      Feature: Count check
        Scenario: Step counter
          Given I set the base URL to "http://example.com"
          When I send a GET request to "/ping"
          Then the response status should be 200
      """
    When I run lobster "validate --features features/counts.feature"
    Then the exit code should be 0
    And the output should contain "1 scenarios"

  # ── config file flag ──────────────────────────────────────────────────────

  Scenario: --config pointing at a non-existent file exits with error
    Given I am in a new temporary directory
    When I run lobster "--config /nonexistent/lobster.yaml validate --features features/any.feature"
    Then the exit code should not be 0

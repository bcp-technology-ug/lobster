@covers:cli:lint
Feature: lobster lint — advanced scenarios
  As a developer
  I want lobster lint to report all quality issues comprehensively
  So that poorly-written specs are caught early with clear diagnostics

  # ── warnings vs errors ────────────────────────────────────────────────────

  Scenario: Empty feature file produces a warning (not an error) without --strict
    Given I am in a new temporary directory
    And I create the file "features/empty.feature" with content:
      """
      Feature: Empty
      """
    When I run lobster "lint --features features/empty.feature"
    Then the exit code should be 0
    And the output should contain "warning"

  Scenario: Empty feature warning does not cause failure without --strict
    Given I am in a new temporary directory
    And I create the file "features/empty.feature" with content:
      """
      Feature: Empty
      """
    When I run lobster "lint --features features/empty.feature"
    Then the exit code should be 0
    And the output should contain "1 warning"

  Scenario: --strict promotes warning to failure and output contains strict label
    Given I am in a new temporary directory
    And I create the file "features/empty.feature" with content:
      """
      Feature: No scenarios here
      """
    When I run lobster "lint --features features/empty.feature --strict"
    Then the exit code should be 2
    And the output should contain "strict"

  # ── error rules ───────────────────────────────────────────────────────────

  Scenario: Feature with no name triggers an error
    Given I am in a new temporary directory
    And I create the file "features/noname.feature" with content:
      """
      Feature:
        Scenario: Orphaned
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "lint --features features/noname.feature"
    Then the exit code should be 2
    And the output should contain "name"

  Scenario: Scenario with no steps triggers an error
    Given I am in a new temporary directory
    And I create the file "features/nosteps.feature" with content:
      """
      Feature: Missing steps

        Scenario: No steps here
      """
    When I run lobster "lint --features features/nosteps.feature"
    Then the exit code should be 2

  Scenario: Scenario with empty step text triggers an error
    Given I am in a new temporary directory
    And I create the file "features/emptystep.feature" with content:
      """
      Feature: Empty step

        Scenario: Has empty step
          Given
      """
    When I run lobster "lint --features features/emptystep.feature"
    Then the exit code should be 2

  # ── parse errors ─────────────────────────────────────────────────────────

  Scenario: File with parse error is reported and lint exits 2
    Given I am in a new temporary directory
    And I create the file "features/broken.feature" with content:
      """
      not valid gherkin at all
      """
    When I run lobster "lint --features features/broken.feature"
    Then the exit code should be 2
    And the output should contain "parse error"

  # ── clean file ───────────────────────────────────────────────────────────

  Scenario: Feature file with all required elements passes cleanly
    Given I am in a new temporary directory
    And I create the file "features/perfect.feature" with content:
      """
      Feature: Perfect feature

        Background:
          Given I set the base URL to "http://example.com"

        @smoke
        Scenario: Named and tagged
          When I send a GET request to "/ping"
          Then the response status should be 200

        Scenario: Second named scenario
          When I send a GET request to "/health"
          Then the response status should be 200
      """
    When I run lobster "lint --features features/perfect.feature"
    Then the exit code should be 0
    And the output should contain "All checks passed"

  # ── multiple diagnostics ──────────────────────────────────────────────────

  Scenario: Multiple issues reported across the same file
    Given I am in a new temporary directory
    And I create the file "features/multi.feature" with content:
      """
      Feature: Multi issues

        Scenario:

        Scenario: Also no steps
      """
    When I run lobster "lint --features features/multi.feature"
    Then the exit code should be 2
    And the output should contain "error"

  # ── no files matched ─────────────────────────────────────────────────────

  Scenario: No files matched exits 0 with warning message
    Given I am in a new temporary directory
    When I run lobster "lint --features features/nowhere/*.feature"
    Then the exit code should be 0
    And the output should contain "No feature files found"

  # ── output summary ────────────────────────────────────────────────────────

  Scenario: Summary line always shows file count
    Given I am in a new temporary directory
    And I create the file "features/clean.feature" with content:
      """
      Feature: Clean

        Scenario: Good
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "lint --features features/clean.feature"
    Then the exit code should be 0
    And the output should contain "1 file"

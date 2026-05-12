Feature: lobster lint command
  As a developer
  I want lobster lint to enforce quality rules on feature files
  So that poorly-written specs are caught early

  Scenario: Clean feature file reports no issues
    Given I am in a new temporary directory
    And I create the file "features/clean.feature" with content:
      """
      Feature: Clean example
        Scenario: First check
          Given I set the base URL to "http://example.com"
          When I send a GET request to "/ping"
          Then the response status should be 200
      """
    When I run lobster "lint --features features/clean.feature"
    Then the exit code should be 0
    And the output should contain "All checks passed"

  Scenario: Scenario without a name triggers a lint error
    Given I am in a new temporary directory
    And I create the file "features/noname.feature" with content:
      """
      Feature: No scenario name

        Scenario:
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "lint --features features/noname.feature"
    Then the exit code should be 2
    And the output should contain "name"

  Scenario: Scenario with no steps triggers a lint error
    Given I am in a new temporary directory
    And I create the file "features/nosteps.feature" with content:
      """
      Feature: No steps

        Scenario: Empty scenario
      """
    When I run lobster "lint --features features/nosteps.feature"
    Then the exit code should be 2

  Scenario: --strict elevates a warning to an error
    Given I am in a new temporary directory
    And I create the file "features/empty.feature" with content:
      """
      Feature: Empty feature with no scenarios
      """
    When I run lobster "lint --features features/empty.feature --strict"
    Then the exit code should be 2
    And the output should contain "strict"

  Scenario: Missing --features flag exits 2
    Given I am in a new temporary directory
    When I run lobster "lint"
    Then the exit code should be 2

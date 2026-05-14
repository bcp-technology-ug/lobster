@covers:cli:plan @covers:PlanService.Plan @covers:PlanService.ListPlans
Feature: lobster plan command
  As a developer
  I want lobster plan to show which scenarios will run
  So that I can preview execution order before running steps

  Scenario: Plan lists all scenarios in the feature file
    Given I am in a new temporary directory
    And I create the file "features/two.feature" with content:
      """
      Feature: Two scenarios
        Scenario: First scenario
          Given I set the base URL to "http://example.com"

        @smoke
        Scenario: Second scenario
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "plan --features features/two.feature"
    Then the exit code should be 0
    And the output should contain "First scenario"
    And the output should contain "Second scenario"

  Scenario: --tags filters to only matching scenarios
    Given I am in a new temporary directory
    And I create the file "features/tagged.feature" with content:
      """
      Feature: Tag filtering
        Scenario: Untagged
          Given I set the base URL to "http://example.com"

        @smoke
        Scenario: Smoke tagged
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "plan --features features/tagged.feature --tags @smoke"
    Then the exit code should be 0
    And the output should contain "Smoke tagged"
    And the output should not contain "Untagged"

  Scenario: --scenario-regex filters by name substring
    Given I am in a new temporary directory
    And I create the file "features/regex.feature" with content:
      """
      Feature: Regex filter
        Scenario: Alpha check
          Given I set the base URL to "http://example.com"

        Scenario: Beta check
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "plan --features features/regex.feature --scenario-regex Alpha"
    Then the exit code should be 0
    And the output should contain "Alpha check"

  Scenario: --format json produces valid JSON output
    Given I am in a new temporary directory
    And I create the file "features/json.feature" with content:
      """
      Feature: JSON plan
        Scenario: One
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "plan --features features/json.feature --format json"
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: --out writes plan JSON to a file
    Given I am in a new temporary directory
    And I create the file "features/out.feature" with content:
      """
      Feature: Out file
        Scenario: Save me
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "plan --features features/out.feature --out plan.json"
    Then the exit code should be 0
    And the file "plan.json" should exist

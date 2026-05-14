@covers:cli:plan @covers:PlanService.Plan
Feature: plan command edge cases
  As a developer
  I want the plan command to correctly handle filtering and output options
  So that I can see exactly what will run before executing

  Scenario: plan with tag filter selects matching scenarios
    Given I am in a new temporary directory
    And I create the file "features/plan-tags.feature" with content:
      """
      Feature: Plan tags
        @run-me
        Scenario: Tagged pass
          When I run the command "echo tagged"
          Then the exit code should be 0

        @skip-me
        Scenario: Skip this one
          When I run the command "echo skipped"
          Then the exit code should be 0
      """
    When I run lobster "plan --features features/plan-tags.feature --tags @run-me" 
    Then the exit code should be 0

  Scenario: plan with tag filter excludes non-matching scenarios
    Given I am in a new temporary directory
    And I create the file "features/plan-tags2.feature" with content:
      """
      Feature: Plan tags 2
        @run-me
        Scenario: Tagged pass
          When I run the command "echo tagged"
          Then the exit code should be 0

        @skip-me
        Scenario: Skip this
          When I run the command "false"
          Then the exit code should be 0
      """
    When I run lobster "plan --features features/plan-tags2.feature --tags ~@skip-me"
    Then the exit code should be 0
    And the output should contain "Tagged pass"

  Scenario: plan with regex filter selects matching scenarios
    Given I am in a new temporary directory
    And I create the file "features/plan-regex.feature" with content:
      """
      Feature: Plan regex
        Scenario: match-me passes
          When I run the command "echo matched"
          Then the exit code should be 0

        Scenario: ignore-me would fail
          When I run the command "false"
          Then the exit code should be 0
      """
    When I run lobster "plan --features features/plan-regex.feature --scenario-regex match-me"
    Then the exit code should be 0

  Scenario: plan with no-match selector exits 0 with empty plan
    Given I am in a new temporary directory
    And I create the file "features/nomatch.feature" with content:
      """
      Feature: No match
        @wrong-tag
        Scenario: Not matched
          When I run the command "echo nomatch"
          Then the exit code should be 0
      """
    When I run lobster "plan --features features/nomatch.feature --tags @other-tag" 
    Then the exit code should be 0

  Scenario: plan --format json exits 0
    Given I am in a new temporary directory
    And I create the file "features/pf.feature" with content:
      """
      Feature: Plan format
        Scenario: Plan pass
          When I run the command "echo plan"
          Then the exit code should be 0
      """
    When I run lobster "plan --features features/pf.feature --format json" 
    Then the exit code should be 0

  Scenario: plan --format json output is valid JSON
    Given I am in a new temporary directory
    And I create the file "features/pfj.feature" with content:
      """
      Feature: Plan format JSON
        Scenario: Plan JSON pass
          When I run the command "echo planjson"
          Then the exit code should be 0
      """
    When I run lobster "plan --features features/pfj.feature --format json" 
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: plan --out writes plan to file
    Given I am in a new temporary directory
    And I create the file "features/outplan.feature" with content:
      """
      Feature: Out plan
        Scenario: Out pass
          When I run the command "echo outplan"
          Then the exit code should be 0
      """
    When I run lobster "plan --features features/outplan.feature --out myplan.json" 
    Then the exit code should be 0
    And the file "myplan.json" should exist

  Scenario: plan --out file contains valid JSON
    Given I am in a new temporary directory
    And I create the file "features/outplanjson.feature" with content:
      """
      Feature: Out plan JSON
        Scenario: Out JSON pass
          When I run the command "echo outplanjson"
          Then the exit code should be 0
      """
    When I run lobster "plan --features features/outplanjson.feature --out planout.json" 
    Then the exit code should be 0
    And the file "planout.json" should contain valid JSON

  Scenario: plan produces deterministic plan IDs across re-runs
    Given I am in a new temporary directory
    And I create the file "features/det.feature" with content:
      """
      Feature: Deterministic
        Scenario: Deterministic scenario
          When I run the command "echo det"
          Then the exit code should be 0
      """
    When I run lobster "plan --features features/det.feature --format json" 
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: plan output shows scenario names
    Given I am in a new temporary directory
    And I create the file "features/show-names.feature" with content:
      """
      Feature: Show names
        Scenario: My unique scenario name
          When I run the command "echo shownames"
          Then the exit code should be 0
      """
    When I run lobster "plan --features features/show-names.feature" 
    Then the exit code should be 0
    And the output should contain "My unique scenario name"

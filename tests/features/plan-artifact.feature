Feature: lobster plan artifact
  As a developer
  I want to save a plan and then execute from it
  So that I can separate the planning phase from execution in CI pipelines

  Scenario: lobster plan --out writes a plan artifact
    Given I am in a new temporary directory
    And I create the file "features/plan.feature" with content:
      """
      Feature: Plan test
        Scenario: First
          Given I am in a new temporary directory
          When I run the command "echo first"
          Then the exit code should be 0
        Scenario: Second
          Given I am in a new temporary directory
          When I run the command "echo second"
          Then the exit code should be 0
      """
    When I run lobster "plan --features features/plan.feature --out plan.json"
    Then the exit code should be 0
    And the file "plan.json" should exist

  Scenario: lobster run --from-plan executes the saved plan
    Given I am in a new temporary directory
    And I create the file "features/plan-exec.feature" with content:
      """
      Feature: Plan exec
        Scenario: Only
          Given I am in a new temporary directory
          When I run the command "echo plan-exec"
          Then the exit code should be 0
      """
    And I run lobster "plan --features features/plan-exec.feature --out exec-plan.json"
    When I run lobster "run --from-plan exec-plan.json --ci"
    Then the exit code should be 0

  Scenario: lobster plan with no matching features exits 0 with empty plan
    Given I am in a new temporary directory
    When I run lobster "plan --features features/nonexistent/*.feature"
    Then the exit code should be 0

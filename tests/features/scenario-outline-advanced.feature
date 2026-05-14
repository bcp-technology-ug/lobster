@covers:cli:run @covers:cli:plan
Feature: scenario outline advanced cases
  As a developer
  I want Scenario Outlines to handle multiple Examples tables and edge cases
  So that parameterized tests work robustly

  Scenario: outline with multiple Examples tables runs all rows
    Given I am in a new temporary directory
    And I create the file "features/multi-examples.feature" with content:
      """
      Feature: Multi-examples
        Scenario Outline: Param <name>
          When I run the command "echo <value>"
          Then the exit code should be 0
          Examples: First table
            | name | value |
            | foo  | hello |
            | bar  | world |
          Examples: Second table
            | name | value |
            | baz  | qux   |
            | quux | corge |
      """
    When I run lobster "run --features features/multi-examples.feature --ci"
    Then the exit code should be 0

  Scenario: outline variable used in step text
    Given I am in a new temporary directory
    And I create the file "features/var-step.feature" with content:
      """
      Feature: Variable in step text
        Scenario Outline: Greet <person>
          When I run the command "echo Hello <person>"
          Then the exit code should be 0
          And the output should contain "<person>"
          Examples:
            | person |
            | Alice  |
            | Bob    |
      """
    When I run lobster "run --features features/var-step.feature --ci"
    Then the exit code should be 0

  Scenario: outline with 0 example rows exits 0 with no scenarios
    Given I am in a new temporary directory
    And I create the file "features/empty-examples.feature" with content:
      """
      Feature: Empty examples
        Scenario Outline: Zero rows <item>
          When I run the command "echo <item>"
          Then the exit code should be 0
          Examples:
            | item |
      """
    When I run lobster "run --features features/empty-examples.feature --ci"
    Then the exit code should be 0

  Scenario: plan shows all expanded outline rows
    Given I am in a new temporary directory
    And I create the file "features/outline-plan.feature" with content:
      """
      Feature: Outline plan
        Scenario Outline: Item <id>
          When I run the command "echo <id>"
          Then the exit code should be 0
          Examples:
            | id |
            | 1  |
            | 2  |
            | 3  |
      """
    When I run lobster "plan --features features/outline-plan.feature"
    Then the exit code should be 0

  Scenario: outline generates deterministic scenario IDs across re-plans
    Given I am in a new temporary directory
    And I create the file "features/det-outline.feature" with content:
      """
      Feature: Deterministic outline
        Scenario Outline: Det <x>
          When I run the command "echo <x>"
          Then the exit code should be 0
          Examples:
            | x |
            | a |
            | b |
      """
    When I run lobster "plan --features features/det-outline.feature --format json"
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: outline variable in DocString is substituted
    Given I am in a new temporary directory
    And I create the file "features/docstring-outline.feature" with content:
      """
      Feature: DocString outline
        Scenario Outline: DocString <lang>
          Given I have content:
            \"\"\"
            Language: <lang>
            \"\"\"
          When I run the command "echo <lang>"
          Then the exit code should be 0
          Examples:
            | lang |
            | go   |
            | py   |
      """
    When I run lobster "run --features features/docstring-outline.feature --ci"
    Then the exit code should be 0

  Scenario: outline tag filter applies to all expanded rows
    Given I am in a new temporary directory
    And I create the file "features/outline-tagged.feature" with content:
      """
      Feature: Outline tagged
        @run-me
        Scenario Outline: Tagged <x>
          When I run the command "echo <x>"
          Then the exit code should be 0
          Examples:
            | x |
            | a |
            | b |

        @skip-me
        Scenario Outline: Skipped <x>
          When I run the command "false"
          Then the exit code should be 0
          Examples:
            | x |
            | c |
      """
    When I run lobster "run --features features/outline-tagged.feature --tags @run-me --ci"
    Then the exit code should be 0

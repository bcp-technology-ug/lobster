@covers:cli:run @covers:cli:plan
Feature: Scenario Outline with Examples
  As a test author
  I want to use Scenario Outline with Examples tables
  So that I can run the same scenario logic across multiple data sets

  Scenario Outline: Run succeeds for a parametrised undefined feature
    Given I am in a new temporary directory
    And I create the file "features/outline.feature" with content:
      """
      Feature: Outline test
        Scenario: <label> scenario
          Given a step named <label>
      """
    When I run lobster "run --features features/outline.feature --ci"
    Then the exit code should be 0
    And the output should contain "<label>"

    Examples:
      | label   |
      | alpha   |
      | beta    |
      | gamma   |

  Scenario Outline: lobster validate accepts feature files without errors
    Given I am in a new temporary directory
    And I create the file "features/<name>.feature" with content:
      """
      Feature: <name>
        Scenario: Basic <name> scenario
          Given a placeholder step
      """
    When I run lobster "validate --features features/<name>.feature"
    Then the exit code should be 0

    Examples:
      | name    |
      | smoke   |
      | sanity  |

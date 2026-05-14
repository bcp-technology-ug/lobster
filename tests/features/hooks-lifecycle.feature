@covers:cli:run
Feature: hooks lifecycle
  As a developer
  I want the lobster framework to handle scenario setup and teardown
  So that each scenario runs in an isolated environment

  Scenario: each scenario runs in a fresh temporary directory
    Given I am in a new temporary directory
    And I create the file "features/isolation.feature" with content:
      """
      Feature: Isolation
        Scenario: First writes a file
          Given I am in a new temporary directory
          When I run the command "touch sentinel.txt"
          Then the exit code should be 0

        Scenario: Second does not see that file
          Given I am in a new temporary directory
          Then the file "sentinel.txt" should not exist
      """
    When I run lobster "run --features features/isolation.feature --ci"
    Then the exit code should be 0

  Scenario: failed scenario causes run to exit non-zero
    Given I am in a new temporary directory
    And I create the file "features/fail.feature" with content:
      """
      Feature: Fail
        Scenario: This will fail
          When I run the command "false"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/fail.feature --ci"
    Then the exit code should not be 0

  Scenario: run continues after one scenario fails and reports total count
    Given I am in a new temporary directory
    And I create the file "features/mixed.feature" with content:
      """
      Feature: Mixed
        Scenario: Pass
          When I run the command "echo pass"
          Then the exit code should be 0

        Scenario: Fail
          When I run the command "false"
          Then the exit code should be 0

        Scenario: Also pass
          When I run the command "echo alsopass"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/mixed.feature --ci"
    Then the exit code should not be 0
    And the output should contain "2"

  Scenario: run with all passing scenarios reports clean output
    Given I am in a new temporary directory
    And I create the file "features/allpass.feature" with content:
      """
      Feature: All pass
        Scenario: One
          When I run the command "echo one"
          Then the exit code should be 0

        Scenario: Two
          When I run the command "echo two"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/allpass.feature --ci"
    Then the exit code should be 0
    And the output should not be empty


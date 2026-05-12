Feature: Soft-assert mode collects multiple failures
  As a test author
  I want --soft-assert to collect all assertion failures before failing the scenario
  So that I can see all regressions in a single run

  Scenario: --soft-assert flag is accepted by lobster run
    Given I am in a new temporary directory
    And I create the file "features/soft.feature" with content:
      """
      Feature: Soft assert feature
        Scenario: Multiple assertions
          Given I am in a new temporary directory
          When I run the command "echo hello"
          Then the output should contain "hello"
          And the output should contain "world"
      """
    When I run lobster "run --features features/soft.feature --soft-assert --ci"
    Then the exit code should be 1
    And the output should contain "world"

  Scenario: Without --soft-assert the run still exits 1 on first assertion failure
    Given I am in a new temporary directory
    And I create the file "features/hard.feature" with content:
      """
      Feature: Hard assert feature
        Scenario: Assertion failure
          Given I am in a new temporary directory
          When I run the command "echo hello"
          Then the output should contain "not-present"
      """
    When I run lobster "run --features features/hard.feature --ci"
    Then the exit code should be 1

  Scenario: Passing scenario exits 0 with --soft-assert
    Given I am in a new temporary directory
    And I create the file "features/pass.feature" with content:
      """
      Feature: Passing soft assert
        Scenario: All pass
          Given I am in a new temporary directory
          When I run the command "echo all-good"
          Then the output should contain "all-good"
      """
    When I run lobster "run --features features/pass.feature --soft-assert --ci"
    Then the exit code should be 0

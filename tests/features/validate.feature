Feature: lobster validate command
  As a developer
  I want lobster validate to catch Gherkin problems
  So that broken feature files are rejected before execution

  Scenario: Valid feature file reports success
    Given I am in a new temporary directory
    And I create the file "features/ok.feature" with content:
      """
      Feature: Greet users
        Scenario: Hello world
          Given I set the base URL to "http://example.com"
          When I send a GET request to "/hello"
          Then the response status should be 200
      """
    When I run lobster "validate --features features/ok.feature"
    Then the exit code should be 0
    And the output should contain "1 file"

  Scenario: Invalid Gherkin syntax reports error and exits 2
    Given I am in a new temporary directory
    And I create the file "features/bad.feature" with content:
      """
      not a feature file
      just random text
      no gherkin structure
      """
    When I run lobster "validate --features features/bad.feature"
    Then the exit code should be 2
    And the output should contain "error"

  Scenario: JSON format output is valid JSON
    Given I am in a new temporary directory
    And I create the file "features/ok.feature" with content:
      """
      Feature: JSON check
        Scenario: Passes
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "validate --features features/ok.feature --format json"
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: Missing --features flag exits 2
    Given I am in a new temporary directory
    When I run lobster "validate"
    Then the exit code should be 2

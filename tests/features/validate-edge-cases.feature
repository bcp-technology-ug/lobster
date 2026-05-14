@covers:cli:validate
Feature: validate command edge cases
  As a developer
  I want the validate command to handle unusual file content correctly
  So that validation is robust across diverse feature file formats

  Scenario: validate feature file with CRLF line endings exits 0
    Given I am in a new temporary directory
    And I create the file "features/crlf.feature" with content:
      """
      Feature: CRLF
        Scenario: CRLF scenario
          When I run the command "echo crlf"
          Then the exit code should be 0
      """
    When I run lobster "validate --features features/crlf.feature"
    Then the exit code should be 0

  Scenario: validate feature file with DataTable exits 0
    Given I am in a new temporary directory
    And I create the file "features/datatable.feature" with content:
      """
      Feature: DataTable
        Scenario: DataTable scenario
          Given the following data:
            | name  | value |
            | foo   | bar   |
            | hello | world |
          When I run the command "echo datatables"
          Then the exit code should be 0
      """
    When I run lobster "validate --features features/datatable.feature"
    Then the exit code should be 0

  Scenario: validate feature file with DocString exits 0
    Given I am in a new temporary directory
    And I create the file "features/docstring.feature" with content:
      """
      Feature: DocString
        Scenario: DocString scenario
          Given I have document:
            \"\"\"
            some content
            goes here
            \"\"\"
          When I run the command "echo docstring"
          Then the exit code should be 0
      """
    When I run lobster "validate --features features/docstring.feature"
    Then the exit code should be 0

  Scenario: validate feature file with Scenario Outline exits 0
    Given I am in a new temporary directory
    And I create the file "features/outline.feature" with content:
      """
      Feature: Outline
        Scenario Outline: Parameterized <name>
          When I run the command "echo <value>"
          Then the exit code should be 0
          Examples:
            | name | value |
            | foo  | hello |
            | bar  | world |
      """
    When I run lobster "validate --features features/outline.feature"
    Then the exit code should be 0

  Scenario: validate feature file with only Background steps exits 0
    Given I am in a new temporary directory
    And I create the file "features/bgonly.feature" with content:
      """
      Feature: Background only
        Background:
          Given I am in a new temporary directory
      """
    When I run lobster "validate --features features/bgonly.feature"
    Then the exit code should be 0

  Scenario: validate with --strict flag exits 0 on valid file
    Given I am in a new temporary directory
    And I create the file "features/strict.feature" with content:
      """
      Feature: Strict valid
        Scenario: Strict pass
          When I run the command "echo strict"
          Then the exit code should be 0
      """
    When I run lobster "validate --features features/strict.feature --strict"
    Then the exit code should be 0

  Scenario: validate --format json exits 0 on valid file
    Given I am in a new temporary directory
    And I create the file "features/jsonout.feature" with content:
      """
      Feature: JSON output
        Scenario: JSON pass
          When I run the command "echo json"
          Then the exit code should be 0
      """
    When I run lobster "validate --features features/jsonout.feature --format json"
    Then the exit code should be 0

  Scenario: validate --format json output is valid JSON
    Given I am in a new temporary directory
    And I create the file "features/jsonout2.feature" with content:
      """
      Feature: JSON output 2
        Scenario: JSON pass 2
          When I run the command "echo json2"
          Then the exit code should be 0
      """
    When I run lobster "validate --features features/jsonout2.feature --format json"
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: validate with glob that matches no files exits 0
    Given I am in a new temporary directory
    When I run lobster "validate --features nonexistent/**/*.feature"
    Then the exit code should be 0

  Scenario: validate multiple files in one invocation exits 0
    Given I am in a new temporary directory
    And I create the file "features/a.feature" with content:
      """
      Feature: A
        Scenario: A
          When I run the command "echo a"
          Then the exit code should be 0
      """
    And I create the file "features/b.feature" with content:
      """
      Feature: B
        Scenario: B
          When I run the command "echo b"
          Then the exit code should be 0
      """
    When I run lobster "validate --features features/*.feature"
    Then the exit code should be 0

  Scenario: validate feature file with multiple scenarios exits 0
    Given I am in a new temporary directory
    And I create the file "features/multi.feature" with content:
      """
      Feature: Multi
        Scenario: One
          When I run the command "echo one"
          Then the exit code should be 0

        Scenario: Two
          When I run the command "echo two"
          Then the exit code should be 0

        Scenario: Three
          When I run the command "echo three"
          Then the exit code should be 0
      """
    When I run lobster "validate --features features/multi.feature"
    Then the exit code should be 0

  Scenario: validate feature file with tagged scenarios exits 0
    Given I am in a new temporary directory
    And I create the file "features/tags.feature" with content:
      """
      @smoke
      Feature: Tags
        @fast @ci
        Scenario: Tagged pass
          When I run the command "echo tagged"
          Then the exit code should be 0
      """
    When I run lobster "validate --features features/tags.feature"
    Then the exit code should be 0

Feature: Filesystem step definitions
  As a test author
  I want filesystem step definitions to create and verify files and directories
  So that I can assert on the side effects of CLI commands

  # ── temporary directory ───────────────────────────────────────────────────

  Scenario: I am in a new temporary directory creates a writable temp dir
    Given I am in a new temporary directory
    When I run the command "touch test-file.txt"
    Then the exit code should be 0
    And the file "test-file.txt" should exist

  Scenario: Each scenario gets an independent temporary directory
    Given I am in a new temporary directory
    And I create the file "unique-to-this-scenario.txt" containing "unique"
    Then the file "unique-to-this-scenario.txt" should exist

  # ── creating files ────────────────────────────────────────────────────────

  Scenario: I create the file with inline content creates a readable file
    Given I am in a new temporary directory
    When I create the file "hello.txt" containing "hello world"
    Then the file "hello.txt" should exist

  Scenario: Created file contains the specified inline content
    Given I am in a new temporary directory
    When I create the file "greeting.txt" containing "good morning"
    Then the file "greeting.txt" should contain "good morning"

  Scenario: I create the file with DocString content creates the file
    Given I am in a new temporary directory
    When I create the file "config.yaml" with content:
      """
      key: value
      nested:
        inner: data
      """
    Then the file "config.yaml" should exist

  Scenario: DocString content is written to the file verbatim
    Given I am in a new temporary directory
    When I create the file "config.yaml" with content:
      """
      key: value
      """
    Then the file "config.yaml" should contain "key: value"

  # ── file existence assertions ─────────────────────────────────────────────

  Scenario: File should exist passes for a created file
    Given I am in a new temporary directory
    When I create the file "present.txt" containing "present"
    Then the file "present.txt" should exist

  Scenario: File should not exist passes for an absent file
    Given I am in a new temporary directory
    Then the file "absent.txt" should not exist

  Scenario: File should not exist fails when the file is present
    Given I am in a new temporary directory
    And I create the file "features/fs-fail.feature" with content:
      """
      Feature: FS fail
        Scenario: Present file should not exist fails
          Given I am in a new temporary directory
          And I create the file "present.txt" containing "data"
          Then the file "present.txt" should not exist
      """
    When I run lobster "run --features features/fs-fail.feature --ci"
    Then the exit code should be 1

  # ── directory existence assertions ────────────────────────────────────────

  Scenario: Directory should exist passes after mkdir
    Given I am in a new temporary directory
    When I run the command "mkdir -p mydir/subdir"
    Then the exit code should be 0
    And the directory "mydir" should exist

  Scenario: Directory should not exist passes for absent dir
    Given I am in a new temporary directory
    Then the directory "nonexistent-dir" should not exist

  Scenario: Nested directory created by command is detectable
    Given I am in a new temporary directory
    When I run the command "mkdir -p a/b/c"
    Then the directory "a/b/c" should exist

  # ── file content assertions ────────────────────────────────────────────────

  Scenario: File should contain passes when content is present
    Given I am in a new temporary directory
    And I create the file "data.txt" containing "search-term"
    Then the file "data.txt" should contain "search-term"

  Scenario: File should contain works with multiline DocString files
    Given I am in a new temporary directory
    And I create the file "multi.yaml" with content:
      """
      first: line
      second: line
      third: line
      """
    Then the file "multi.yaml" should contain "second: line"

  # ── creating directories via nested file paths ────────────────────────────

  Scenario: Creating a file in a nested path auto-creates parent directories
    Given I am in a new temporary directory
    When I create the file "deep/nested/file.txt" containing "deep content"
    Then the file "deep/nested/file.txt" should exist
    And the directory "deep/nested" should exist
    And the file "deep/nested/file.txt" should contain "deep content"

  # ── feature file scaffold ─────────────────────────────────────────────────

  Scenario: A feature file created inline can be used with lobster run
    Given I am in a new temporary directory
    And I create the file "features/inline.feature" with content:
      """
      Feature: Inline
        Scenario: Inline scenario
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/inline.feature --ci"
    Then the exit code should be 0
    And the output should contain "1 passed"

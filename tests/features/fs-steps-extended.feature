Feature: Filesystem step extensions
  As a test author
  I want extended filesystem steps for reading files, regex assertions, YAML/JSON validation,
  appending, deleting, and directory membership checks
  So that I can fully verify file-based side effects of CLI tools

  # ── reading files into variables ──────────────────────────────────────────

  Scenario: I read the file into variable loads the file content
    Given I am in a new temporary directory
    And I create the file "data.txt" containing "file-content-xyz"
    When I read the file "data.txt" into variable "FILE_CONTENT"
    Then the variable "FILE_CONTENT" should contain "file-content-xyz"

  Scenario: File content read into variable can be used in subsequent assertions
    Given I am in a new temporary directory
    And I create the file "config.txt" containing "host=localhost"
    When I read the file "config.txt" into variable "CFG"
    Then the variable "CFG" should match "host=\w+"

  # ── file content regex assertions ─────────────────────────────────────────

  Scenario: File content should match passes when file matches the regex
    Given I am in a new temporary directory
    And I create the file "output.log" with content:
      """
      [2026-05-14] ERROR code=500 message=internal error
      [2026-05-14] INFO  code=200 message=ok
      """
    Then the file "output.log" content should match "ERROR code=\d+"

  Scenario: File content should not match passes when file does not match regex
    Given I am in a new temporary directory
    And I create the file "clean.log" containing "INFO all systems go"
    Then the file "clean.log" content should not match "ERROR|FATAL|PANIC"

  Scenario: File content should match fails when file does not match
    Given I am in a new temporary directory
    And I create the file "features/regex-fail.feature" with content:
      """
      Feature: File regex fail
        Scenario: File does not match expected regex
          Given I am in a new temporary directory
          And I create the file "plain.txt" containing "hello world"
          Then the file "plain.txt" content should match "^\d+$"
      """
    When I run lobster "run --features features/regex-fail.feature --ci"
    Then the exit code should be 1

  # ── file content exact equality ───────────────────────────────────────────

  Scenario: File content should equal passes for exact content match
    Given I am in a new temporary directory
    And I create the file "exact.txt" with content:
      """
      line one
      line two
      """
    Then the file "exact.txt" content should equal:
      """
      line one
      line two
      """

  Scenario: File content should equal fails when content differs
    Given I am in a new temporary directory
    And I create the file "features/content-eq-fail.feature" with content:
      """
      Feature: Content equal fail
        Scenario: File content does not match expected
          Given I am in a new temporary directory
          And I create the file "f.txt" containing "actual content"
          Then the file "f.txt" content should equal:
            \"\"\"
            expected different content
            \"\"\"
      """
    When I run lobster "run --features features/content-eq-fail.feature --ci"
    Then the exit code should be 1

  # ── YAML validation ───────────────────────────────────────────────────────

  Scenario: File should contain valid YAML passes for well-formed YAML
    Given I am in a new temporary directory
    And I create the file "config.yaml" with content:
      """
      project: my-app
      version: 1.0.0
      services:
        - name: api
          port: 8080
        - name: db
          port: 5432
      """
    Then the file "config.yaml" should contain valid YAML

  Scenario: File should contain valid YAML passes for JSON (JSON is valid YAML)
    Given I am in a new temporary directory
    And I create the file "data.yaml" containing "{\"key\": \"value\"}"
    Then the file "data.yaml" should contain valid YAML

  Scenario: File should contain valid YAML fails for invalid YAML
    Given I am in a new temporary directory
    And I create the file "features/yaml-fail.feature" with content:
      """
      Feature: YAML fail
        Scenario: Invalid YAML fails validation
          Given I am in a new temporary directory
          And I create the file "bad.yaml" containing "key: [unclosed bracket"
          Then the file "bad.yaml" should contain valid YAML
      """
    When I run lobster "run --features features/yaml-fail.feature --ci"
    Then the exit code should be 1

  # ── JSON file field assertion ─────────────────────────────────────────────

  Scenario: JSON file field should equal passes for matching field value
    Given I am in a new temporary directory
    And I create the file "manifest.json" containing "{\"name\":\"my-service\",\"version\":\"2.0.0\"}"
    Then the JSON file "manifest.json" field "name" should equal "my-service"

  Scenario: JSON file field should equal passes for nested field
    Given I am in a new temporary directory
    And I create the file "app.json" with content:
      """
      {"server": {"host": "localhost", "port": 8080}}
      """
    Then the JSON file "app.json" field "server.host" should equal "localhost"

  Scenario: JSON file field should equal fails for wrong value
    Given I am in a new temporary directory
    And I create the file "features/json-field-fail.feature" with content:
      """
      Feature: JSON field fail
        Scenario: JSON field has wrong value
          Given I am in a new temporary directory
          And I create the file "m.json" containing "{\"env\":\"dev\"}"
          Then the JSON file "m.json" field "env" should equal "prod"
      """
    When I run lobster "run --features features/json-field-fail.feature --ci"
    Then the exit code should be 1

  # ── appending to files ────────────────────────────────────────────────────

  Scenario: I append to file adds content after existing content
    Given I am in a new temporary directory
    And I create the file "log.txt" containing "line one"
    When I append to file "log.txt" with content:
      """
      line two
      """
    Then the file "log.txt" should contain "line one"
    And the file "log.txt" should contain "line two"

  Scenario: I append to file creates the file if it does not exist
    Given I am in a new temporary directory
    When I append to file "new.txt" with content:
      """
      created by append
      """
    Then the file "new.txt" should exist
    And the file "new.txt" should contain "created by append"

  # ── deleting files ────────────────────────────────────────────────────────

  Scenario: I delete the file removes an existing file
    Given I am in a new temporary directory
    And I create the file "to-delete.txt" containing "temporary"
    When I delete the file "to-delete.txt"
    Then the file "to-delete.txt" should not exist

  Scenario: I delete the file is idempotent for non-existent files
    Given I am in a new temporary directory
    When I delete the file "does-not-exist.txt"
    Then the file "does-not-exist.txt" should not exist

  # ── directory membership ──────────────────────────────────────────────────

  Scenario: Directory should contain passes when the named entry exists
    Given I am in a new temporary directory
    And I create the file "mydir/README.md" containing "docs"
    Then the directory "mydir" should contain "README.md"

  Scenario: Directory should contain fails when the entry is absent
    Given I am in a new temporary directory
    And I create the file "features/dir-contains-fail.feature" with content:
      """
      Feature: Dir contains fail
        Scenario: Directory does not contain expected file
          Given I am in a new temporary directory
          And I run the command "mkdir empty-dir"
          Then the directory "empty-dir" should contain "missing.txt"
      """
    When I run lobster "run --features features/dir-contains-fail.feature --ci"
    Then the exit code should be 1

  Scenario: Directory should not contain passes when the entry is absent
    Given I am in a new temporary directory
    And I run the command "mkdir mydir"
    Then the directory "mydir" should not contain "absent.txt"

  Scenario: Directory should not contain fails when the entry exists
    Given I am in a new temporary directory
    And I create the file "features/dir-not-contains-fail.feature" with content:
      """
      Feature: Dir not contains fail
        Scenario: Directory contains file but should not
          Given I am in a new temporary directory
          And I create the file "d/present.txt" containing "x"
          Then the directory "d" should not contain "present.txt"
      """
    When I run lobster "run --features features/dir-not-contains-fail.feature --ci"
    Then the exit code should be 1

  # ── file size assertion ───────────────────────────────────────────────────

  Scenario: File size less than passes for a small file
    Given I am in a new temporary directory
    And I create the file "small.txt" containing "tiny"
    Then the file "small.txt" should have size less than 100 bytes

  Scenario: File size less than fails when file exceeds the limit
    Given I am in a new temporary directory
    And I create the file "features/size-fail.feature" with content:
      """
      Feature: File size fail
        Scenario: File exceeds size limit
          Given I am in a new temporary directory
          And I create the file "big.txt" containing "this file has more than ten bytes of content"
          Then the file "big.txt" should have size less than 5 bytes
      """
    When I run lobster "run --features features/size-fail.feature --ci"
    Then the exit code should be 1

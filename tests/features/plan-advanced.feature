Feature: lobster plan — advanced scenarios
  As a developer
  I want lobster plan to enumerate matching scenarios with full metadata
  So that I can review and export test execution plans

  # ── tag filtering ─────────────────────────────────────────────────────────

  Scenario: --tags with non-matching expression exits 0 with no match message
    Given I am in a new temporary directory
    And I create the file "features/tagged.feature" with content:
      """
      Feature: Tagged

        @smoke
        Scenario: Smoke test
          Given I set the base URL to "http://example.com"

        Scenario: Untagged test
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "plan --features features/tagged.feature --tags @nonexistent"
    Then the exit code should be 0
    And the output should contain "No scenarios matched"

  Scenario: --tags filters to only matching scenarios
    Given I am in a new temporary directory
    And I create the file "features/tagged.feature" with content:
      """
      Feature: Tagged

        @smoke
        Scenario: Smoke test
          Given I set the base URL to "http://example.com"

        Scenario: Untagged test
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "plan --features features/tagged.feature --tags @smoke"
    Then the exit code should be 0
    And the output should contain "Smoke test"
    And the output should not contain "Untagged test"

  # ── regex filtering ───────────────────────────────────────────────────────

  Scenario: --scenario-regex filters scenarios matching the pattern
    Given I am in a new temporary directory
    And I create the file "features/regex.feature" with content:
      """
      Feature: Regex filter

        Scenario: Alpha check
          Given I set the base URL to "http://example.com"

        Scenario: Beta check
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "plan --features features/regex.feature --scenario-regex Alpha"
    Then the exit code should be 0
    And the output should contain "Alpha check"

  # ── no glob match ─────────────────────────────────────────────────────────

  Scenario: Glob matching no files exits 0 with no-match message
    Given I am in a new temporary directory
    When I run lobster "plan --features features/nowhere/*.feature"
    Then the exit code should be 0
    And the output should contain "No scenarios matched"

  # ── JSON output ───────────────────────────────────────────────────────────

  Scenario: --format json outputs valid JSON with required fields
    Given I am in a new temporary directory
    And I create the file "features/simple.feature" with content:
      """
      Feature: Simple

        Scenario: One step
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "plan --features features/simple.feature --format json"
    Then the exit code should be 0
    And the output should be valid JSON
    And the output should contain "plan_id"
    And the output should contain "scenarios"
    And the output should contain "total"

  Scenario: JSON plan contains scenario name and feature
    Given I am in a new temporary directory
    And I create the file "features/named.feature" with content:
      """
      Feature: Named Feature

        Scenario: My Named Scenario
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "plan --features features/named.feature --format json"
    Then the exit code should be 0
    And the output should be valid JSON
    And the output should contain "My Named Scenario"
    And the output should contain "Named Feature"

  # ── --out flag ────────────────────────────────────────────────────────────

  Scenario: --out writes plan file to disk
    Given I am in a new temporary directory
    And I create the file "features/export.feature" with content:
      """
      Feature: Export

        Scenario: Exportable scenario
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "plan --features features/export.feature --out plan.json"
    Then the exit code should be 0
    And the file "plan.json" should exist

  Scenario: Plan file written by --out contains valid JSON
    Given I am in a new temporary directory
    And I create the file "features/export.feature" with content:
      """
      Feature: Export

        Scenario: Exportable scenario
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "plan --features features/export.feature --out plan.json"
    Then the exit code should be 0
    And the file "plan.json" should exist
    And the file "plan.json" should contain "plan_id"
    And the file "plan.json" should contain "Exportable scenario"

  # ── tags in JSON output ───────────────────────────────────────────────────

  Scenario: Tags appear in JSON plan output
    Given I am in a new temporary directory
    And I create the file "features/withtagged.feature" with content:
      """
      Feature: With tags

        @smoke @critical
        Scenario: Tagged scenario
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "plan --features features/withtagged.feature --format json"
    Then the exit code should be 0
    And the output should be valid JSON
    And the output should contain "smoke"
    And the output should contain "critical"

  # ── multiple scenarios count ──────────────────────────────────────────────

  Scenario: Total count matches number of scenarios in the feature
    Given I am in a new temporary directory
    And I create the file "features/multi.feature" with content:
      """
      Feature: Multi

        Scenario: First
          Given I set the base URL to "http://example.com"

        Scenario: Second
          Given I set the base URL to "http://example.com"

        Scenario: Third
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "plan --features features/multi.feature --format json"
    Then the exit code should be 0
    And the output should be valid JSON
    And the output should contain "\"total\": 3"

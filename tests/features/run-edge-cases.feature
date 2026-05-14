@covers:cli:run
Feature: run command edge cases
  As a developer
  I want the run command to handle edge cases and unusual inputs correctly
  So that tests work across varied configurations and environments

  # ── Environment variable injection ────────────────────────────────────────

  Scenario: run with --env injects variable into scenario context
    Given I am in a new temporary directory
    And I create the file "features/env.feature" with content:
      """
      Feature: Env injection
        Scenario: Reads env var
          When I run the command "echo $MY_VAR"
          Then the exit code should be 0
          And the output should contain "hello_from_env"
      """
    When I run lobster "run --features features/env.feature --env MY_VAR=hello_from_env --ci"
    Then the exit code should be 0

  Scenario: run with multiple --env flags injects all variables
    Given I am in a new temporary directory
    And I create the file "features/multienv.feature" with content:
      """
      Feature: Multi-env
        Scenario: Reads multiple env vars
          When I run the command "echo $A $B"
          Then the exit code should be 0
          And the output should contain "foo"
          And the output should contain "bar"
      """
    When I run lobster "run --features features/multienv.feature --env A=foo --env B=bar --ci"
    Then the exit code should be 0

  # ── Tag expressions ────────────────────────────────────────────────────────

  Scenario: run with tag expression @tag selects tagged scenarios
    Given I am in a new temporary directory
    And I create the file "features/tagged.feature" with content:
      """
      Feature: Tagged
        @run-me
        Scenario: Tagged pass
          When I run the command "echo tagged"
          Then the exit code should be 0

        @skip-me
        Scenario: Skipped
          When I run the command "echo skipped"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/tagged.feature --tags @run-me --ci"
    Then the exit code should be 0

  Scenario: run with tag expression excludes @skip scenarios
    Given I am in a new temporary directory
    And I create the file "features/skiptag.feature" with content:
      """
      Feature: Skip tag
        @skip
        Scenario: Should be skipped
          When I run the command "false"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/skiptag.feature --tags ~@skip --ci"
    Then the exit code should be 0

  Scenario: run with AND tag expression selects correctly
    Given I am in a new temporary directory
    And I create the file "features/andtag.feature" with content:
      """
      Feature: And tag
        @a @b
        Scenario: Has both
          When I run the command "echo both"
          Then the exit code should be 0

        @a
        Scenario: Has only a
          When I run the command "false"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/andtag.feature --tags \"@a @b\" --ci"
    Then the exit code should be 0

  # ── Scenario regex ────────────────────────────────────────────────────────

  Scenario: run with --scenario-regex selects matching scenarios
    Given I am in a new temporary directory
    And I create the file "features/regex.feature" with content:
      """
      Feature: Regex
        Scenario: match-me passes
          When I run the command "echo matched"
          Then the exit code should be 0

        Scenario: ignore-me would fail
          When I run the command "false"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/regex.feature --scenario-regex match-me --ci"
    Then the exit code should be 0

  # ── Unicode feature names ─────────────────────────────────────────────────

  Scenario: run handles unicode characters in feature file names
    Given I am in a new temporary directory
    And I create the file "features/unicode_test.feature" with content:
      """
      Feature: Unicode names ✓
        Scenario: Passes with unicode in name ✓
          When I run the command "echo unicode"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/unicode_test.feature --ci"
    Then the exit code should be 0

  # ── Background steps ──────────────────────────────────────────────────────

  Scenario: run executes Background before each scenario
    Given I am in a new temporary directory
    And I create the file "features/background.feature" with content:
      """
      Feature: Background
        Background:
          Given I am in a new temporary directory

        Scenario: First scenario with background
          When I run the command "echo bg1"
          Then the exit code should be 0

        Scenario: Second scenario with background
          When I run the command "echo bg2"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/background.feature --ci"
    Then the exit code should be 0

  # ── Multiple feature globs ────────────────────────────────────────────────

  Scenario: run accepts multiple --features globs
    Given I am in a new temporary directory
    And I create the file "featsA/a.feature" with content:
      """
      Feature: A
        Scenario: A passes
          When I run the command "echo A"
          Then the exit code should be 0
      """
    And I create the file "featsB/b.feature" with content:
      """
      Feature: B
        Scenario: B passes
          When I run the command "echo B"
          Then the exit code should be 0
      """
    When I run lobster "run --features featsA/*.feature --features featsB/*.feature --ci"
    Then the exit code should be 0

  # ── Feature with only Background steps ───────────────────────────────────

  Scenario: run with feature file that has Background but no scenarios exits 0
    Given I am in a new temporary directory
    And I create the file "features/bgonly.feature" with content:
      """
      Feature: Background only
        Background:
          Given I am in a new temporary directory
      """
    When I run lobster "run --features features/bgonly.feature --ci"
    Then the exit code should be 0

  # ── Duplicate scenario names ──────────────────────────────────────────────

  Scenario: run allows duplicate scenario names in the same feature
    Given I am in a new temporary directory
    And I create the file "features/dupnames.feature" with content:
      """
      Feature: Duplicate names
        Scenario: Passes
          When I run the command "echo first"
          Then the exit code should be 0

        Scenario: Passes
          When I run the command "echo second"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/dupnames.feature --ci"
    Then the exit code should be 0

  # ── Nonexistent feature glob ──────────────────────────────────────────────

  Scenario: run with glob that matches no files exits 0 with empty results
    Given I am in a new temporary directory
    When I run lobster "run --features nonexistent/**/*.feature --ci"
    Then the exit code should be 0

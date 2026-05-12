Feature: lobster run — persistence retention policy
  As a developer
  I want lobster to prune old run records after a run completes
  So that the local SQLite database does not grow unbounded over time

  # ── no retention config — no pruning ──────────────────────────────────────

  Scenario: Run completes successfully with no retention config set
    Given I am in a new temporary directory
    And I create the file "features/simple.feature" with content:
      """
      Feature: Retention check

        Scenario: Passes cleanly
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/simple.feature --ci"
    Then the exit code should be 0

  # ── retention via lobster.yaml ────────────────────────────────────────────

  Scenario: persistence.retention.max_runs is accepted in lobster.yaml
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: retention-test

      persistence:
        retention:
          max_runs: 100
      """
    And I create the file "features/simple.feature" with content:
      """
      Feature: Retention max runs

        Scenario: Runs fine with max_runs set
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/simple.feature --ci"
    Then the exit code should be 0

  Scenario: persistence.retention.max_age is accepted in lobster.yaml
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: retention-test

      persistence:
        retention:
          max_age: 720h
      """
    And I create the file "features/simple.feature" with content:
      """
      Feature: Retention max age

        Scenario: Runs fine with max_age set
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/simple.feature --ci"
    Then the exit code should be 0

  Scenario: Both max_runs and max_age can be configured together
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: retention-combined

      persistence:
        retention:
          max_runs: 50
          max_age: 168h
      """
    And I create the file "features/simple.feature" with content:
      """
      Feature: Combined retention

        Scenario: Runs fine with both limits set
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/simple.feature --ci"
    Then the exit code should be 0

  # ── zero / negative values are safe ──────────────────────────────────────

  Scenario: max_runs set to 0 (disabled) does not cause an error
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: retention-zero

      persistence:
        retention:
          max_runs: 0
      """
    And I create the file "features/simple.feature" with content:
      """
      Feature: Zero max runs

        Scenario: Runs fine when max_runs is 0
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/simple.feature --ci"
    Then the exit code should be 0

  # ── retention interacts with workspace ───────────────────────────────────

  Scenario: Retention config with explicit workspace flag succeeds
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: retention-workspace

      persistence:
        retention:
          max_runs: 10
      """
    And I create the file "features/simple.feature" with content:
      """
      Feature: Retention with workspace

        Scenario: Workspace-scoped retention succeeds
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/simple.feature --workspace my-ws --ci"
    Then the exit code should be 0

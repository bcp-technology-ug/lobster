@covers:cli:run
Feature: lobster run — quarantine tag support
  As a developer
  I want scenarios tagged with a quarantine marker to be demoted to skipped
  So that known-flaky tests do not break CI while still being tracked

  # ── quarantine disabled (default) ─────────────────────────────────────────

  Scenario: Failing scenario without quarantine config exits 1
    Given I am in a new temporary directory
    And I create the file "features/fail.feature" with content:
      """
      Feature: Fail

        Scenario: Intentional failure
          Given I set the base URL to "http://localhost:19999"
          When I send a GET request to "/ping"
          Then the response status should be 200
      """
    When I run lobster "run --features features/fail.feature --ci"
    Then the exit code should be 1

  # ── quarantine enabled — non-blocking (default blocking=false) ────────────

  Scenario: @quarantine tagged failing scenario is demoted to skipped
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: quarantine-test

      quarantine:
        enabled: true
        blocking_in_main_ci: false
      """
    And I create the file "features/quarantined.feature" with content:
      """
      Feature: Quarantined failures

        @quarantine
        Scenario: Flaky scenario that fails
          Given I set the base URL to "http://localhost:19999"
          When I send a GET request to "/ping"
          Then the response status should be 200
      """
    When I run lobster "run --features features/quarantined.feature --ci"
    Then the exit code should be 0
    And the output should contain "skipped"

  Scenario: Non-quarantined failing scenario still exits 1 when quarantine is enabled
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: quarantine-test

      quarantine:
        enabled: true
        blocking_in_main_ci: false
      """
    And I create the file "features/mixed.feature" with content:
      """
      Feature: Mixed

        Scenario: Normal failure
          Given I set the base URL to "http://localhost:19999"
          When I send a GET request to "/ping"
          Then the response status should be 200
      """
    When I run lobster "run --features features/mixed.feature --ci"
    Then the exit code should be 1

  # ── quarantine enabled — blocking=true ────────────────────────────────────

  Scenario: @quarantine with blocking_in_main_ci true still exits 1 on failure
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: quarantine-blocking-test

      quarantine:
        enabled: true
        blocking_in_main_ci: true
      """
    And I create the file "features/blocking.feature" with content:
      """
      Feature: Blocking quarantine

        @quarantine
        Scenario: Quarantined but blocking
          Given I set the base URL to "http://localhost:19999"
          When I send a GET request to "/ping"
          Then the response status should be 200
      """
    When I run lobster "run --features features/blocking.feature --ci"
    Then the exit code should be 1

  # ── custom quarantine tag ─────────────────────────────────────────────────

  Scenario: Custom quarantine tag demotes matching scenarios
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: custom-tag-test

      quarantine:
        enabled: true
        tag: known-flaky
        blocking_in_main_ci: false
      """
    And I create the file "features/custom.feature" with content:
      """
      Feature: Custom quarantine tag

        @known-flaky
        Scenario: Fails but is known flaky
          Given I set the base URL to "http://localhost:19999"
          When I send a GET request to "/ping"
          Then the response status should be 200
      """
    When I run lobster "run --features features/custom.feature --ci"
    Then the exit code should be 0
    And the output should contain "skipped"

  Scenario: Custom tag does not match the default @quarantine tag
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: custom-tag-mismatch-test

      quarantine:
        enabled: true
        tag: known-flaky
        blocking_in_main_ci: false
      """
    And I create the file "features/default-tag.feature" with content:
      """
      Feature: Default tag mismatch

        @quarantine
        Scenario: Tagged quarantine but wrong key
          Given I set the base URL to "http://localhost:19999"
          When I send a GET request to "/ping"
          Then the response status should be 200
      """
    When I run lobster "run --features features/default-tag.feature --ci"
    Then the exit code should be 1

  # ── passing quarantined scenario ──────────────────────────────────────────

  Scenario: @quarantine tagged passing scenario still exits 0
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: quarantine-pass-test

      quarantine:
        enabled: true
        blocking_in_main_ci: false
      """
    And I create the file "features/pass.feature" with content:
      """
      Feature: Quarantine pass

        @quarantine
        Scenario: This one passes
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/pass.feature --ci"
    Then the exit code should be 0

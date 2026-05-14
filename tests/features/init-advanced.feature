@covers:cli:init
Feature: lobster init — project initialisation
  As a developer
  I want lobster init to scaffold a complete project from flags
  So that I can start a new test project quickly and without manual configuration

  # ── basic --no-interactive ────────────────────────────────────────────────

  Scenario: --no-interactive with --project creates lobster.yaml
    Given I am in a new temporary directory
    When I run lobster "init . --project init-test --no-interactive"
    Then the exit code should be 0
    And the file "lobster.yaml" should exist
    And the file "lobster.yaml" should contain "init-test"

  Scenario: --no-interactive with --project creates features directory
    Given I am in a new temporary directory
    When I run lobster "init . --project init-test --no-interactive"
    Then the exit code should be 0
    And the directory "features" should exist

  Scenario: --no-interactive with --project creates hidden .lobster directory
    Given I am in a new temporary directory
    When I run lobster "init . --project init-test --no-interactive"
    Then the exit code should be 0
    And the directory ".lobster" should exist

  Scenario: --no-interactive without --project exits 2
    Given I am in a new temporary directory
    When I run lobster "init . --no-interactive"
    Then the exit code should be 2

  # ── --workspace flag ──────────────────────────────────────────────────────

  Scenario: --workspace writes workspace.selected into lobster.yaml
    Given I am in a new temporary directory
    When I run lobster "init . --project ws-test --workspace staging --no-interactive"
    Then the exit code should be 0
    And the file "lobster.yaml" should exist
    And the file "lobster.yaml" should contain "staging"

  # ── --features flag ───────────────────────────────────────────────────────

  Scenario: --features overrides the default glob in lobster.yaml
    Given I am in a new temporary directory
    When I run lobster "init . --project feat-test --features specs/**/*.feature --no-interactive"
    Then the exit code should be 0
    And the file "lobster.yaml" should exist
    And the file "lobster.yaml" should contain "specs/**/*.feature"

  # ── --compose flag ────────────────────────────────────────────────────────

  Scenario: --compose includes docker-compose file in lobster.yaml
    Given I am in a new temporary directory
    When I run lobster "init . --project compose-test --compose docker-compose.yml --no-interactive"
    Then the exit code should be 0
    And the file "lobster.yaml" should exist
    And the file "lobster.yaml" should contain "docker-compose.yml"

  # ── all flags together ────────────────────────────────────────────────────

  Scenario: All flags together produce a fully-populated lobster.yaml
    Given I am in a new temporary directory
    When I run lobster "init . --project fulltest --workspace prod --features e2e/**/*.feature --compose docker-compose.yml --no-interactive"
    Then the exit code should be 0
    And the file "lobster.yaml" should exist
    And the file "lobster.yaml" should contain "fulltest"
    And the file "lobster.yaml" should contain "prod"
    And the file "lobster.yaml" should contain "e2e/**/*.feature"
    And the file "lobster.yaml" should contain "docker-compose.yml"

  # ── init into a subdirectory ──────────────────────────────────────────────

  Scenario: Init into a named subdirectory creates lobster.yaml inside it
    Given I am in a new temporary directory
    When I run lobster "init myproject --project subdir-test --no-interactive"
    Then the exit code should be 0
    And the file "myproject/lobster.yaml" should exist
    And the file "myproject/lobster.yaml" should contain "subdir-test"

  # ── idempotency guard ─────────────────────────────────────────────────────

  Scenario: Init in a directory that already has lobster.yaml exits 1
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" containing "project: existing"
    When I run lobster "init . --project newproject --no-interactive"
    Then the exit code should be 1
    And the stderr should contain "already exists"

  # ── success output ────────────────────────────────────────────────────────

  Scenario: Successful init displays a tree of created files
    Given I am in a new temporary directory
    When I run lobster "init . --project tree-test --no-interactive"
    Then the exit code should be 0
    And the output should contain "lobster.yaml"
    And the output should contain "features"

  # ── example feature file ─────────────────────────────────────────────────

  Scenario: Init creates an example feature file inside features/
    Given I am in a new temporary directory
    When I run lobster "init . --project example-test --no-interactive"
    Then the exit code should be 0
    And the file "features/example.feature" should exist

  # ── stdin fallback ────────────────────────────────────────────────────────

  Scenario: Stdin provides project name when not using --no-interactive
    Given I am in a new temporary directory
    When I run the command "printf 'stdin-project\n\n\n\n' | lobster init ."
    Then the exit code should be 0
    And the file "lobster.yaml" should exist
    And the file "lobster.yaml" should contain "stdin-project"

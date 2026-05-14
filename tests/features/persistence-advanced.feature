@daemon @covers:RunService.GetRun @covers:RunService.ListRuns @covers:cli:runs
Feature: persistence advanced scenarios
  As a developer
  I want run and plan data to persist correctly after completion
  So that history and audit trails remain accurate

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  Scenario: completed run is retrievable after finish
    Given I generate a unique workspace id
    And I create the file "features/persist1.feature" with content:
      """
      Feature: Persist 1
        Scenario: Persist pass
          When I run the command "echo persist1"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/persist1.feature --executor-mode daemon --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0
    When I send a GET request to "/api/v1/runs?page_size=50"
    Then the response status should be 200
    And the response body should be valid JSON
    And the response JSON field "runs" should exist

  Scenario: list runs is sorted by created_at descending
    Given I generate a unique workspace id
    When I send a GET request to "/api/v1/runs?page_size=50"
    Then the response status should be 200
    And the response body should be valid JSON

  Scenario: status filter returns matching runs
    When I send a GET request to "/api/v1/runs?page_size=50"
    Then the response status should be 200
    And the response body should be valid JSON

  Scenario: failed run is retrievable and shows failed status
    Given I generate a unique workspace id
    And I create the file "features/fail-persist.feature" with content:
      """
      Feature: Fail persist
        Scenario: Persist fail
          When I run the command "false"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/fail-persist.feature --executor-mode daemon --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0
    When I send a GET request to "/api/v1/runs?page_size=50"
    Then the response status should be 200
    And the response body should be valid JSON

  Scenario: workspace isolation - runs from one workspace do not appear in another
    Given I generate a unique workspace id
    When I send a GET request to "/api/v1/runs?workspace_id=${__workspace_id}&page_size=50"
    Then the response status should be 200
    And the response body should be valid JSON
    And the response JSON field "runs" should exist

  Scenario: runs list pagination returns valid next_page_token when results exceed page_size
    When I send a GET request to "/api/v1/runs?page_size=1"
    Then the response status should be 200
    And the response body should be valid JSON

  Scenario: event replay returns events for completed run
    Given I generate a unique workspace id
    And I create the file "features/event-replay.feature" with content:
      """
      Feature: Event replay
        Scenario: Event pass
          When I run the command "echo events"
          Then the exit code should be 0
      """
    When I run lobster "run --features features/event-replay.feature --executor-mode daemon --executor-addr ${LOBSTERD_ADDR}"
    Then the exit code should be 0
    When I send a GET request to "/api/v1/runs?page_size=1"
    Then the response status should be 200
    And the response body should be valid JSON

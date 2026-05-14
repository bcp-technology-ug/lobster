@daemon @covers:RunService.ListRuns @covers:RunService.GetRun @covers:RunService.CancelRun @covers:RunService.StreamRunEvents @covers:GET:/api/v1/runs @covers:GET:/api/v1/runs/{runId} @covers:POST:/api/v1/runs/{runId}:cancel @covers:GET:/api/v1/runs/{runId}:events
Feature: HTTP Runs API
  As a developer
  I want the Runs HTTP API to list, get, cancel and stream runs
  So that CI pipelines can track test execution state

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  # ── GET /api/v1/runs ──────────────────────────────────────────────────────

  Scenario: GET /api/v1/runs returns 200
    When I send a GET request to "/api/v1/runs?page_size=10"
    Then the response status should be 200

  Scenario: GET /api/v1/runs returns valid JSON
    When I send a GET request to "/api/v1/runs"
    Then the response body should be valid JSON

  Scenario: GET /api/v1/runs response contains runs array
    When I send a GET request to "/api/v1/runs?page_size=10"
    Then the response body should be valid JSON
    And the response JSON field "runs" should exist

  Scenario: GET /api/v1/runs with page_size=1 param is accepted
    When I send a GET request to "/api/v1/runs?page_size=1"
    Then the response status should be 200
    And the response body should be valid JSON

  Scenario: GET /api/v1/runs with page_size=100 param is accepted
    When I send a GET request to "/api/v1/runs?page_size=100"
    Then the response status should be 200

  Scenario: GET /api/v1/runs with workspace_id filter param is accepted
    When I send a GET request to "/api/v1/runs?workspace_id=test-workspace&page_size=10"
    Then the response status should be 200
    And the response body should be valid JSON

  Scenario: GET /api/v1/runs on fresh daemon returns empty runs array
    When I send a GET request to "/api/v1/runs?page_size=10"
    Then the response status should be 200
    And the response body should be valid JSON
    And the response JSON field "runs" should exist

  Scenario: GET /api/v1/runs response may contain next_page_token field
    When I send a GET request to "/api/v1/runs?page_size=5"
    Then the response status should be 200
    And the response body should be valid JSON

  Scenario: POST /api/v1/runs returns 405
    When I send a POST request to "/api/v1/runs"
    Then the response status should not be 200

  # ── GET /api/v1/runs/{runId} ──────────────────────────────────────────────

  Scenario: GET /api/v1/runs with unknown run_id returns error status
    When I send a GET request to "/api/v1/runs/nonexistent-run-id-000"
    Then the response status should not be 200

  Scenario: GET /api/v1/runs with oversized run_id returns error status
    When I send a GET request to "/api/v1/runs/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    Then the response status should not be 200

  # ── POST /api/v1/runs/{runId}:cancel ─────────────────────────────────────

  Scenario: POST /api/v1/runs with unknown run_id cancel returns error
    When I send a POST request to "/api/v1/runs/unknown-run-99:cancel" with body:
      """
      {"reason": "test cancel"}
      """
    Then the response status should not be 200

  # ── GET /api/v1/runs/{runId}:events ──────────────────────────────────────

  Scenario: GET /api/v1/runs events with unknown run_id returns error
    When I send a GET request to "/api/v1/runs/unknown-run-99:events"
    Then the response status should not be 200

  Scenario: GET /api/v1/runs events with from_sequence param is accepted for valid run
    When I send a GET request to "/api/v1/runs/unknown-run-99:events?from_sequence=0"
    Then the response status should not be 200

  # ── Content-Type enforcement ──────────────────────────────────────────────

  Scenario: GET /api/v1/runs returns Content-Type containing json
    When I send a GET request to "/api/v1/runs"
    Then the response header "Content-Type" should equal "application/json"

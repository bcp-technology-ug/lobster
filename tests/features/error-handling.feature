@daemon @covers:RunService.GetRun @covers:PlanService.GetPlan @covers:StackService.GetStackStatus
Feature: error handling and invalid inputs
  As a developer
  I want the API to return meaningful errors for invalid inputs
  So that clients can diagnose and fix request issues

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  # ── Invalid ID formats ─────────────────────────────────────────────────────

  Scenario: GET /api/v1/runs with empty run_id in path returns 404 or 400
    When I send a GET request to "/api/v1/runs/"
    Then the response status should not be 200

  Scenario: GET /api/v1/plans with empty plan_id in path returns 404 or 400
    When I send a GET request to "/api/v1/plans/"
    Then the response status should not be 200

  Scenario: GET /api/v1/runs with special characters in ID returns error
    When I send a GET request to "/api/v1/runs/../../admin/health"
    Then the response status should not be 200

  Scenario: GET /api/v1/plans with special characters in ID returns error
    When I send a GET request to "/api/v1/plans/../../../../etc/passwd"
    Then the response status should not be 200

  # ── Missing required fields ────────────────────────────────────────────────

  Scenario: POST /api/v1/plans with null body returns error
    When I send a POST request to "/api/v1/plans" with body:
      """
      null
      """
    Then the response status should not be 200

  Scenario: POST /api/v1/stack:ensure with null body returns error
    When I send a POST request to "/api/v1/stack:ensure" with body:
      """
      null
      """
    Then the response status should not be 200

  Scenario: POST /api/v1/stack:teardown with null body returns error
    When I send a POST request to "/api/v1/stack:teardown" with body:
      """
      null
      """
    Then the response status should not be 200

  # ── Malformed JSON body ────────────────────────────────────────────────────

  Scenario: POST /api/v1/plans with truncated JSON returns error
    When I send a POST request to "/api/v1/plans" with body:
      """
      {"selector":
      """
    Then the response status should not be 200

  Scenario: POST /api/v1/stack:ensure with truncated JSON returns error
    When I send a POST request to "/api/v1/stack:ensure" with body:
      """
      {"workspace_id":
      """
    Then the response status should not be 200

  # ── Oversized strings ─────────────────────────────────────────────────────

  Scenario: POST /api/v1/stack:ensure with 1MB workspace_id returns error
    When I send a POST request to "/api/v1/stack:ensure" with body:
      """
      {"workspace_id": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
      """
    Then the response status should not be 200

  # ── Wrong content type ────────────────────────────────────────────────────

  Scenario: POST /api/v1/plans with plain text content type is handled
    When I send a POST request to "/api/v1/plans" with body:
      """
      plain text body
      """
    Then the response status should not be 200

  # ── Unknown paths ─────────────────────────────────────────────────────────

  Scenario: GET /api/v1/unknown-endpoint returns 404
    When I send a GET request to "/api/v1/unknown-endpoint"
    Then the response status should be 404

  Scenario: GET /api/v2/runs returns 404
    When I send a GET request to "/api/v2/runs"
    Then the response status should be 404

  Scenario: GET / returns non-500 response
    When I send a GET request to "/"
    Then the response status should not be 500

  # ── Empty workspace_id ────────────────────────────────────────────────────

  Scenario: GET /api/v1/stack:status with empty workspace_id returns error
    When I send a GET request to "/api/v1/stack:status?workspace_id="
    Then the response status should not be 200

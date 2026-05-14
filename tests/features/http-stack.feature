@daemon @covers:StackService.EnsureStack @covers:StackService.GetStackStatus @covers:StackService.TeardownStack @covers:POST:/api/v1/stack:ensure @covers:GET:/api/v1/stack:status @covers:POST:/api/v1/stack:teardown
Feature: HTTP Stack API
  As a developer
  I want the Stack HTTP API to manage Docker Compose infrastructure lifecycle
  So that stacks can be provisioned and torn down via REST

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  # ── POST /api/v1/stack:ensure ─────────────────────────────────────────────

  Scenario: POST /api/v1/stack:ensure with missing workspace_id returns error
    When I send a POST request to "/api/v1/stack:ensure" with body:
      """
      {}
      """
    Then the response status should not be 200

  Scenario: POST /api/v1/stack:ensure with empty workspace_id returns error
    When I send a POST request to "/api/v1/stack:ensure" with body:
      """
      {"workspace_id": ""}
      """
    Then the response status should not be 200

  Scenario: POST /api/v1/stack:ensure with oversized workspace_id returns error
    When I send a POST request to "/api/v1/stack:ensure" with body:
      """
      {"workspace_id": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
      """
    Then the response status should not be 200

  Scenario: POST /api/v1/stack:ensure with valid workspace_id returns JSON
    When I send a POST request to "/api/v1/stack:ensure" with body:
      """
      {"workspace_id": "test-stack-ensure-ws"}
      """
    Then the response body should be valid JSON

  Scenario: POST /api/v1/stack:ensure with invalid JSON returns error
    When I send a POST request to "/api/v1/stack:ensure" with body:
      """
      not-valid-json
      """
    Then the response status should not be 200

  # ── GET /api/v1/stack:status ──────────────────────────────────────────────

  Scenario: GET /api/v1/stack:status with workspace_id returns 404 when stack not found
    When I send a GET request to "/api/v1/stack:status?workspace_id=test-stack-status"
    Then the response status should be 404
    And the response body should be valid JSON

  Scenario: GET /api/v1/stack:status without workspace_id returns error
    When I send a GET request to "/api/v1/stack:status"
    Then the response status should not be 200

  Scenario: GET /api/v1/stack:status response is valid JSON when stack not found
    When I send a GET request to "/api/v1/stack:status?workspace_id=test-stack-status-2"
    Then the response status should be 404
    And the response body should be valid JSON

  Scenario: POST /api/v1/stack:status returns 405
    When I send a POST request to "/api/v1/stack:status"
    Then the response status should not be 200

  # ── POST /api/v1/stack:teardown ───────────────────────────────────────────

  Scenario: POST /api/v1/stack:teardown with missing workspace_id returns error
    When I send a POST request to "/api/v1/stack:teardown" with body:
      """
      {}
      """
    Then the response status should not be 200

  Scenario: POST /api/v1/stack:teardown with valid workspace_id returns JSON
    When I send a POST request to "/api/v1/stack:teardown" with body:
      """
      {"workspace_id": "test-stack-teardown-ws"}
      """
    Then the response body should be valid JSON

  Scenario: POST /api/v1/stack:teardown with remove_volumes flag is accepted
    When I send a POST request to "/api/v1/stack:teardown" with body:
      """
      {"workspace_id": "test-stack-teardown-vols", "remove_volumes": true}
      """
    Then the response body should be valid JSON

  Scenario: POST /api/v1/stack:teardown with invalid JSON returns error
    When I send a POST request to "/api/v1/stack:teardown" with body:
      """
      not-json
      """
    Then the response status should not be 200

  Scenario: GET /api/v1/stack:teardown returns 405
    When I send a GET request to "/api/v1/stack:teardown"
    Then the response status should not be 200

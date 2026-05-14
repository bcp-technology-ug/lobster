@daemon @covers:PlanService.Plan @covers:PlanService.GetPlan @covers:PlanService.ListPlans @covers:POST:/api/v1/plans @covers:GET:/api/v1/plans @covers:GET:/api/v1/plans/{planId}
Feature: HTTP Plans API
  As a developer
  I want the Plans HTTP API to create, list and retrieve execution plans
  So that I can review what will run before executing

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  # ── GET /api/v1/plans ────────────────────────────────────────────────────

  Scenario: GET /api/v1/plans returns 200
    When I send a GET request to "/api/v1/plans?page_size=10"
    Then the response status should be 200

  Scenario: GET /api/v1/plans returns valid JSON
    When I send a GET request to "/api/v1/plans"
    Then the response body should be valid JSON

  Scenario: GET /api/v1/plans response contains plans array
    When I send a GET request to "/api/v1/plans?page_size=10"
    Then the response body should be valid JSON
    And the response JSON field "plans" should exist

  Scenario: GET /api/v1/plans with page_size param is accepted
    When I send a GET request to "/api/v1/plans?page_size=10"
    Then the response status should be 200
    And the response body should be valid JSON

  Scenario: GET /api/v1/plans with workspace_id filter param is accepted
    When I send a GET request to "/api/v1/plans?workspace_id=test-ws&page_size=10"
    Then the response status should be 200
    And the response body should be valid JSON

  Scenario: GET /api/v1/plans response may contain next_page_token
    When I send a GET request to "/api/v1/plans?page_size=5"
    Then the response status should be 200
    And the response body should be valid JSON

  # ── POST /api/v1/plans ───────────────────────────────────────────────────

  Scenario: POST /api/v1/plans with missing selector returns error
    When I send a POST request to "/api/v1/plans" with body:
      """
      {}
      """
    Then the response status should not be 200

  Scenario: POST /api/v1/plans with empty workspace_id returns error
    When I send a POST request to "/api/v1/plans" with body:
      """
      {"selector": {"workspace_id": ""}}
      """
    Then the response status should not be 200

  Scenario: POST /api/v1/plans with invalid JSON body returns error
    When I send a POST request to "/api/v1/plans" with body:
      """
      not-valid-json
      """
    Then the response status should not be 200

  Scenario: POST /api/v1/plans with valid selector returns plan structure
    When I send a POST request to "/api/v1/plans" with body:
      """
      {"selector": {"workspace_id": "test-plans-api"}}
      """
    Then the response status should be 200
    And the response body should be valid JSON
    And the response JSON field "plan" should exist

  Scenario: POST /api/v1/plans response plan contains plan_id
    When I send a POST request to "/api/v1/plans" with body:
      """
      {"selector": {"workspace_id": "test-plans-api-id"}}
      """
    Then the response status should be 200
    And the response body should be valid JSON
    And the response JSON field "plan.planId" should exist

  Scenario: POST /api/v1/plans response plan contains workspace_id
    When I send a POST request to "/api/v1/plans" with body:
      """
      {"selector": {"workspace_id": "test-plans-api-ws"}}
      """
    Then the response status should be 200
    And the response body should be valid JSON
    And the response JSON field "plan.workspaceId" should exist

  Scenario: POST /api/v1/plans response plan contains scenarios array
    When I send a POST request to "/api/v1/plans" with body:
      """
      {"selector": {"workspace_id": "test-plans-api-sc"}}
      """
    Then the response status should be 200
    And the response body should be valid JSON
    And the response JSON field "plan.scenarios" should exist

  Scenario: POST /api/v1/plans response plan contains created_at timestamp
    When I send a POST request to "/api/v1/plans" with body:
      """
      {"selector": {"workspace_id": "test-plans-api-ts"}}
      """
    Then the response status should be 200
    And the response body should be valid JSON
    And the response JSON field "plan.createdAt" should exist

  # ── GET /api/v1/plans/{planId} ───────────────────────────────────────────

  Scenario: GET /api/v1/plans with unknown plan_id returns error
    When I send a GET request to "/api/v1/plans/nonexistent-plan-00"
    Then the response status should not be 200

  Scenario: GET /api/v1/plans with oversized plan_id returns error
    When I send a GET request to "/api/v1/plans/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    Then the response status should not be 200

  # ── Wrong methods ─────────────────────────────────────────────────────────

  Scenario: DELETE /api/v1/plans returns 405
    When I send a DELETE request to "/api/v1/plans"
    Then the response status should not be 200

  Scenario: PUT /api/v1/plans returns 405
    When I send a PUT request to "/api/v1/plans"
    Then the response status should not be 200

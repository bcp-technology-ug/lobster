@daemon @covers:GET:/api/v1/admin/health @covers:AdminService.GetHealth
Feature: HTTP healthz endpoint
  As an operator
  I want the /healthz endpoint to report daemon liveness
  So that load balancers and readiness probes work correctly

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  Scenario: GET /healthz returns 200
    When I send a GET request to "/healthz"
    Then the response status should be 200

  Scenario: GET /healthz returns valid JSON
    When I send a GET request to "/healthz"
    Then the response body should be valid JSON

  Scenario: GET /healthz response body contains lobsterd
    When I send a GET request to "/healthz"
    Then the response body should contain "lobsterd"

  Scenario: GET /healthz response body contains ready
    When I send a GET request to "/healthz"
    Then the response body should contain "ready"

  Scenario: POST /healthz returns 405
    When I send a POST request to "/healthz"
    Then the response status should be 405

  Scenario: PUT /healthz returns 405
    When I send a PUT request to "/healthz"
    Then the response status should be 405

  Scenario: DELETE /healthz returns 405
    When I send a DELETE request to "/healthz"
    Then the response status should be 405

  Scenario: PATCH /healthz returns 405
    When I send a PATCH request to "/healthz"
    Then the response status should be 405

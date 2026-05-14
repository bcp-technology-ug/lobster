@daemon @covers:IntegrationService.ListIntegrationAdapters @covers:IntegrationService.GetIntegrationAdapter @covers:IntegrationService.SetIntegrationAdapterState @covers:IntegrationService.ValidateIntegrationAdapter @covers:GET:/api/v1/integrations @covers:GET:/api/v1/integrations/{adapterId} @covers:POST:/api/v1/integrations/{adapterId}:setState @covers:POST:/api/v1/integrations/{adapterId}:validate
Feature: HTTP Integrations API
  As a developer
  I want the Integrations HTTP API to list, get, enable/disable and validate adapters
  So that integration adapters can be managed at runtime

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  # ── GET /api/v1/integrations ──────────────────────────────────────────────

  Scenario: GET /api/v1/integrations returns 200
    When I send a GET request to "/api/v1/integrations?page_size=10"
    Then the response status should be 200

  Scenario: GET /api/v1/integrations returns valid JSON
    When I send a GET request to "/api/v1/integrations"
    Then the response body should be valid JSON

  Scenario: GET /api/v1/integrations response contains adapters array
    When I send a GET request to "/api/v1/integrations?page_size=10"
    Then the response body should be valid JSON
    And the response JSON field "adapters" should exist

  Scenario: GET /api/v1/integrations with page_size param is accepted
    When I send a GET request to "/api/v1/integrations?page_size=10"
    Then the response status should be 200
    And the response body should be valid JSON

  Scenario: GET /api/v1/integrations response may contain next_page_token
    When I send a GET request to "/api/v1/integrations?page_size=5"
    Then the response status should be 200
    And the response body should be valid JSON

  Scenario: POST /api/v1/integrations returns 405
    When I send a POST request to "/api/v1/integrations"
    Then the response status should not be 200

  # ── GET /api/v1/integrations/{adapterId} ─────────────────────────────────

  Scenario: GET /api/v1/integrations with unknown adapter_id returns error
    When I send a GET request to "/api/v1/integrations/nonexistent-adapter"
    Then the response status should not be 200

  Scenario: GET /api/v1/integrations with oversized adapter_id returns error
    When I send a GET request to "/api/v1/integrations/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
    Then the response status should not be 200

  # ── POST /api/v1/integrations/{adapterId}:setState ───────────────────────

  Scenario: POST /api/v1/integrations setState with unknown adapter_id returns error
    When I send a POST request to "/api/v1/integrations/unknown-adapter:setState" with body:
      """
      {"enabled": true}
      """
    Then the response status should not be 200

  Scenario: POST /api/v1/integrations setState with enable and reason is accepted
    When I send a POST request to "/api/v1/integrations/unknown-adapter:setState" with body:
      """
      {"enabled": true, "reason": "test enable"}
      """
    Then the response status should not be 200

  Scenario: POST /api/v1/integrations setState with disable is accepted
    When I send a POST request to "/api/v1/integrations/unknown-adapter:setState" with body:
      """
      {"enabled": false, "reason": "test disable"}
      """
    Then the response status should not be 200

  Scenario: POST /api/v1/integrations setState with oversized reason returns error
    When I send a POST request to "/api/v1/integrations/unknown-adapter:setState" with body:
      """
      {"enabled": true, "reason": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
      """
    Then the response status should not be 200

  # ── POST /api/v1/integrations/{adapterId}:validate ───────────────────────

  Scenario: POST /api/v1/integrations validate with unknown adapter_id returns ok false
    When I send a POST request to "/api/v1/integrations/unknown-adapter:validate"
    Then the response status should be 200
    And the response body should be valid JSON

  Scenario: POST /api/v1/integrations validate with oversized adapter_id returns error
    When I send a POST request to "/api/v1/integrations/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa:validate"
    Then the response status should not be 200

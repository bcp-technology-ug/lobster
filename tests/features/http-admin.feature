@daemon @covers:AdminService.GetHealth @covers:AdminService.GetCapabilities @covers:AdminService.GetConfigSummary @covers:GET:/api/v1/admin/health @covers:GET:/api/v1/admin/capabilities @covers:GET:/api/v1/admin/config-summary
Feature: HTTP Admin API
  As a developer
  I want the Admin HTTP API to expose health, capabilities, and config
  So that operators can inspect daemon state without needing a gRPC client

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  # ── GET /api/v1/admin/health ──────────────────────────────────────────────

  Scenario: GET /api/v1/admin/health returns 200
    When I send a GET request to "/api/v1/admin/health"
    Then the response status should be 200

  Scenario: GET /api/v1/admin/health returns valid JSON
    When I send a GET request to "/api/v1/admin/health"
    Then the response body should be valid JSON

  Scenario: GET /api/v1/admin/health response contains version field
    When I send a GET request to "/api/v1/admin/health"
    Then the response body should be valid JSON
    And the response JSON field "health.version" should exist

  Scenario: GET /api/v1/admin/health response contains live field
    When I send a GET request to "/api/v1/admin/health"
    Then the response body should be valid JSON
    And the response JSON field "health.live" should exist

  Scenario: GET /api/v1/admin/health response contains ready field
    When I send a GET request to "/api/v1/admin/health"
    Then the response body should be valid JSON
    And the response JSON field "health.ready" should exist

  Scenario: GET /api/v1/admin/health response contains observed_at timestamp
    When I send a GET request to "/api/v1/admin/health"
    Then the response body should be valid JSON
    And the response JSON field "health.observedAt" should exist

  Scenario: POST /api/v1/admin/health returns 405
    When I send a POST request to "/api/v1/admin/health"
    Then the response status should not be 200

  Scenario: PUT /api/v1/admin/health returns 405
    When I send a PUT request to "/api/v1/admin/health"
    Then the response status should not be 200

  # ── GET /api/v1/admin/capabilities ───────────────────────────────────────

  Scenario: GET /api/v1/admin/capabilities returns 200
    When I send a GET request to "/api/v1/admin/capabilities"
    Then the response status should be 200

  Scenario: GET /api/v1/admin/capabilities returns valid JSON
    When I send a GET request to "/api/v1/admin/capabilities"
    Then the response body should be valid JSON

  Scenario: GET /api/v1/admin/capabilities response contains api_package
    When I send a GET request to "/api/v1/admin/capabilities"
    Then the response body should be valid JSON
    And the response JSON field "apiPackage" should exist

  Scenario: GET /api/v1/admin/capabilities response contains api_version
    When I send a GET request to "/api/v1/admin/capabilities"
    Then the response body should be valid JSON
    And the response JSON field "apiVersion" should exist

  Scenario: GET /api/v1/admin/capabilities response contains capabilities array
    When I send a GET request to "/api/v1/admin/capabilities"
    Then the response body should be valid JSON
    And the response JSON field "capabilities" should exist

  Scenario: POST /api/v1/admin/capabilities returns 405
    When I send a POST request to "/api/v1/admin/capabilities"
    Then the response status should not be 200

  # ── GET /api/v1/admin/config-summary ─────────────────────────────────────

  Scenario: GET /api/v1/admin/config-summary returns 200
    When I send a GET request to "/api/v1/admin/config-summary"
    Then the response status should be 200

  Scenario: GET /api/v1/admin/config-summary returns valid JSON
    When I send a GET request to "/api/v1/admin/config-summary"
    Then the response body should be valid JSON

  Scenario: GET /api/v1/admin/config-summary response contains config field
    When I send a GET request to "/api/v1/admin/config-summary"
    Then the response body should be valid JSON
    And the response JSON field "config" should exist

  Scenario: GET /api/v1/admin/config-summary response contains workspace_id
    When I send a GET request to "/api/v1/admin/config-summary"
    Then the response body should be valid JSON
    And the response JSON field "config.workspaceId" should exist

  Scenario: POST /api/v1/admin/config-summary returns 405
    When I send a POST request to "/api/v1/admin/config-summary"
    Then the response status should not be 200

  # ── Unknown admin path ────────────────────────────────────────────────────

  Scenario: GET /api/v1/admin/nonexistent returns 404
    When I send a GET request to "/api/v1/admin/nonexistent"
    Then the response status should be 404

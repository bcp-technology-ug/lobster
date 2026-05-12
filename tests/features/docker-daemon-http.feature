@daemon
Feature: lobsterd — HTTP/JSON gateway
  As a developer
  I want the HTTP gateway to proxy the gRPC API correctly
  So that REST clients can interact with the daemon without a gRPC client

  # Requires a running lobsterd. Set LOBSTERD_HTTP_URL (HTTP base URL).
  # These are injected by `make test-all` via --env flags.

  Background:
    Given I am in a new temporary directory
    And I set the base URL to "${LOBSTERD_HTTP_URL}"
    And I wait up to 30s for URL "${LOBSTERD_HTTP_URL}/healthz" to be reachable

  # ── Health ────────────────────────────────────────────────────────────────

  Scenario: GET /healthz returns 200
    When I send a GET request to "/healthz"
    Then the response status should be 200

  Scenario: GET /healthz returns JSON with service and status fields
    When I send a GET request to "/healthz"
    Then the response body should be valid JSON
    And the response body should contain "lobsterd"
    And the response body should contain "ready"

  Scenario: Non-GET /healthz returns 405
    When I send a POST request to "/healthz"
    Then the response status should be 405

  # ── Admin API ─────────────────────────────────────────────────────────────

  Scenario: GET /api/v1/admin/config-summary returns 200
    When I send a GET request to "/api/v1/admin/config-summary"
    Then the response status should be 200
    And the response body should be valid JSON

  Scenario: GET /api/v1/admin/health returns 200 with version
    When I send a GET request to "/api/v1/admin/health"
    Then the response status should be 200
    And the response body should contain "version"

  # ── Run API ───────────────────────────────────────────────────────────────

  Scenario: GET /api/v1/runs returns 200 with empty list on a fresh daemon
    When I send a GET request to "/api/v1/runs?page_size=10"
    Then the response status should be 200
    And the response body should be valid JSON

  # ── Plan API ─────────────────────────────────────────────────────────────

  Scenario: GET /api/v1/plans returns 200 on a fresh daemon
    When I send a GET request to "/api/v1/plans?page_size=10"
    Then the response status should be 200
    And the response body should be valid JSON

  # ── Unknown route ─────────────────────────────────────────────────────────

  Scenario: Unknown API path returns 404
    When I send a GET request to "/api/v1/notaroute"
    Then the response status should be 404

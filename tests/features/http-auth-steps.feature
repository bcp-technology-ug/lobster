Feature: HTTP authentication and extended request steps
  As a test author
  I want bearer tokens, basic auth, form data, redirect control and response time assertions
  So that I can test authenticated APIs and complex HTTP interactions end-to-end
  #
  # All scenarios tagged @docker require the compose stack (httpbin on 19780,
  # fixture-server on 19790).  Run with: make test-all

  # ── bearer token ──────────────────────────────────────────────────────────

  @docker
  Scenario: I set the bearer token sends Authorization header to the server
    Given I am in a new temporary directory
    And I set the base URL to "${HTTPBIN_URL}"
    When I set the bearer token "my-secret-token"
    And I send a GET request to "/bearer"
    Then the response status should be 200

  @docker
  Scenario: Requests without a bearer token receive 401 from the server
    Given I am in a new temporary directory
    And I set the base URL to "${HTTPBIN_URL}"
    When I send a GET request to "/bearer"
    Then the response status should be 401

  # ── basic auth ────────────────────────────────────────────────────────────

  @docker
  Scenario: I set the basic auth sends correct Authorization header
    Given I am in a new temporary directory
    And I set the base URL to "${HTTPBIN_URL}"
    When I set the basic auth username "admin" and password "secret"
    And I send a GET request to "/basic-auth/admin/secret"
    Then the response status should be 200

  @docker
  Scenario: Requests with wrong basic auth credentials receive 401
    Given I am in a new temporary directory
    And I set the base URL to "${HTTPBIN_URL}"
    When I send a GET request to "/basic-auth/admin/secret"
    Then the response status should be 401

  # ── form data ─────────────────────────────────────────────────────────────

  @docker
  Scenario: I send a request with form data posts URL-encoded body
    Given I am in a new temporary directory
    And I set the base URL to "${HTTPBIN_URL}"
    When I send a POST request to "/post" with form data:
      | username | alice  |
      | password | s3cr3t |
    Then the response status should be 200
    And the response body should contain "alice"

  # ── response body regex ───────────────────────────────────────────────────

  @docker
  Scenario: Response body should match passes for matching regex
    Given I am in a new temporary directory
    And I set the base URL to "${FIXTURE_URL}"
    When I send a GET request to "/payload.json"
    Then the response body should match "\"name\":\s*\"alice-"

  @docker
  Scenario: Response body should not match passes when body does not match
    Given I am in a new temporary directory
    And I set the base URL to "${FIXTURE_URL}"
    When I send a GET request to "/payload.json"
    Then the response body should not match "\"status\":\s*\"error\""

  # ── response header regex ─────────────────────────────────────────────────

  @docker
  Scenario: Response header should match passes for matching regex
    Given I am in a new temporary directory
    And I set the base URL to "${HTTPBIN_URL}"
    When I send a GET request to "/response-headers?X-Version=v2.3.1-release"
    Then the response header "X-Version" should match "^v\d+\.\d+\.\d+"

  # ── response time assertion ───────────────────────────────────────────────

  @docker
  Scenario: Response time should be less than passes for a fast local server
    Given I am in a new temporary directory
    And I set the base URL to "${HTTPBIN_URL}"
    When I send a GET request to "/get"
    Then the response time should be less than 5000ms

  # ── redirect control ──────────────────────────────────────────────────────

  @docker
  Scenario: I do not follow redirects returns the 3xx response directly
    Given I am in a new temporary directory
    And I set the base URL to "${HTTPBIN_URL}"
    When I do not follow redirects
    And I send a GET request to "/redirect-to?url=/get&status_code=301"
    Then the response status should be 301

  @docker
  Scenario: I follow redirects restores default redirect-following behaviour
    Given I am in a new temporary directory
    And I set the base URL to "${HTTPBIN_URL}"
    When I do not follow redirects
    And I follow redirects
    And I send a GET request to "/redirect-to?url=/get&status_code=301"
    Then the response status should be 200

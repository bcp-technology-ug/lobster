Feature: TCP port and service step definitions
  As a test author
  I want to check TCP port availability and wait for ports to open
  So that I can gate test execution on infrastructure readiness
  #
  # @docker scenarios use the compose services (httpbin on 19780) as their
  # TCP target — no need to manage ephemeral servers in the test itself.

  # ── TCP port open assertion ───────────────────────────────────────────────

  @docker
  Scenario: TCP port should be open passes when a server is listening
    Given I am in a new temporary directory
    Then the TCP port "19780" on "127.0.0.1" should be open

  Scenario: TCP port should be open fails for a closed port
    Given I am in a new temporary directory
    And I create the file "features/tcp-fail.feature" with content:
      """
      Feature: TCP fail
        Scenario: Closed port fails assertion
          Given I am in a new temporary directory
          Then the TCP port "19899" on "127.0.0.1" should be open
      """
    When I run lobster "run --features features/tcp-fail.feature --ci"
    Then the exit code should be 1

  # ── wait for TCP port ─────────────────────────────────────────────────────

  @docker
  Scenario: I wait for TCP port to be open succeeds when the port is already up
    Given I am in a new temporary directory
    When I wait up to 5s for TCP port "19780" on "127.0.0.1" to be open
    Then the TCP port "19780" on "127.0.0.1" should be open

  Scenario: I wait for TCP port times out when port never opens
    Given I am in a new temporary directory
    And I create the file "features/tcp-wait-fail.feature" with content:
      """
      Feature: TCP wait fail
        Scenario: Port never opens
          Given I am in a new temporary directory
          When I wait up to 2s for TCP port "19898" on "127.0.0.1" to be open
          Then the directory "." should exist
      """
    When I run lobster "run --features features/tcp-wait-fail.feature --ci"
    Then the exit code should be 1

  # ── combined HTTP + TCP check ─────────────────────────────────────────────

  @docker
  Scenario: HTTP server port is detectable as an open TCP port
    Given I am in a new temporary directory
    Then the TCP port "19790" on "127.0.0.1" should be open
    And I set the base URL to "${FIXTURE_URL}"
    When I send a GET request to "/token.json"
    Then the response status should be 200

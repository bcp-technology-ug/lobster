Feature: gRPC health check step definitions
  As a test author
  I want to verify gRPC health check steps
  So that I can gate test suites on real gRPC service availability

  # All positive (against a real server) scenarios require a running gRPC server,
  # so they are tagged @integration and skipped by the default lobster.yaml config.
  # The error-path tests run without any server and use the inner-lobster pattern.

  # ── error paths (no server needed) ───────────────────────────────────────

  Scenario: gRPC health check fails when no server is listening
    Given I am in a new temporary directory
    And I create the file "features/grpc-health-fail.feature" with content:
      """
      Feature: gRPC health fail
        Scenario: Server is not reachable
          Then the gRPC service at "localhost:19890" should be healthy
      """
    When I run lobster "run --features features/grpc-health-fail.feature --ci"
    Then the exit code should be 1

  Scenario: gRPC named service check fails when no server is listening
    Given I am in a new temporary directory
    And I create the file "features/grpc-named-fail.feature" with content:
      """
      Feature: gRPC named service fail
        Scenario: Named service not reachable
          Then the gRPC service at "localhost:19891" serving "my.Service" should be healthy
      """
    When I run lobster "run --features features/grpc-named-fail.feature --ci"
    Then the exit code should be 1

  Scenario: gRPC wait step times out when server never starts
    Given I am in a new temporary directory
    And I create the file "features/grpc-wait-fail.feature" with content:
      """
      Feature: gRPC wait fail
        Scenario: gRPC server never comes up
          When I wait up to 3s for the gRPC service at "localhost:19892" to be healthy
      """
    When I run lobster "run --features features/grpc-wait-fail.feature --ci"
    Then the exit code should be 1

  # ── integration tests (require a live gRPC health server) ────────────────

  @integration
  Scenario: gRPC service overall health check passes when server is healthy
    Then the gRPC service at "localhost:50051" should be healthy

  @integration
  Scenario: gRPC named service health check passes when service is healthy
    Then the gRPC service at "localhost:50051" serving "my.Service" should be healthy

  @integration
  Scenario: Wait for gRPC service succeeds when server is healthy
    When I wait up to 10s for the gRPC service at "localhost:50051" to be healthy
    Then the gRPC service at "localhost:50051" should be healthy

Feature: lobster run — OpenTelemetry tracing
  As a developer
  I want lobster run to emit traces when an OTel endpoint is configured
  So that I can observe test execution in a distributed tracing backend

  # ── no endpoint — noop tracing (default) ─────────────────────────────────

  Scenario: Run completes successfully without any OTel configuration
    Given I am in a new temporary directory
    And I create the file "features/simple.feature" with content:
      """
      Feature: OTel default

        Scenario: No tracing configured
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/simple.feature --ci"
    Then the exit code should be 0

  # ── --otel-endpoint flag ──────────────────────────────────────────────────

  Scenario: --otel-endpoint flag is accepted by lobster run
    Given I am in a new temporary directory
    And I create the file "features/simple.feature" with content:
      """
      Feature: OTel endpoint flag

        Scenario: Tracing flag accepted
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/simple.feature --otel-endpoint http://localhost:4318 --ci"
    Then the exit code should be 0

  Scenario: --otel-service-name flag is accepted alongside --otel-endpoint
    Given I am in a new temporary directory
    And I create the file "features/simple.feature" with content:
      """
      Feature: OTel service name

        Scenario: Custom service name accepted
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/simple.feature --otel-endpoint http://localhost:4318 --otel-service-name my-suite --ci"
    Then the exit code should be 0

  # ── unreachable endpoint — run still completes ────────────────────────────

  Scenario: Unreachable OTel endpoint does not fail the run
    Given I am in a new temporary directory
    And I create the file "features/simple.feature" with content:
      """
      Feature: OTel unreachable

        Scenario: Tracing failure is non-fatal
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/simple.feature --otel-endpoint http://localhost:19999 --ci"
    Then the exit code should be 0

  # ── config-file-based OTel ────────────────────────────────────────────────

  Scenario: telemetry.otel.endpoint in lobster.yaml is picked up
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: otel-config-test

      telemetry:
        otel:
          endpoint: "http://localhost:4318"
          service_name: "lobster-ci"
      """
    And I create the file "features/simple.feature" with content:
      """
      Feature: Config-based OTel

        Scenario: Endpoint from config
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/simple.feature --ci"
    Then the exit code should be 0

  Scenario: --otel-endpoint flag overrides telemetry.otel.endpoint in lobster.yaml
    Given I am in a new temporary directory
    And I create the file "lobster.yaml" with content:
      """
      project: otel-override-test

      telemetry:
        otel:
          endpoint: "http://config-endpoint:4318"
      """
    And I create the file "features/simple.feature" with content:
      """
      Feature: OTel flag override

        Scenario: Flag wins over config
          Given I set the base URL to "http://example.com"
      """
    When I run lobster "run --features features/simple.feature --otel-endpoint http://localhost:4318 --ci"
    Then the exit code should be 0

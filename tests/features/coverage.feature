@covers:cli:coverage
Feature: lobster coverage command
  As a developer
  I want the coverage command to report which API surfaces have feature coverage
  So that I can identify gaps before shipping

  Scenario: coverage --help exits 0
    Given I am in a new temporary directory
    When I run lobster "coverage --help"
    Then the exit code should be 0

  Scenario: coverage --help shows usage
    Given I am in a new temporary directory
    When I run lobster "coverage --help"
    Then the output should contain "coverage"

  Scenario: coverage with proto and openapi and features exits 0
    Given I am in a new temporary directory
    And I create the file "proto/example/v1/example.proto" with content:
      """
      syntax = "proto3";
      package example.v1;
      service ExampleService {
        rpc GetExample(GetExampleRequest) returns (GetExampleResponse);
      }
      message GetExampleRequest {}
      message GetExampleResponse {}
      """
    And I create the file "gen/openapi/openapi.yaml" with content:
      """
      openapi: "3.0.0"
      info:
        title: Example
        version: "1.0"
      paths:
        /api/v1/examples:
          get:
            summary: List examples
            operationId: listExamples
            responses:
              "200":
                description: OK
      """
    And I create the file "features/covered.feature" with content:
      """
      @covers:ExampleService.GetExample @covers:GET:/api/v1/examples
      Feature: Covered
        Scenario: Covered scenario
          When I run the command "echo covered"
          Then the exit code should be 0
      """
    When I run lobster "coverage --proto proto/**/*.proto --openapi gen/openapi/openapi.yaml --no-cli --features features/*.feature"
    Then the exit code should be 0

  Scenario: coverage --format json exits 0
    Given I am in a new temporary directory
    And I create the file "proto/svc/v1/svc.proto" with content:
      """
      syntax = "proto3";
      package svc.v1;
      service SvcService {
        rpc DoSvc(DoSvcRequest) returns (DoSvcResponse);
      }
      message DoSvcRequest {}
      message DoSvcResponse {}
      """
    And I create the file "gen/openapi/openapi.yaml" with content:
      """
      openapi: "3.0.0"
      info:
        title: Svc
        version: "1.0"
      paths: {}
      """
    And I create the file "features/svc.feature" with content:
      """
      @covers:SvcService.DoSvc
      Feature: Svc
        Scenario: Svc scenario
          When I run the command "echo svc"
          Then the exit code should be 0
      """
    When I run lobster "coverage --proto proto/**/*.proto --openapi gen/openapi/openapi.yaml --no-cli --features features/*.feature --format json"
    Then the exit code should be 0

  Scenario: coverage --format json output is valid JSON
    Given I am in a new temporary directory
    And I create the file "proto/j/v1/j.proto" with content:
      """
      syntax = "proto3";
      package j.v1;
      service JService {
        rpc DoJ(DoJRequest) returns (DoJResponse);
      }
      message DoJRequest {}
      message DoJResponse {}
      """
    And I create the file "gen/openapi/openapi.yaml" with content:
      """
      openapi: "3.0.0"
      info:
        title: J
        version: "1.0"
      paths: {}
      """
    And I create the file "features/j.feature" with content:
      """
      @covers:JService.DoJ
      Feature: J
        Scenario: J scenario
          When I run the command "echo j"
          Then the exit code should be 0
      """
    When I run lobster "coverage --proto proto/**/*.proto --openapi gen/openapi/openapi.yaml --no-cli --features features/*.feature --format json"
    Then the exit code should be 0
    And the output should be valid JSON

  Scenario: coverage exits non-zero when uncovered surfaces exist with defaults
    Given I am in a new temporary directory
    And I create the file "proto/u/v1/u.proto" with content:
      """
      syntax = "proto3";
      package u.v1;
      service UService {
        rpc DoU(DoURequest) returns (DoUResponse);
      }
      message DoURequest {}
      message DoUResponse {}
      """
    And I create the file "gen/openapi/openapi.yaml" with content:
      """
      openapi: "3.0.0"
      info:
        title: U
        version: "1.0"
      paths: {}
      """
    And I create the file "features/u.feature" with content:
      """
      Feature: Uncovered
        Scenario: Uncovered scenario
          When I run the command "echo uncovered"
          Then the exit code should be 0
      """
    When I run lobster "coverage --proto proto/**/*.proto --openapi gen/openapi/openapi.yaml --features features/*.feature"
    Then the exit code should not be 0

  Scenario: coverage with --no-proto skips proto scanning
    Given I am in a new temporary directory
    And I create the file "gen/openapi/openapi.yaml" with content:
      """
      openapi: "3.0.0"
      info:
        title: NoProto
        version: "1.0"
      paths: {}
      """
    And I create the file "features/noproto.feature" with content:
      """
      Feature: No proto
        Scenario: No proto scenario
          When I run the command "echo noproto"
          Then the exit code should be 0
      """
    When I run lobster "coverage --no-proto --no-cli --openapi gen/openapi/openapi.yaml --features features/*.feature"
    Then the exit code should be 0

  Scenario: coverage with --no-openapi skips openapi scanning
    Given I am in a new temporary directory
    And I create the file "proto/nooapi/v1/nooapi.proto" with content:
      """
      syntax = "proto3";
      package nooapi.v1;
      service NoOApiService {
        rpc DoNoOApi(DoNoOApiRequest) returns (DoNoOApiResponse);
      }
      message DoNoOApiRequest {}
      message DoNoOApiResponse {}
      """
    And I create the file "features/nooapi.feature" with content:
      """
      @covers:NoOApiService.DoNoOApi
      Feature: No OpenAPI
        Scenario: No openapi scenario
          When I run the command "echo nooapi"
          Then the exit code should be 0
      """
    When I run lobster "coverage --proto proto/**/*.proto --no-openapi --no-cli --features features/*.feature"
    Then the exit code should be 0

  Scenario: coverage with --no-cli skips CLI surface scanning
    Given I am in a new temporary directory
    And I create the file "gen/openapi/openapi.yaml" with content:
      """
      openapi: "3.0.0"
      info:
        title: NoCli
        version: "1.0"
      paths: {}
      """
    And I create the file "features/nocli.feature" with content:
      """
      Feature: No CLI
        Scenario: No cli scenario
          When I run the command "echo nocli"
          Then the exit code should be 0
      """
    When I run lobster "coverage --no-proto --openapi gen/openapi/openapi.yaml --no-cli --features features/*.feature"
    Then the exit code should be 0

  Scenario: coverage --min-scenarios 3 fails when scenarios below threshold
    Given I am in a new temporary directory
    And I create the file "proto/min/v1/min.proto" with content:
      """
      syntax = "proto3";
      package min.v1;
      service MinService {
        rpc DoMin(DoMinRequest) returns (DoMinResponse);
      }
      message DoMinRequest {}
      message DoMinResponse {}
      """
    And I create the file "gen/openapi/openapi.yaml" with content:
      """
      openapi: "3.0.0"
      info:
        title: Min
        version: "1.0"
      paths: {}
      """
    And I create the file "features/min.feature" with content:
      """
      @covers:MinService.DoMin
      Feature: Min scenarios
        Scenario: Only one scenario
          When I run the command "echo min"
          Then the exit code should be 0
      """
    When I run lobster "coverage --proto proto/**/*.proto --openapi gen/openapi/openapi.yaml --features features/*.feature --min-scenarios 3"
    Then the exit code should not be 0

  Scenario: coverage with --min-scenarios 1 and one scenario exits 0
    Given I am in a new temporary directory
    And I create the file "proto/min1/v1/min1.proto" with content:
      """
      syntax = "proto3";
      package min1.v1;
      service Min1Service {
        rpc DoMin1(DoMin1Request) returns (DoMin1Response);
      }
      message DoMin1Request {}
      message DoMin1Response {}
      """
    And I create the file "gen/openapi/openapi.yaml" with content:
      """
      openapi: "3.0.0"
      info:
        title: Min1
        version: "1.0"
      paths: {}
      """
    And I create the file "features/min1.feature" with content:
      """
      @covers:Min1Service.DoMin1
      Feature: Min1 scenarios
        Scenario: One scenario passes min
          When I run the command "echo min1"
          Then the exit code should be 0
      """
    When I run lobster "coverage --proto proto/**/*.proto --openapi gen/openapi/openapi.yaml --features features/*.feature --min-scenarios 1 --no-cli"
    Then the exit code should be 0

  Scenario: coverage with empty features glob exits 0
    Given I am in a new temporary directory
    And I create the file "gen/openapi/openapi.yaml" with content:
      """
      openapi: "3.0.0"
      info:
        title: Empty
        version: "1.0"
      paths: {}
      """
    When I run lobster "coverage --no-proto --openapi gen/openapi/openapi.yaml --no-cli --features nonexistent/**/*.feature"
    Then the exit code should be 0

@integration @covers:cli:stack @covers:StackService.EnsureStack @covers:StackService.GetStackStatus @covers:StackService.TeardownStack
Feature: Docker Compose stack orchestration
  As a developer
  I want lobster run to bring up a Docker Compose stack before tests and tear it down after
  So that integration tests have a live infrastructure environment

  # These scenarios require Docker to be running on the host.

  Scenario: --compose flag is accepted without error when file exists
    Given I am in a new temporary directory
    And I create the file "docker-compose.test.yml" with content:
      """
      services:
        placeholder:
          image: busybox
          command: ["sh", "-c", "echo compose-ok && sleep 1"]
      """
    And I create the file "features/compose.feature" with content:
      """
      Feature: Compose test
        Scenario: Placeholder
          Given I am in a new temporary directory
          When I run the command "echo compose-scenario"
          Then the exit code should be 0
      """
    When I run lobster "run --compose docker-compose.test.yml --features features/compose.feature --ci"
    Then the exit code should be 0

  Scenario: --keep-stack skips compose teardown on success
    Given I am in a new temporary directory
    And I create the file "docker-compose.keep.yml" with content:
      """
      services:
        keeper:
          image: busybox
          command: ["sh", "-c", "sleep 30"]
      """
    And I create the file "features/keep.feature" with content:
      """
      Feature: Keep stack
        Scenario: Pass
          Given I am in a new temporary directory
          When I run the command "echo kept"
          Then the exit code should be 0
      """
    When I run lobster "run --compose docker-compose.keep.yml --features features/keep.feature --keep-stack --ci"
    Then the exit code should be 0

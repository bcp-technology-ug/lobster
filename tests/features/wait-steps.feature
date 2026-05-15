Feature: Wait and retry step definitions
  As a test author
  I want wait and retry primitives to handle eventual consistency and async systems
  So that my tests can cope with services that take time to become ready

  # ── sleep steps ───────────────────────────────────────────────────────────

  Scenario: I wait N seconds pauses execution and allows subsequent steps to run
    Given I am in a new temporary directory
    When I wait 1 seconds
    And I run the command "echo after-wait"
    Then the output should contain "after-wait"

  Scenario: I wait N milliseconds is accepted and runs without error
    Given I am in a new temporary directory
    When I wait 100 milliseconds
    And I run the command "echo after-ms-wait"
    Then the output should contain "after-ms-wait"

  Scenario: I wait 1 second singular form is accepted
    Given I am in a new temporary directory
    When I wait 1 second
    And I run the command "echo singular"
    Then the output should contain "singular"

  Scenario: I wait 1 millisecond singular form is accepted
    Given I am in a new temporary directory
    When I wait 1 millisecond
    And I run the command "echo ms-singular"
    Then the output should contain "ms-singular"

  # ── retry command ─────────────────────────────────────────────────────────

  Scenario: I retry up to N times executes until the command exits 0
    Given I am in a new temporary directory
    And I create the file "counter.txt" containing "0"
    When I retry up to 5 times every 1s until the command "COUNT=$(cat counter.txt); COUNT=$((COUNT+1)); echo $COUNT > counter.txt; [ $COUNT -ge 3 ]" exits 0
    Then the file "counter.txt" should contain "3"

  Scenario: I retry up to N times succeeds on first attempt when command is already passing
    Given I am in a new temporary directory
    When I retry up to 3 times every 1s until the command "echo immediate-success" exits 0
    Then the output should contain "immediate-success"

  Scenario: I retry up to N times fails when max attempts exhausted
    Given I am in a new temporary directory
    And I create the file "features/retry-fail.feature" with content:
      """
      Feature: Retry fail
        Scenario: Command never exits 0
          Given I am in a new temporary directory
          When I retry up to 3 times every 1s until the command "exit 1" exits 0
          Then the exit code should be 0
      """
    When I run lobster "run --features features/retry-fail.feature --ci"
    Then the exit code should be 1

  # ── HTTP polling ──────────────────────────────────────────────────────────

  @docker
  Scenario: I poll URL until the status is 200 succeeds when server is available
    Given I am in a new temporary directory
    When I poll "${HTTPBIN_URL}/get" every 1s until the status is 200 for up to 5s
    Then the directory "." should exist

  Scenario: I poll URL until status fails when server never returns expected status
    Given I am in a new temporary directory
    And I create the file "features/poll-fail.feature" with content:
      """
      Feature: Poll fail
        Scenario: Polling unreachable URL times out
          Given I am in a new temporary directory
          When I poll "http://127.0.0.1:19771" every 1s until the status is 200 for up to 3s
          Then the directory "." should exist
      """
    When I run lobster "run --features features/poll-fail.feature --ci"
    Then the exit code should be 1

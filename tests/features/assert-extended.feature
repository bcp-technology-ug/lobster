Feature: Extended JSON assertion step definitions
  As a test author
  I want rich JSON assertions including regex, type checks, numeric comparisons
  and array searches
  So that I can deeply validate complex API response structures
  #
  # All scenarios tagged @docker require the compose stack (fixture-server on
  # 19790 serving /payload.json with the test payload).

  # ── regex match ───────────────────────────────────────────────────────────

  @docker
  Scenario: JSON field should match passes for a matching regex
    Given I am in a new temporary directory
    And I set the base URL to "${FIXTURE_URL}"
    When I send a GET request to "/payload.json"
    Then the response JSON field "name" should match "^alice-\d+$"

  @docker
  Scenario: JSON field should match fails for a non-matching regex
    Given I am in a new temporary directory
    And I create the file "features/json-regex-fail.feature" with content:
      """
      Feature: JSON regex fail
        Scenario: JSON field does not match regex
          Given I set the base URL to "${FIXTURE_URL}"
          When I send a GET request to "/payload.json"
          Then the response JSON field "name" should match "^\d+$"
      """
    When I run lobster "run --features features/json-regex-fail.feature --env FIXTURE_URL=${FIXTURE_URL} --no-compose --ci"
    Then the exit code should be 1

  # ── type assertions ───────────────────────────────────────────────────────

  @docker
  Scenario: JSON field should be a number passes for a numeric field
    Given I am in a new temporary directory
    And I set the base URL to "${FIXTURE_URL}"
    When I send a GET request to "/payload.json"
    Then the response JSON field "age" should be a number

  @docker
  Scenario: JSON field should be a string passes for a string field
    Given I am in a new temporary directory
    And I set the base URL to "${FIXTURE_URL}"
    When I send a GET request to "/payload.json"
    Then the response JSON field "name" should be a string

  @docker
  Scenario: JSON field should be a boolean passes for a boolean field
    Given I am in a new temporary directory
    And I set the base URL to "${FIXTURE_URL}"
    When I send a GET request to "/payload.json"
    Then the response JSON field "active" should be a boolean

  @docker
  Scenario: JSON field should be null passes for a null field
    Given I am in a new temporary directory
    And I set the base URL to "${FIXTURE_URL}"
    When I send a GET request to "/payload.json"
    Then the response JSON field "meta" should be null

  @docker
  Scenario: JSON field type check fails when the type is wrong
    Given I am in a new temporary directory
    And I create the file "features/json-type-fail.feature" with content:
      """
      Feature: JSON type fail
        Scenario: Field is not a number
          Given I set the base URL to "${FIXTURE_URL}"
          When I send a GET request to "/payload.json"
          Then the response JSON field "name" should be a number
      """
    When I run lobster "run --features features/json-type-fail.feature --env FIXTURE_URL=${FIXTURE_URL} --no-compose --ci"
    Then the exit code should be 1

  # ── numeric comparisons ───────────────────────────────────────────────────

  @docker
  Scenario: JSON field should be greater than passes when value exceeds threshold
    Given I am in a new temporary directory
    And I set the base URL to "${FIXTURE_URL}"
    When I send a GET request to "/payload.json"
    Then the response JSON field "age" should be greater than 18

  @docker
  Scenario: JSON field should be less than passes when value is below threshold
    Given I am in a new temporary directory
    And I set the base URL to "${FIXTURE_URL}"
    When I send a GET request to "/payload.json"
    Then the response JSON field "age" should be less than 100

  @docker
  Scenario: JSON field should be between passes when value is in range
    Given I am in a new temporary directory
    And I set the base URL to "${FIXTURE_URL}"
    When I send a GET request to "/payload.json"
    Then the response JSON field "score" should be between 90 and 100

  @docker
  Scenario: JSON field greater than fails when value is not greater
    Given I am in a new temporary directory
    And I create the file "features/json-gt-fail.feature" with content:
      """
      Feature: JSON gt fail
        Scenario: Field is not greater than threshold
          Given I set the base URL to "${FIXTURE_URL}"
          When I send a GET request to "/payload.json"
          Then the response JSON field "age" should be greater than 100
      """
    When I run lobster "run --features features/json-gt-fail.feature --env FIXTURE_URL=${FIXTURE_URL} --no-compose --ci"
    Then the exit code should be 1

  # ── array element search ──────────────────────────────────────────────────

  @docker
  Scenario: JSON array should contain element where field equals value
    Given I am in a new temporary directory
    And I set the base URL to "${FIXTURE_URL}"
    When I send a GET request to "/payload.json"
    Then the response JSON array "items" should contain an element where "id" equals "item-001"

  @docker
  Scenario: JSON array element search works for any position in the array
    Given I am in a new temporary directory
    And I set the base URL to "${FIXTURE_URL}"
    When I send a GET request to "/payload.json"
    Then the response JSON array "items" should contain an element where "id" equals "item-002"

  @docker
  Scenario: Array element search fails when no element matches
    Given I am in a new temporary directory
    And I create the file "features/json-arr-fail.feature" with content:
      """
      Feature: JSON array element fail
        Scenario: No element matches
          Given I set the base URL to "${FIXTURE_URL}"
          When I send a GET request to "/payload.json"
          Then the response JSON array "items" should contain an element where "id" equals "nonexistent-id"
      """
    When I run lobster "run --features features/json-arr-fail.feature --env FIXTURE_URL=${FIXTURE_URL} --no-compose --ci"
    Then the exit code should be 1

  # ── DataTable bulk assertion ──────────────────────────────────────────────

  @docker
  Scenario: JSON should include fields passes for all matching fields
    Given I am in a new temporary directory
    And I set the base URL to "${FIXTURE_URL}"
    When I send a GET request to "/payload.json"
    Then the response JSON should include fields:
      | field | value    |
      | name  | alice-42 |
      | role  | admin    |

  @docker
  Scenario: JSON include fields fails when a field has wrong value
    Given I am in a new temporary directory
    And I create the file "features/json-fields-fail.feature" with content:
      """
      Feature: JSON fields fail
        Scenario: One field has wrong value
          Given I set the base URL to "${FIXTURE_URL}"
          When I send a GET request to "/payload.json"
          Then the response JSON should include fields:
            | field | value      |
            | name  | wrong-name |
            | role  | admin      |
      """
    When I run lobster "run --features features/json-fields-fail.feature --env FIXTURE_URL=${FIXTURE_URL} --no-compose --ci"
    Then the exit code should be 1

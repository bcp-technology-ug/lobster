# Lobster Concepts

Lobster is an end-to-end BDD testing tool for systems that include multiple services, infrastructure dependencies, and cross-service behavior.

## BDD and Gherkin

Behavior-Driven Development (BDD) describes system behavior in plain language so that engineering, QA, and product teams can align on expected outcomes.

Lobster uses Gherkin feature files:

- Feature: describes a capability
- Background: shared preconditions that run before each scenario in a feature
- Scenario: describes one concrete behavior path
- Scenario Outline: parameterized scenario expanded by Examples rows
- Given: describes initial state
- When: describes an action or event
- Then: describes expected outcomes

Example:

```gherkin
Feature: Order checkout
  Scenario: Successful card payment
    Given the payment service is running
    When I submit a valid card payment request
    Then the response status should be 200
    And the order should be marked as paid
```

## Living documentation

Feature files are both executable tests and documentation.

This enables:

- Readable acceptance criteria
- Traceable behavior expectations
- Easier collaboration across engineering and non-engineering roles

## E2E-first design

Lobster is explicitly focused on end-to-end behavior.

It validates that:

- Services can start together
- Integrations can communicate correctly
- User-visible workflows succeed under realistic conditions

Lobster complements unit and integration tests. It does not replace them.

## Execution modes

Lobster is CLI-first and supports two execution modes:

- local mode: run orchestration and scenarios in the local CLI process
- daemon mode: submit and observe runs through a remote daemon API

Both modes preserve the same scenario semantics and report behavior.

Capability surfaces:

- CLI over gRPC
- Wish over gRPC
- HTTP clients through gRPC-Gateway

All surfaces map to one backend capability model.

## Scenario execution model

At a high level, lobster runs a scenario in this order:

1. Parse and validate feature files
2. Resolve matching step definitions
3. Ensure required infrastructure is running
4. Execute step handlers in sequence
5. Evaluate assertions and report outcomes
6. Teardown or retain stack based on run policy

Run interaction models:

- synchronous streaming: client stays attached and receives run events live
- asynchronous jobs: client submits, receives run identifier, and polls or streams later

### Tag filtering

Lobster uses Cucumber-style boolean tag expressions for scenario selection.

Example:

```text
@smoke and not @wip
```

### Undefined steps during run

When undefined steps are found during `lobster run`, execution continues to collect all undefined steps discovered in the run and then fails at the end with a consolidated summary.

## Step definitions

A step definition maps a natural-language step to executable logic.

Lobster supports:

- Built-in step definitions for common E2E patterns
- Statically registered project-specific step extensions in v0.1
- Runtime plugin loading planned for a later release (v0.2 target)

Data and variable model in v0.1:

- Scenario-scoped variables for isolated test behavior
- Optional suite-scoped shared variables for cross-scenario coordination
- Gherkin Data Tables supported as structured step arguments

See docs/step-definitions.md for details.

## Determinism and reliability

Reliable E2E tests require repeatable environments and predictable actions.

Lobster encourages:

- Explicit setup and teardown
- Health-check-based service readiness
- Timeouts and retries for eventually consistent systems
- Clear failure diagnostics for fast triage

## Persistence model

Lobster uses SQLite as the primary datastore and sqlc for typed query generation.

Persisted data includes run history and detailed scenario/step outcomes. JSON and JUnit files remain available for CI and external tooling compatibility.

In daemon mode, persistence writes are daemon-owned to prevent cross-process write contention.

See docs/persistence.md and docs/api-reference.md for canonical behavior.

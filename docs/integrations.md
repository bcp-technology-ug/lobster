# Integrations

Lobster provides integration adapters that help test suites prepare, interact with, and reset external systems.

## Integration adapter concept

An integration adapter is a service-specific module that exposes structured lifecycle hooks.

Typical hooks:

- setup: prepare required entities before suite or scenario execution
- reset: clean or reseed state between scenarios
- teardown: remove transient artifacts after execution

Conceptual adapter contract:

```go
type Adapter interface {
  Setup(ctx context.Context) error
  Reset(ctx context.Context) error
  Teardown(ctx context.Context) error
}
```

## Why adapters exist

Many E2E suites need predictable external-system state. Adapters reduce brittle ad-hoc setup scripts and provide reusable behavior with clear diagnostics.

## Keycloak example

Keycloak is a common identity dependency in distributed systems. A Keycloak adapter can automate:

- Realm creation and update
- Client creation for applications
- User and role provisioning
- Token acquisition for test requests
- Realm import/export for fixture portability

Example integration config:

```yaml
integrations:
  keycloak:
    enabled: true
    base_url: http://keycloak:8080
    admin_user: admin
    admin_password_env: KEYCLOAK_ADMIN_PASSWORD
    realm: test-realm
    reset_between_scenarios: true
```

Example usage in steps:

```gherkin
Given keycloak realm "test-realm" exists
And user "alice" exists in realm "test-realm"
When I request an access token for user "alice"
Then the token should contain role "customer"
```

## Other adapter candidates

Lobster can support additional adapters using the same pattern:

- message brokers
- mail servers
- object storage
- feature flag services
- custom internal platform services

## Extension model by version

- v0.1: adapters are registered statically through the build.
- v0.2 target: runtime adapter loading through RPC plugin infrastructure.
- The runtime loader must preserve the same explicit compatibility and redaction rules as static registration.
- Static registration remains the deterministic fallback path if a runtime-loaded adapter fails validation or negotiation.

Transport contract direction:

- Adapter lifecycle behavior maps cleanly to proto-first service contracts.
- In daemon mode, adapter lifecycle hooks are executed on the daemon host.
- CLI and Wish clients observe adapter effects through run events and diagnostics.

## Adapter design principles

- deterministic setup and reset behavior
- clear failure messages with system context
- minimal hidden side effects
- explicit configuration and secret handling
- sensitive value redaction by default in logs and reports

## Remote execution behavior

In daemon mode:

- setup/reset/teardown hooks run server-side
- adapter state metadata is persisted in SQLite for deterministic lifecycle handling
- clients do not directly mutate adapter state stores

See docs/api-reference.md and docs/persistence.md for canonical contract and storage rules.

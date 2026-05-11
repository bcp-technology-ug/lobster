# Docker Compose Integration

Lobster is designed to run tests against a real, compose-managed system.

## Why Compose integration matters

End-to-end scenarios are only meaningful if services run in realistic conditions. Lobster orchestrates Docker Compose as part of each test run so tests execute against actual service interactions.

In v0.1, lobster controls the stack through Docker APIs with compose-compatible semantics rather than shelling out to `docker compose`.

In daemon mode, the daemon owns compose orchestration and exposes the same behavior through API calls.

## Compose inputs

Lobster accepts one or more compose files:

- Base environment file (for core services)
- Test override file (for mocks, extra tooling, or test-specific config)

Example:

```bash
lobster run --compose docker-compose.yml --compose docker-compose.test.yml
```

## Stack lifecycle

During lobster run:

1. Resolve compose file list and project name
2. Start services (`compose up -d` equivalent)
3. Wait for readiness strategy (health checks or explicit probes)
4. Execute selected scenarios
5. Collect logs and artifacts when configured
6. Teardown (`compose down`) unless keep-stack is enabled

Mode behavior:

- local mode: lifecycle runs in the local CLI process
- daemon mode: lifecycle runs on the daemon host; clients observe lifecycle through API events

Execution note:

- Scenario execution is serial in v0.1 to reduce race conditions and simplify environment determinism.

## Readiness and health checks

Recommended approach:

- Define healthcheck for each critical service
- Set explicit timeouts in lobster config
- Fail early with a startup diagnostics summary if required services never become healthy

## Networking model

Test execution runs in an environment that can reach compose services by service name. This allows steps to use stable internal endpoints (for example, http://api:8080) without machine-specific host mapping.

In daemon mode, service-name networking is resolved on the daemon host network.

## Teardown strategies

- default: full teardown after run
- keep-stack: retain containers for local debugging
- on-failure retain policy: optional for post-failure inspection

## Failure diagnostics

If startup or runtime fails, lobster should report:

- services that failed health checks
- final container state
- relevant service logs
- scenario and step context of the failure

When running through daemon mode, these diagnostics are streamed and persisted by the daemon before client presentation.

## Best practices

- Keep test compose files minimal and deterministic
- Avoid hidden state between runs
- Use explicit service dependencies and health checks
- Pin image tags used in CI for reproducibility

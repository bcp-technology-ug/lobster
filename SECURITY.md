# Security Policy

## Supported versions

Only the latest release of lobster receives security fixes.

| Version | Supported |
| ------- | --------- |
| Latest  | ✅        |
| Older   | ❌        |

## Reporting a vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Report vulnerabilities by emailing **[support@bcp.technology](mailto:support@bcp.technology)** with the subject line `[lobster] Security Vulnerability`.

Include as much of the following as you can:

- A description of the vulnerability and its potential impact
- Steps to reproduce or a proof-of-concept
- Affected versions
- Any suggested mitigations

You will receive an acknowledgement within **3 business days**. We aim to release a fix or mitigation within **14 days** of confirmation, depending on complexity.

We follow [responsible disclosure](https://en.wikipedia.org/wiki/Responsible_disclosure): please give us reasonable time to address the issue before public disclosure.

## Scope

The following are in scope:

- `lobster` CLI binary
- `lobsterd` daemon and its gRPC / HTTP API
- Container images published to `ghcr.io/bcp-technology-ug/`

The following are **out of scope**:

- Vulnerabilities in third-party dependencies that have no available fix
- Issues requiring physical access to a host machine
- Denial-of-service through resource exhaustion in a local, single-user setup

## Security design notes

- `lobsterd` accepts bearer tokens and supports mutual TLS for authentication. Use `--insecure-local` only on trusted local networks.
- SQLite databases are created with restrictive file permissions; do not expose the database file path to untrusted users.
- Lobster does not phone home, collect telemetry, or require network access except where explicitly configured (OTLP, Keycloak, remote daemon).

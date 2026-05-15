# Roadmap

This document describes what is planned for lobster and the principles that guide prioritisation.

Lobster uses [SemVer](https://semver.org/). Before v1.0, minor releases may include breaking changes with explicit deprecation notices and removal targets documented in [CHANGELOG.md](CHANGELOG.md).

---

## Current release — v0.1.0 (May 2026)

The initial public release. The core execution loop is stable and dogfooded against lobster's own test suite.

**Stable in v0.1:**

- `lobster` CLI — `init`, `validate`, `lint`, `plan`, `run`, `config`, `runs`, `plans`, `stack`, `integrations`, `admin`
- `lobsterd` daemon with gRPC + HTTP/JSON gateway
- Local in-process execution and remote daemon execution (`--executor-mode daemon`)
- Docker Compose lifecycle management via Docker SDK
- Background, Scenario Outline, and Data Table support
- Built-in HTTP, shell, filesystem, service, variable, wait, and gRPC step definitions
- SQLite persistence with run history and step-level detail
- Console, JUnit XML, and JSON report output
- Terraform-style planning (`lobster plan` → `lobster run --from-plan`)
- Quarantine-tag workflow (`@quarantine`)
- Monorepo workspace support
- Basic Keycloak integration adapter
- OpenTelemetry trace export

---

## v0.2 — planned

Focus: **stability, DX polish, and plugin foundations.**

| Item | Description |
|---|---|
| Parallel scenario execution | Optional `--parallel N` flag for concurrent scenario runs with per-scenario isolation |
| Step registry plugin loading | Runtime discovery of external step registries from lobster.yaml `steps.registries` |
| `lobster steps` output improvements | Markdown table format, filterable by module, JSON output |
| `lobster coverage` CLI | Coverage report generation as a standalone command (`lobster coverage --format html`) |
| Improved TUI foundation | Bubbletea-based interactive run watcher (experimental, opt-in) |
| Windows CGO improvements | Documented MinGW-w64 setup and pre-built binary support for Windows |
| `lobster doctor` expansion | Check Docker, Compose, daemon connectivity, and migration status in one command |
| Retry step improvements | Per-step retry policy configurable in lobster.yaml |
| gRPC step definitions | Stable built-in gRPC step module (currently experimental) |
| Matrix run profiles | First-class support for running the same suite against multiple configuration profiles |

---

## v0.3 — planned

Focus: **remote-first features and ecosystem integration.**

| Item | Description |
|---|---|
| Web UI for `lobsterd` | Optional read-only web dashboard for run history and live status |
| Artifact storage backends | S3/GCS/MinIO support for plan artifact storage alongside the current filesystem backend |
| Expanded integration adapters | PostgreSQL, Redis, and custom HTTP health-check adapters |
| `lobster init` template gallery | Community-contributed `lobster init --template` project templates |
| Nix flake | First-class Nix packaging for reproducible dev environments |
| Homebrew tap | `brew install bcp-technology-ug/tap/lobster` — formula published on first stable release |

---

## v1.0 — target

Focus: **stable API contract, full plugin ecosystem, and production hardening.**

- API stability contract: all CLI flags, config schema, gRPC API, and step registry interface are stable
- No breaking changes without a deprecation cycle of at least one minor release
- Full TUI with Bubbletea-powered interactive run, watch, and history browsing
- Runtime plugin loading with verified extension registry support
- Comprehensive E2E coverage of all features via lobster's own dogfood suite

---

## What is not on the roadmap

Consistent with [MISSION.md](MISSION.md), lobster will not:

- Become a SaaS product or add any billing-gated features
- Add telemetry, analytics, or data collection of any kind
- Replace unit or integration tests — it complements them
- Require an internet connection to run

---

## Contributing to the roadmap

Roadmap items are proposals, not commitments. Priorities shift based on real usage and community feedback.

To propose something:

1. Open a [GitHub Discussion](https://github.com/bcp-technology-ug/lobster/discussions) in the **Ideas** category.
2. If the idea gains traction, file a [feature request issue](https://github.com/bcp-technology-ug/lobster/issues/new?template=feature_request.yml).
3. Implementation PRs that align with roadmap items are prioritised for review.

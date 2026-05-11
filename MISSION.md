# Mission

## What is lobster?

Lobster is an open-source, CLI-first, end-to-end BDD testing framework. It exists to give engineering teams the power to test their *entire system* — all services, all infrastructure, all integrations — as a cohesive whole, using human-readable Gherkin specifications that serve as both executable tests and living documentation.

## Why does lobster exist?

End-to-end testing is the hardest and most valuable kind of testing. It is also the most underserved by open tooling. The ecosystem has fractured into:

- **Browser-only tools** that ignore backend services entirely.
- **API-only tools** that ignore the broader system context.
- **SaaS platforms** that charge per seat, per run, or per month — and take your test data with them.
- **Framework-specific runners** that do not compose with real infrastructure.

The result is that most teams either skip true E2E tests, wire together brittle shell scripts, or pay for a managed platform and accept its constraints. None of these are acceptable outcomes.

Lobster is built because the open-source ecosystem deserves a tool that:

1. Speaks the language engineers and stakeholders already understand (Gherkin / BDD).
2. Works with the infrastructure tooling teams already use (Docker Compose).
3. Runs everywhere — on a laptop and in CI — without special accounts or network access.
4. Is extensible without being overwhelming.

## What lobster will always be

- **Free and open.** MIT licensed. No usage limits. No telemetry. No accounts required.
- **CLI/CI-first.** Every capability must work headlessly in a pipeline. The interactive TUI is an enhancement, not a requirement.
- **Infrastructure-aware.** Real E2E tests require real services. Lobster orchestrates those services; it does not mock them away.
- **BDD-native.** Gherkin is the lingua franca. Test intent is always expressed in plain language first.
- **Community-owned.** Decisions are made in the open. Contributions are welcome. No single company controls the roadmap.
- **Dual-developed in one codebase.** Lobster is used internally at BCP Technology and developed openly on GitHub, with both streams published and maintained together.

## What lobster will never be

- A SaaS product with a billing page.
- A tool that requires an internet connection to run.
- A closed-source fork sold under a different name.
- A tool that trades contributor work for venture-backed commercial lock-in.
- A replacement for unit or integration tests — lobster complements them, it does not replace them.

## Who is lobster for?

Lobster is for engineering teams who build systems composed of multiple services and who want a disciplined, readable, and repeatable way to verify that those systems work correctly — together, end to end, in an environment that mirrors production as closely as possible.

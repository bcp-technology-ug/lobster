# Contributing to lobster

Thanks for your interest in contributing to lobster.

Lobster is a CLI-first, open-source, end-to-end BDD testing tool. We welcome bug reports, documentation improvements, feature work, and integration contributions.

## Ways to contribute

- Report bugs and edge cases
- Improve documentation and examples
- Add or improve built-in step definitions
- Build new integration adapters
- Improve CLI and CI behavior

## Before you start

- Search existing issues and discussions first
- Open an issue for larger feature ideas before implementation
- Keep pull requests focused and scoped to one concern

## Dual-development model

Lobster is dual-developed:

- It is an internal BCP Technology tool and a public GitHub project.
- Internal and public contributions target the same codebase and roadmap.
- Internal changes are published alongside public changes as part of normal development.
- GitHub contributions are reviewed for inclusion in the same ongoing internal development stream.

This means there is no separate public-only fork and no separate long-term private code line for core behavior.

## Development workflow

1. Fork the repository and create a feature branch from main.
2. Make your changes with clear commit messages.
3. Ensure tests and linters pass locally.
4. Open a pull request with context, motivation, and testing notes.

## Commit conventions

Use Conventional Commits where possible:

- feat: new functionality
- fix: bug fix
- docs: documentation-only changes
- refactor: internal code changes without behavior change
- test: adding or updating tests
- chore: maintenance and tooling changes

Example:

```text
feat(cli): add validate --strict mode
```

## Pull request checklist

- Pull request title describes the change clearly
- Linked issue included when relevant
- Tests added or updated for behavior changes
- Documentation updated when user-facing behavior changes
- No unrelated refactors mixed into the same pull request

## Adding built-in step definitions

When adding built-in steps, follow these principles:

- Keep steps deterministic and idempotent where possible
- Prefer explicit, readable step text over magic behavior
- Return clear error messages that include scenario context
- Avoid hidden side effects across scenarios

Include:

- Unit tests for parser/matcher behavior
- Integration tests for real execution behavior
- Documentation updates in docs/step-definitions.md

## Writing step extensions

Lobster supports extending step definitions through statically registered extensions in v0.1.

Suggested extension structure:

1. Create a Go package for your extension
2. Implement the StepRegistrar contract expected by lobster
3. Register step patterns and handlers
4. Add extension registry configuration in lobster.yaml

Document extension behavior and required configuration in your pull request.

## Code quality expectations

- Prefer small, composable functions
- Handle errors explicitly and return actionable messages
- Keep CLI output readable in both interactive and non-interactive modes
- Avoid introducing breaking changes without proposal and discussion

## Compatibility and versioning policy

- Lobster uses SemVer for release versions.
- Before v1.0, minor releases may include breaking changes if clearly documented.
- Extension compatibility must declare and validate `steps.api_version`.

## Deprecation policy

- Deprecated flags/config/options should emit runtime warnings.
- Every deprecation must include a target removal version in docs and changelog notes.

## Release channels

- `stable`: production-ready releases
- `nightly`: pre-release builds for early testing and feedback

## Reporting security issues

Do not open public issues for security vulnerabilities.

Instead, contact: opensource@bcp.technology

## Code of Conduct

By participating in this project, you agree to follow the Code of Conduct in CODE_OF_CONDUCT.md.

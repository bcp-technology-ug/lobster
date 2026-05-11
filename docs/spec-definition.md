# Spec Definition

This document defines how Lobster capabilities are specified using company-standard, contract-first practices.

## Purpose

Lobster specifications must be executable contracts, not prose-only intent.

Source-of-truth artifacts:

- proto contracts for API surface and validation rules
- SQL queries for data access behavior
- SQL migrations for schema evolution

Business logic is implemented after these contracts are defined and generated.

## Authoring standards

## 1. Define proto contract first

For each new capability, update or add proto files under `proto/lobster/v1/`.

Requirements:

- versioned package names
- explicit request and response messages
- explicit service RPC definitions
- explicit `google.api.http` annotations
- `buf.validate` field rules for request validation

## 2. Define persistence contract

If persistence is needed:

- add migration pair in `migrations/`
- add sqlc queries under `sql/<surface>/`
- follow sqlc naming conventions (`Get`, `List`, `Create`, `Update`, `Delete`, `Count`)

## 3. Generate all derived artifacts

Regenerate from source contracts:

- buf generation for Go, gateway, OpenAPI, optional TypeScript
- sqlc generation for repository packages

Generated code is never hand-edited.

Current scope decision:

- OpenAPI generation is required in CI.
- TypeScript client generation is deferred until API stabilization.

## 4. Implement business logic

Implement service and runner logic against generated interfaces and repository abstractions.

Do not implement manual transport handlers or handwritten request mappers for generated APIs.

## 5. Validate compatibility

Before merge:

- proto lint passes
- proto breaking-change checks pass (or are intentionally versioned)
- generation is up to date
- migrations are reversible and reviewed
- generated artifacts are committed with no CI drift
- changelog entry exists for every PR touching proto or SQL contracts

## Required repository standards

## Proto and buf

- `buf.yaml` defines modules, dependencies, lint, and breaking policy
- lint policy uses STANDARD
- breaking policy uses FILE against main branch
- `buf.gen.yaml` defines plugin outputs under `gen/`

## sqlc

- `sqlc.yaml` defines one or more query sets by surface
- schema source points to `migrations/`
- generated output is written under `gen/sqlc/<surface>/`

## Migrations

- sequential migration numbering
- up/down pair for every migration
- backward-compatible rollout patterns for high-risk changes

## Checklist for new API capability

1. Add or update proto contract in `proto/lobster/v1/`.
2. Add HTTP annotations and validation rules.
3. Add migration files if schema changes are required.
4. Add or update sqlc query files in `sql/<surface>/`.
5. Regenerate proto and sqlc outputs.
6. Implement service logic using generated interfaces.
7. Update docs impacted by the capability.
8. Run lint, breaking checks, tests, and build.

## Review guardrails

- No manual edits to generated files.
- No undocumented breaking contract changes.
- No schema change without migration.
- No data access path that bypasses reviewed SQL contracts.

## Contract ownership

- Lobster core maintainers own contract approval.
- Contract changes require explicit changelog entries.
- Before v1.0, contract removals may be made in a breaking release when clearly documented.
- After v1.0, breaking removals require at least one minor release of deprecation before removal.
- Contract governance follows the dual-development model: internal BCP and public GitHub changes are maintained in the same codebase and changelog stream.

PR governance:

- every PR that modifies proto or SQL contracts must include changelog impact notes.

See [api-reference.md](api-reference.md), [persistence.md](persistence.md), and [project-structure.md](project-structure.md) for concrete contract and layout details.
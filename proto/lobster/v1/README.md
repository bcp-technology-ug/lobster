# Lobster v1 Proto Contracts

This directory defines the canonical Lobster API contracts.

Current v1 surfaces:
- common
- config
- run
- plan
- stack
- admin
- integrations

Notes:
- API paths are mapped under /api/v1.
- Redaction metadata is defined via custom field options in common.proto.
- Integrations remain intentionally generic in v1; vendor-heavy states are deferred to v2.

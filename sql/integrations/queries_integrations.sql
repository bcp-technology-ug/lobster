-- name: UpsertIntegrationAdapter :exec
INSERT INTO integration_adapters (
  adapter_id,
  name,
  type,
  state,
  config_extension_type_url,
  config_extension_value,
  state_extension_type_url,
  state_extension_value,
  updated_at
) VALUES (
  sqlc.arg(adapter_id),
  sqlc.arg(name),
  sqlc.arg(type),
  sqlc.arg(state),
  sqlc.narg(config_extension_type_url),
  sqlc.narg(config_extension_value),
  sqlc.narg(state_extension_type_url),
  sqlc.narg(state_extension_value),
  sqlc.arg(updated_at)
)
ON CONFLICT (adapter_id) DO UPDATE SET
  name = excluded.name,
  type = excluded.type,
  state = excluded.state,
  config_extension_type_url = excluded.config_extension_type_url,
  config_extension_value = excluded.config_extension_value,
  state_extension_type_url = excluded.state_extension_type_url,
  state_extension_value = excluded.state_extension_value,
  updated_at = excluded.updated_at;

-- name: SetIntegrationAdapterState :exec
UPDATE integration_adapters
SET
  state = sqlc.arg(state),
  updated_at = sqlc.arg(updated_at)
WHERE adapter_id = sqlc.arg(adapter_id);

-- name: GetIntegrationAdapter :one
SELECT *
FROM integration_adapters
WHERE adapter_id = sqlc.arg(adapter_id)
LIMIT 1;

-- name: ListIntegrationAdaptersPage :many
SELECT *
FROM integration_adapters
WHERE (
    sqlc.narg(cursor_updated_at) IS NULL
    OR updated_at < sqlc.narg(cursor_updated_at)
    OR (updated_at = sqlc.narg(cursor_updated_at) AND adapter_id > sqlc.narg(cursor_adapter_id))
  )
ORDER BY updated_at DESC, adapter_id ASC
LIMIT sqlc.arg(page_size);

-- name: DeleteIntegrationAdapter :exec
DELETE FROM integration_adapters
WHERE adapter_id = sqlc.arg(adapter_id);

-- name: DeleteIntegrationAdapterCapabilities :exec
DELETE FROM integration_adapter_capabilities
WHERE adapter_id = sqlc.arg(adapter_id);

-- name: UpsertIntegrationAdapterCapability :exec
INSERT INTO integration_adapter_capabilities (adapter_id, name, enabled)
VALUES (
  sqlc.arg(adapter_id),
  sqlc.arg(name),
  sqlc.arg(enabled)
)
ON CONFLICT (adapter_id, name) DO UPDATE SET
  enabled = excluded.enabled;

-- name: ListIntegrationAdapterCapabilities :many
SELECT *
FROM integration_adapter_capabilities
WHERE adapter_id = sqlc.arg(adapter_id)
ORDER BY name ASC;

-- name: AppendIntegrationAdapterStateEvent :exec
INSERT INTO integration_adapter_state_events (
  adapter_id,
  sequence,
  previous_state,
  next_state,
  reason,
  changed_at
) VALUES (
  sqlc.arg(adapter_id),
  sqlc.arg(sequence),
  sqlc.narg(previous_state),
  sqlc.arg(next_state),
  sqlc.narg(reason),
  sqlc.arg(changed_at)
);

-- name: ListIntegrationAdapterStateEvents :many
SELECT *
FROM integration_adapter_state_events
WHERE adapter_id = sqlc.arg(adapter_id)
ORDER BY sequence DESC
LIMIT sqlc.arg(limit_rows);

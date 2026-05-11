-- name: UpsertStack :exec
INSERT INTO stacks (
  stack_id,
  workspace_id,
  profile_name,
  project_name,
  status,
  created_at,
  updated_at
) VALUES (
  sqlc.arg(stack_id),
  sqlc.arg(workspace_id),
  sqlc.arg(profile_name),
  sqlc.arg(project_name),
  sqlc.arg(status),
  sqlc.arg(created_at),
  sqlc.arg(updated_at)
)
ON CONFLICT (workspace_id) DO UPDATE SET
  stack_id = excluded.stack_id,
  profile_name = excluded.profile_name,
  project_name = excluded.project_name,
  status = excluded.status,
  updated_at = excluded.updated_at;

-- name: GetStackByWorkspace :one
SELECT *
FROM stacks
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 1;

-- name: GetStackByID :one
SELECT *
FROM stacks
WHERE stack_id = sqlc.arg(stack_id)
LIMIT 1;

-- name: ListStacksByStatus :many
SELECT *
FROM stacks
WHERE status = sqlc.arg(status)
ORDER BY updated_at DESC, stack_id ASC
LIMIT sqlc.arg(limit_rows);

-- name: DeleteStackByWorkspace :exec
DELETE FROM stacks
WHERE workspace_id = sqlc.arg(workspace_id);

-- name: DeleteStackComponents :exec
DELETE FROM stack_components
WHERE stack_id = sqlc.arg(stack_id);

-- name: UpsertStackComponent :exec
INSERT INTO stack_components (
  stack_id,
  name,
  image,
  container_id,
  status,
  health,
  updated_at
) VALUES (
  sqlc.arg(stack_id),
  sqlc.arg(name),
  sqlc.narg(image),
  sqlc.narg(container_id),
  sqlc.narg(status),
  sqlc.arg(health),
  sqlc.arg(updated_at)
)
ON CONFLICT (stack_id, name) DO UPDATE SET
  image = excluded.image,
  container_id = excluded.container_id,
  status = excluded.status,
  health = excluded.health,
  updated_at = excluded.updated_at;

-- name: ListStackComponents :many
SELECT *
FROM stack_components
WHERE stack_id = sqlc.arg(stack_id)
ORDER BY name ASC;

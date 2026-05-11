-- name: CreateExecutionPlan :exec
INSERT INTO execution_plans (
  plan_id,
  workspace_id,
  profile_name,
  selector_feature_path,
  selector_tag_expression,
  selector_profile_name,
  estimated_duration_nanos,
  created_at
) VALUES (
  sqlc.arg(plan_id),
  sqlc.arg(workspace_id),
  sqlc.arg(profile_name),
  sqlc.narg(selector_feature_path),
  sqlc.narg(selector_tag_expression),
  sqlc.narg(selector_profile_name),
  sqlc.arg(estimated_duration_nanos),
  sqlc.arg(created_at)
);

-- name: GetExecutionPlan :one
SELECT *
FROM execution_plans
WHERE plan_id = sqlc.arg(plan_id)
LIMIT 1;

-- name: ListExecutionPlansPage :many
SELECT *
FROM execution_plans
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (
    sqlc.narg(cursor_created_at) IS NULL
    OR created_at < sqlc.narg(cursor_created_at)
    OR (created_at = sqlc.narg(cursor_created_at) AND plan_id < sqlc.narg(cursor_plan_id))
  )
ORDER BY created_at DESC, plan_id DESC
LIMIT sqlc.arg(page_size);

-- name: DeleteExecutionPlan :exec
DELETE FROM execution_plans
WHERE plan_id = sqlc.arg(plan_id);

-- name: UpsertExecutionPlanScenario :exec
INSERT INTO execution_plan_scenarios (
  plan_id,
  scenario_id,
  ordinal,
  feature_name,
  scenario_name,
  estimated_duration_nanos,
  deterministic_feature_name,
  deterministic_scenario_name,
  deterministic_example_row_index,
  deterministic_normalization_version,
  deterministic_stable_hash
) VALUES (
  sqlc.arg(plan_id),
  sqlc.arg(scenario_id),
  sqlc.arg(ordinal),
  sqlc.arg(feature_name),
  sqlc.arg(scenario_name),
  sqlc.arg(estimated_duration_nanos),
  sqlc.narg(deterministic_feature_name),
  sqlc.narg(deterministic_scenario_name),
  sqlc.narg(deterministic_example_row_index),
  sqlc.narg(deterministic_normalization_version),
  sqlc.narg(deterministic_stable_hash)
)
ON CONFLICT (plan_id, scenario_id) DO UPDATE SET
  ordinal = excluded.ordinal,
  feature_name = excluded.feature_name,
  scenario_name = excluded.scenario_name,
  estimated_duration_nanos = excluded.estimated_duration_nanos,
  deterministic_feature_name = excluded.deterministic_feature_name,
  deterministic_scenario_name = excluded.deterministic_scenario_name,
  deterministic_example_row_index = excluded.deterministic_example_row_index,
  deterministic_normalization_version = excluded.deterministic_normalization_version,
  deterministic_stable_hash = excluded.deterministic_stable_hash;

-- name: DeleteExecutionPlanScenarioTags :exec
DELETE FROM execution_plan_scenario_tags
WHERE plan_id = sqlc.arg(plan_id)
  AND scenario_id = sqlc.arg(scenario_id);

-- name: CreateExecutionPlanScenarioTag :exec
INSERT INTO execution_plan_scenario_tags (plan_id, scenario_id, ordinal, tag)
VALUES (
  sqlc.arg(plan_id),
  sqlc.arg(scenario_id),
  sqlc.arg(ordinal),
  sqlc.arg(tag)
);

-- name: UpsertPlanArtifact :exec
INSERT INTO plan_artifacts (
  plan_id,
  artifact_id,
  storage_path,
  envelope_schema_version,
  envelope_schema_revision,
  envelope_media_type,
  envelope_json_export,
  envelope_created_at,
  envelope_payload_sha256,
  envelope_compression_type,
  envelope_signature
) VALUES (
  sqlc.arg(plan_id),
  sqlc.arg(artifact_id),
  sqlc.arg(storage_path),
  sqlc.narg(envelope_schema_version),
  sqlc.narg(envelope_schema_revision),
  sqlc.narg(envelope_media_type),
  sqlc.narg(envelope_json_export),
  sqlc.narg(envelope_created_at),
  sqlc.narg(envelope_payload_sha256),
  sqlc.narg(envelope_compression_type),
  sqlc.narg(envelope_signature)
)
ON CONFLICT (plan_id) DO UPDATE SET
  artifact_id = excluded.artifact_id,
  storage_path = excluded.storage_path,
  envelope_schema_version = excluded.envelope_schema_version,
  envelope_schema_revision = excluded.envelope_schema_revision,
  envelope_media_type = excluded.envelope_media_type,
  envelope_json_export = excluded.envelope_json_export,
  envelope_created_at = excluded.envelope_created_at,
  envelope_payload_sha256 = excluded.envelope_payload_sha256,
  envelope_compression_type = excluded.envelope_compression_type,
  envelope_signature = excluded.envelope_signature;

-- name: GetPlanArtifactByPlanID :one
SELECT *
FROM plan_artifacts
WHERE plan_id = sqlc.arg(plan_id)
LIMIT 1;

-- name: ListExecutionPlanScenarios :many
SELECT *
FROM execution_plan_scenarios
WHERE plan_id = sqlc.arg(plan_id)
ORDER BY ordinal ASC;

-- name: ListExecutionPlanScenarioTags :many
SELECT *
FROM execution_plan_scenario_tags
WHERE plan_id = sqlc.arg(plan_id)
  AND scenario_id = sqlc.arg(scenario_id)
ORDER BY ordinal ASC;

-- name: CreateRun :exec
INSERT INTO runs (
  run_id,
  workspace_id,
  profile_name,
  status,
  idempotency_key,
  selector_feature_path,
  selector_tag_expression,
  selector_profile_name,
  execution_mode,
  execution_run_timeout_nanos,
  execution_step_timeout_nanos,
  execution_soft_assert,
  execution_fail_fast,
  execution_keep_stack,
  created_at
) VALUES (
  sqlc.arg(run_id),
  sqlc.arg(workspace_id),
  sqlc.arg(profile_name),
  sqlc.arg(status),
  sqlc.narg(idempotency_key),
  sqlc.narg(selector_feature_path),
  sqlc.narg(selector_tag_expression),
  sqlc.narg(selector_profile_name),
  sqlc.narg(execution_mode),
  sqlc.narg(execution_run_timeout_nanos),
  sqlc.narg(execution_step_timeout_nanos),
  sqlc.narg(execution_soft_assert),
  sqlc.narg(execution_fail_fast),
  sqlc.narg(execution_keep_stack),
  sqlc.arg(created_at)
);

-- name: SetRunFeature :exec
UPDATE runs
SET
  feature_name = sqlc.narg(feature_name),
  feature_description = sqlc.narg(feature_description)
WHERE run_id = sqlc.arg(run_id);

-- name: UpdateRunStatus :exec
UPDATE runs
SET
  status = sqlc.arg(status),
  started_at = COALESCE(sqlc.narg(started_at), started_at),
  ended_at = COALESCE(sqlc.narg(ended_at), ended_at)
WHERE run_id = sqlc.arg(run_id);

-- name: UpdateRunSummary :exec
UPDATE runs
SET
  summary_total_scenarios = sqlc.arg(summary_total_scenarios),
  summary_passed_scenarios = sqlc.arg(summary_passed_scenarios),
  summary_failed_scenarios = sqlc.arg(summary_failed_scenarios),
  summary_skipped_scenarios = sqlc.arg(summary_skipped_scenarios),
  summary_duration_nanos = sqlc.arg(summary_duration_nanos)
WHERE run_id = sqlc.arg(run_id);

-- name: GetRun :one
SELECT *
FROM runs
WHERE run_id = sqlc.arg(run_id)
LIMIT 1;

-- name: GetRunByWorkspaceAndIdempotencyKey :one
SELECT *
FROM runs
WHERE workspace_id = sqlc.arg(workspace_id)
  AND idempotency_key = sqlc.arg(idempotency_key)
LIMIT 1;

-- name: ListRunsPage :many
SELECT *
FROM runs
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (
    sqlc.narg(cursor_created_at) IS NULL
    OR created_at < sqlc.narg(cursor_created_at)
    OR (created_at = sqlc.narg(cursor_created_at) AND run_id < sqlc.narg(cursor_run_id))
  )
ORDER BY created_at DESC, run_id DESC
LIMIT sqlc.arg(page_size);

-- name: ListRunsPageByStatus :many
SELECT *
FROM runs
WHERE workspace_id = sqlc.arg(workspace_id)
  AND status = sqlc.arg(status)
  AND (
    sqlc.narg(cursor_created_at) IS NULL
    OR created_at < sqlc.narg(cursor_created_at)
    OR (created_at = sqlc.narg(cursor_created_at) AND run_id < sqlc.narg(cursor_run_id))
  )
ORDER BY created_at DESC, run_id DESC
LIMIT sqlc.arg(page_size);

-- name: CountRunsByWorkspace :one
SELECT COUNT(*)
FROM runs
WHERE workspace_id = sqlc.arg(workspace_id);

-- name: DeleteRun :exec
DELETE FROM runs
WHERE run_id = sqlc.arg(run_id);

-- name: ListRetentionCandidateRunsByAge :many
SELECT run_id
FROM runs
WHERE workspace_id = sqlc.arg(workspace_id)
  AND status IN (3, 4, 5)
  AND ended_at IS NOT NULL
  AND ended_at < sqlc.arg(cutoff_time)
ORDER BY ended_at ASC, run_id ASC
LIMIT sqlc.arg(limit_rows);

-- name: ListRetentionCandidateRunsByCount :many
SELECT run_id
FROM runs
WHERE workspace_id = sqlc.arg(workspace_id)
  AND status IN (3, 4, 5)
ORDER BY created_at DESC, run_id DESC
LIMIT -1 OFFSET sqlc.arg(max_keep_runs);

-- name: UpsertRunFeatureTag :exec
INSERT INTO run_feature_tags (run_id, ordinal, tag)
VALUES (sqlc.arg(run_id), sqlc.arg(ordinal), sqlc.arg(tag))
ON CONFLICT (run_id, ordinal) DO UPDATE SET
  tag = excluded.tag;

-- name: DeleteRunFeatureTags :exec
DELETE FROM run_feature_tags
WHERE run_id = sqlc.arg(run_id);

-- name: ListRunFeatureTags :many
SELECT *
FROM run_feature_tags
WHERE run_id = sqlc.arg(run_id)
ORDER BY ordinal ASC;

-- name: UpsertRunBackgroundStep :exec
INSERT INTO run_background_steps (
  run_id,
  step_id,
  ordinal,
  keyword,
  text,
  doc_string_content_type,
  doc_string_content
) VALUES (
  sqlc.arg(run_id),
  sqlc.arg(step_id),
  sqlc.arg(ordinal),
  sqlc.arg(keyword),
  sqlc.arg(text),
  sqlc.narg(doc_string_content_type),
  sqlc.narg(doc_string_content)
)
ON CONFLICT (run_id, step_id) DO UPDATE SET
  ordinal = excluded.ordinal,
  keyword = excluded.keyword,
  text = excluded.text,
  doc_string_content_type = excluded.doc_string_content_type,
  doc_string_content = excluded.doc_string_content;

-- name: DeleteRunBackgroundStepDataTable :exec
DELETE FROM run_background_step_data_table_headers
WHERE run_id = sqlc.arg(run_id)
  AND step_id = sqlc.arg(step_id);

-- name: CreateRunBackgroundStepDataTableHeader :exec
INSERT INTO run_background_step_data_table_headers (run_id, step_id, ordinal, value)
VALUES (
  sqlc.arg(run_id),
  sqlc.arg(step_id),
  sqlc.arg(ordinal),
  sqlc.arg(value)
);

-- name: CreateRunBackgroundStepDataTableRow :exec
INSERT INTO run_background_step_data_table_rows (run_id, step_id, row_index)
VALUES (
  sqlc.arg(run_id),
  sqlc.arg(step_id),
  sqlc.arg(row_index)
);

-- name: CreateRunBackgroundStepDataTableCell :exec
INSERT INTO run_background_step_data_table_cells (run_id, step_id, row_index, cell_index, value)
VALUES (
  sqlc.arg(run_id),
  sqlc.arg(step_id),
  sqlc.arg(row_index),
  sqlc.arg(cell_index),
  sqlc.arg(value)
);

-- name: UpsertRunScenario :exec
INSERT INTO run_scenarios (
  run_id,
  scenario_id,
  ordinal,
  name,
  description,
  status,
  duration_nanos,
  feature_name,
  deterministic_feature_name,
  deterministic_scenario_name,
  deterministic_example_row_index,
  deterministic_normalization_version,
  deterministic_stable_hash
) VALUES (
  sqlc.arg(run_id),
  sqlc.arg(scenario_id),
  sqlc.arg(ordinal),
  sqlc.arg(name),
  sqlc.arg(description),
  sqlc.arg(status),
  sqlc.arg(duration_nanos),
  sqlc.narg(feature_name),
  sqlc.narg(deterministic_feature_name),
  sqlc.narg(deterministic_scenario_name),
  sqlc.narg(deterministic_example_row_index),
  sqlc.narg(deterministic_normalization_version),
  sqlc.narg(deterministic_stable_hash)
)
ON CONFLICT (run_id, scenario_id) DO UPDATE SET
  ordinal = excluded.ordinal,
  name = excluded.name,
  description = excluded.description,
  status = excluded.status,
  duration_nanos = excluded.duration_nanos,
  feature_name = excluded.feature_name,
  deterministic_feature_name = excluded.deterministic_feature_name,
  deterministic_scenario_name = excluded.deterministic_scenario_name,
  deterministic_example_row_index = excluded.deterministic_example_row_index,
  deterministic_normalization_version = excluded.deterministic_normalization_version,
  deterministic_stable_hash = excluded.deterministic_stable_hash;

-- name: DeleteRunScenarioTags :exec
DELETE FROM run_scenario_tags
WHERE run_id = sqlc.arg(run_id)
  AND scenario_id = sqlc.arg(scenario_id);

-- name: CreateRunScenarioTag :exec
INSERT INTO run_scenario_tags (run_id, scenario_id, ordinal, tag)
VALUES (
  sqlc.arg(run_id),
  sqlc.arg(scenario_id),
  sqlc.arg(ordinal),
  sqlc.arg(tag)
);

-- name: UpsertRunScenarioExample :exec
INSERT INTO run_scenario_examples (run_id, scenario_id, row_index)
VALUES (
  sqlc.arg(run_id),
  sqlc.arg(scenario_id),
  sqlc.arg(row_index)
)
ON CONFLICT (run_id, scenario_id) DO UPDATE SET
  row_index = excluded.row_index;

-- name: DeleteRunScenarioExampleValues :exec
DELETE FROM run_scenario_example_values
WHERE run_id = sqlc.arg(run_id)
  AND scenario_id = sqlc.arg(scenario_id);

-- name: CreateRunScenarioExampleValue :exec
INSERT INTO run_scenario_example_values (run_id, scenario_id, key, value)
VALUES (
  sqlc.arg(run_id),
  sqlc.arg(scenario_id),
  sqlc.arg(key),
  sqlc.arg(value)
);

-- name: UpsertRunStep :exec
INSERT INTO run_steps (
  run_id,
  scenario_id,
  step_id,
  ordinal,
  keyword,
  text,
  status,
  duration_nanos,
  error_code,
  error_message,
  error_details_json
) VALUES (
  sqlc.arg(run_id),
  sqlc.arg(scenario_id),
  sqlc.arg(step_id),
  sqlc.arg(ordinal),
  sqlc.arg(keyword),
  sqlc.arg(text),
  sqlc.arg(status),
  sqlc.arg(duration_nanos),
  sqlc.narg(error_code),
  sqlc.narg(error_message),
  sqlc.narg(error_details_json)
)
ON CONFLICT (run_id, scenario_id, step_id) DO UPDATE SET
  ordinal = excluded.ordinal,
  keyword = excluded.keyword,
  text = excluded.text,
  status = excluded.status,
  duration_nanos = excluded.duration_nanos,
  error_code = excluded.error_code,
  error_message = excluded.error_message,
  error_details_json = excluded.error_details_json;

-- name: DeleteRunStepAssertionFailures :exec
DELETE FROM run_step_assertion_failures
WHERE run_id = sqlc.arg(run_id)
  AND scenario_id = sqlc.arg(scenario_id)
  AND step_id = sqlc.arg(step_id);

-- name: CreateRunStepAssertionFailure :exec
INSERT INTO run_step_assertion_failures (
  run_id,
  scenario_id,
  step_id,
  ordinal,
  path,
  operator,
  expected_json,
  actual_json,
  message
) VALUES (
  sqlc.arg(run_id),
  sqlc.arg(scenario_id),
  sqlc.arg(step_id),
  sqlc.arg(ordinal),
  sqlc.narg(path),
  sqlc.narg(operator),
  sqlc.narg(expected_json),
  sqlc.narg(actual_json),
  sqlc.narg(message)
);

-- name: DeleteRunStepDataTable :exec
DELETE FROM run_step_data_table_headers
WHERE run_id = sqlc.arg(run_id)
  AND scenario_id = sqlc.arg(scenario_id)
  AND step_id = sqlc.arg(step_id);

-- name: CreateRunStepDataTableHeader :exec
INSERT INTO run_step_data_table_headers (run_id, scenario_id, step_id, ordinal, value)
VALUES (
  sqlc.arg(run_id),
  sqlc.arg(scenario_id),
  sqlc.arg(step_id),
  sqlc.arg(ordinal),
  sqlc.arg(value)
);

-- name: CreateRunStepDataTableRow :exec
INSERT INTO run_step_data_table_rows (run_id, scenario_id, step_id, row_index)
VALUES (
  sqlc.arg(run_id),
  sqlc.arg(scenario_id),
  sqlc.arg(step_id),
  sqlc.arg(row_index)
);

-- name: CreateRunStepDataTableCell :exec
INSERT INTO run_step_data_table_cells (run_id, scenario_id, step_id, row_index, cell_index, value)
VALUES (
  sqlc.arg(run_id),
  sqlc.arg(scenario_id),
  sqlc.arg(step_id),
  sqlc.arg(row_index),
  sqlc.arg(cell_index),
  sqlc.arg(value)
);

-- name: UpsertRunStepDocString :exec
INSERT INTO run_step_doc_strings (run_id, scenario_id, step_id, content_type, content)
VALUES (
  sqlc.arg(run_id),
  sqlc.arg(scenario_id),
  sqlc.arg(step_id),
  sqlc.narg(content_type),
  sqlc.arg(content)
)
ON CONFLICT (run_id, scenario_id, step_id) DO UPDATE SET
  content_type = excluded.content_type,
  content = excluded.content;

-- name: DeleteRunStepDocString :exec
DELETE FROM run_step_doc_strings
WHERE run_id = sqlc.arg(run_id)
  AND scenario_id = sqlc.arg(scenario_id)
  AND step_id = sqlc.arg(step_id);

-- name: UpsertRunHookResult :exec
INSERT INTO run_hook_results (
  run_id,
  hook_id,
  scenario_id,
  ordinal,
  hook_type,
  status,
  duration_nanos,
  error_code,
  error_message,
  error_details_json
) VALUES (
  sqlc.arg(run_id),
  sqlc.arg(hook_id),
  sqlc.narg(scenario_id),
  sqlc.arg(ordinal),
  sqlc.arg(hook_type),
  sqlc.arg(status),
  sqlc.arg(duration_nanos),
  sqlc.narg(error_code),
  sqlc.narg(error_message),
  sqlc.narg(error_details_json)
)
ON CONFLICT (run_id, hook_id) DO UPDATE SET
  scenario_id = excluded.scenario_id,
  ordinal = excluded.ordinal,
  hook_type = excluded.hook_type,
  status = excluded.status,
  duration_nanos = excluded.duration_nanos,
  error_code = excluded.error_code,
  error_message = excluded.error_message,
  error_details_json = excluded.error_details_json;

-- name: UpsertRunVariable :exec
INSERT INTO run_variables (run_id, variable_name, scope, value_json)
VALUES (
  sqlc.arg(run_id),
  sqlc.arg(variable_name),
  sqlc.arg(scope),
  sqlc.arg(value_json)
)
ON CONFLICT (run_id, variable_name, scope) DO UPDATE SET
  value_json = excluded.value_json;

-- name: ListRunVariables :many
SELECT *
FROM run_variables
WHERE run_id = sqlc.arg(run_id)
ORDER BY scope ASC, variable_name ASC;

-- name: AppendRunEvent :exec
INSERT INTO run_events (
  run_id,
  sequence,
  observed_at,
  event_type,
  terminal,
  metadata_json,
  payload_run_status,
  payload_scenario_id,
  payload_step_id,
  payload_hook_id,
  payload_diagnostic_code,
  payload_diagnostic_message,
  payload_diagnostic_details_json,
  payload_summary_total_scenarios,
  payload_summary_passed_scenarios,
  payload_summary_failed_scenarios,
  payload_summary_skipped_scenarios,
  payload_summary_duration_nanos
) VALUES (
  sqlc.arg(run_id),
  sqlc.arg(sequence),
  sqlc.arg(observed_at),
  sqlc.arg(event_type),
  sqlc.arg(terminal),
  sqlc.narg(metadata_json),
  sqlc.narg(payload_run_status),
  sqlc.narg(payload_scenario_id),
  sqlc.narg(payload_step_id),
  sqlc.narg(payload_hook_id),
  sqlc.narg(payload_diagnostic_code),
  sqlc.narg(payload_diagnostic_message),
  sqlc.narg(payload_diagnostic_details_json),
  sqlc.narg(payload_summary_total_scenarios),
  sqlc.narg(payload_summary_passed_scenarios),
  sqlc.narg(payload_summary_failed_scenarios),
  sqlc.narg(payload_summary_skipped_scenarios),
  sqlc.narg(payload_summary_duration_nanos)
);

-- name: ListRunEventsFromSequence :many
SELECT *
FROM run_events
WHERE run_id = sqlc.arg(run_id)
  AND sequence >= sqlc.arg(from_sequence)
ORDER BY sequence ASC;

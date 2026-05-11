BEGIN;

CREATE TABLE IF NOT EXISTS lobster_schema_meta (
  schema_key TEXT PRIMARY KEY,
  schema_version INTEGER NOT NULL,
  description TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  updated_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

INSERT INTO lobster_schema_meta (schema_key, schema_version, description)
VALUES ('lobster', 1, 'initial schema')
ON CONFLICT(schema_key) DO UPDATE SET
  schema_version = excluded.schema_version,
  description = excluded.description,
  updated_at = CURRENT_TIMESTAMP;

CREATE TABLE IF NOT EXISTS runs (
  run_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  profile_name TEXT NOT NULL,
  status INTEGER NOT NULL CHECK (status BETWEEN 0 AND 5),
  idempotency_key TEXT,
  selector_feature_path TEXT,
  selector_tag_expression TEXT,
  selector_profile_name TEXT,
  execution_mode INTEGER CHECK (execution_mode BETWEEN 0 AND 2),
  execution_run_timeout_nanos INTEGER,
  execution_step_timeout_nanos INTEGER,
  execution_soft_assert INTEGER CHECK (execution_soft_assert IN (0, 1)),
  execution_fail_fast INTEGER CHECK (execution_fail_fast IN (0, 1)),
  execution_keep_stack INTEGER CHECK (execution_keep_stack IN (0, 1)),
  feature_name TEXT,
  feature_description TEXT,
  summary_total_scenarios INTEGER NOT NULL DEFAULT 0,
  summary_passed_scenarios INTEGER NOT NULL DEFAULT 0,
  summary_failed_scenarios INTEGER NOT NULL DEFAULT 0,
  summary_skipped_scenarios INTEGER NOT NULL DEFAULT 0,
  summary_duration_nanos INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  started_at TEXT,
  ended_at TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS runs_workspace_idempotency_idx
  ON runs (workspace_id, idempotency_key)
  WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS runs_workspace_created_idx
  ON runs (workspace_id, created_at DESC, run_id DESC);

CREATE INDEX IF NOT EXISTS runs_workspace_status_created_idx
  ON runs (workspace_id, status, created_at DESC, run_id DESC);

CREATE INDEX IF NOT EXISTS runs_created_idx
  ON runs (created_at DESC, run_id DESC);

CREATE TABLE IF NOT EXISTS run_feature_tags (
  run_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  tag TEXT NOT NULL,
  PRIMARY KEY (run_id, ordinal),
  FOREIGN KEY (run_id) REFERENCES runs(run_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run_background_steps (
  run_id TEXT NOT NULL,
  step_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  keyword TEXT NOT NULL,
  text TEXT NOT NULL,
  doc_string_content_type TEXT,
  doc_string_content TEXT,
  PRIMARY KEY (run_id, step_id),
  UNIQUE (run_id, ordinal),
  FOREIGN KEY (run_id) REFERENCES runs(run_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run_background_step_data_table_headers (
  run_id TEXT NOT NULL,
  step_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  value TEXT NOT NULL,
  PRIMARY KEY (run_id, step_id, ordinal),
  FOREIGN KEY (run_id, step_id) REFERENCES run_background_steps(run_id, step_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run_background_step_data_table_rows (
  run_id TEXT NOT NULL,
  step_id TEXT NOT NULL,
  row_index INTEGER NOT NULL,
  PRIMARY KEY (run_id, step_id, row_index),
  FOREIGN KEY (run_id, step_id) REFERENCES run_background_steps(run_id, step_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run_background_step_data_table_cells (
  run_id TEXT NOT NULL,
  step_id TEXT NOT NULL,
  row_index INTEGER NOT NULL,
  cell_index INTEGER NOT NULL,
  value TEXT NOT NULL,
  PRIMARY KEY (run_id, step_id, row_index, cell_index),
  FOREIGN KEY (run_id, step_id, row_index) REFERENCES run_background_step_data_table_rows(run_id, step_id, row_index) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run_scenarios (
  run_id TEXT NOT NULL,
  scenario_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  status INTEGER NOT NULL CHECK (status BETWEEN 0 AND 6),
  duration_nanos INTEGER NOT NULL DEFAULT 0,
  feature_name TEXT,
  deterministic_feature_name TEXT,
  deterministic_scenario_name TEXT,
  deterministic_example_row_index INTEGER,
  deterministic_normalization_version TEXT,
  deterministic_stable_hash TEXT,
  PRIMARY KEY (run_id, scenario_id),
  UNIQUE (run_id, ordinal),
  FOREIGN KEY (run_id) REFERENCES runs(run_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS run_scenarios_run_status_idx
  ON run_scenarios (run_id, status, ordinal);

CREATE INDEX IF NOT EXISTS run_scenarios_det_key_idx
  ON run_scenarios (
    deterministic_feature_name,
    deterministic_scenario_name,
    deterministic_example_row_index,
    deterministic_normalization_version
  );

CREATE TABLE IF NOT EXISTS run_scenario_tags (
  run_id TEXT NOT NULL,
  scenario_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  tag TEXT NOT NULL,
  PRIMARY KEY (run_id, scenario_id, ordinal),
  FOREIGN KEY (run_id, scenario_id) REFERENCES run_scenarios(run_id, scenario_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run_scenario_examples (
  run_id TEXT NOT NULL,
  scenario_id TEXT NOT NULL,
  row_index INTEGER NOT NULL,
  PRIMARY KEY (run_id, scenario_id),
  FOREIGN KEY (run_id, scenario_id) REFERENCES run_scenarios(run_id, scenario_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run_scenario_example_values (
  run_id TEXT NOT NULL,
  scenario_id TEXT NOT NULL,
  key TEXT NOT NULL,
  value TEXT NOT NULL,
  PRIMARY KEY (run_id, scenario_id, key),
  FOREIGN KEY (run_id, scenario_id) REFERENCES run_scenario_examples(run_id, scenario_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run_steps (
  run_id TEXT NOT NULL,
  scenario_id TEXT NOT NULL,
  step_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  keyword TEXT NOT NULL,
  text TEXT NOT NULL,
  status INTEGER NOT NULL CHECK (status BETWEEN 0 AND 6),
  duration_nanos INTEGER NOT NULL DEFAULT 0,
  error_code INTEGER,
  error_message TEXT,
  error_details_json TEXT,
  PRIMARY KEY (run_id, scenario_id, step_id),
  UNIQUE (run_id, scenario_id, ordinal),
  FOREIGN KEY (run_id, scenario_id) REFERENCES run_scenarios(run_id, scenario_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS run_steps_lookup_idx
  ON run_steps (run_id, scenario_id, ordinal);

CREATE TABLE IF NOT EXISTS run_step_assertion_failures (
  run_id TEXT NOT NULL,
  scenario_id TEXT NOT NULL,
  step_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  path TEXT,
  operator TEXT,
  expected_json TEXT,
  actual_json TEXT,
  message TEXT,
  PRIMARY KEY (run_id, scenario_id, step_id, ordinal),
  FOREIGN KEY (run_id, scenario_id, step_id) REFERENCES run_steps(run_id, scenario_id, step_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run_step_data_table_headers (
  run_id TEXT NOT NULL,
  scenario_id TEXT NOT NULL,
  step_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  value TEXT NOT NULL,
  PRIMARY KEY (run_id, scenario_id, step_id, ordinal),
  FOREIGN KEY (run_id, scenario_id, step_id) REFERENCES run_steps(run_id, scenario_id, step_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run_step_data_table_rows (
  run_id TEXT NOT NULL,
  scenario_id TEXT NOT NULL,
  step_id TEXT NOT NULL,
  row_index INTEGER NOT NULL,
  PRIMARY KEY (run_id, scenario_id, step_id, row_index),
  FOREIGN KEY (run_id, scenario_id, step_id) REFERENCES run_steps(run_id, scenario_id, step_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run_step_data_table_cells (
  run_id TEXT NOT NULL,
  scenario_id TEXT NOT NULL,
  step_id TEXT NOT NULL,
  row_index INTEGER NOT NULL,
  cell_index INTEGER NOT NULL,
  value TEXT NOT NULL,
  PRIMARY KEY (run_id, scenario_id, step_id, row_index, cell_index),
  FOREIGN KEY (run_id, scenario_id, step_id, row_index) REFERENCES run_step_data_table_rows(run_id, scenario_id, step_id, row_index) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run_step_doc_strings (
  run_id TEXT NOT NULL,
  scenario_id TEXT NOT NULL,
  step_id TEXT NOT NULL,
  content_type TEXT,
  content TEXT NOT NULL,
  PRIMARY KEY (run_id, scenario_id, step_id),
  FOREIGN KEY (run_id, scenario_id, step_id) REFERENCES run_steps(run_id, scenario_id, step_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run_hook_results (
  run_id TEXT NOT NULL,
  hook_id TEXT NOT NULL,
  scenario_id TEXT,
  ordinal INTEGER NOT NULL,
  hook_type INTEGER NOT NULL CHECK (hook_type BETWEEN 0 AND 4),
  status INTEGER NOT NULL CHECK (status BETWEEN 0 AND 6),
  duration_nanos INTEGER NOT NULL DEFAULT 0,
  error_code INTEGER,
  error_message TEXT,
  error_details_json TEXT,
  PRIMARY KEY (run_id, hook_id),
  UNIQUE (run_id, scenario_id, ordinal),
  FOREIGN KEY (run_id) REFERENCES runs(run_id) ON DELETE CASCADE,
  FOREIGN KEY (run_id, scenario_id) REFERENCES run_scenarios(run_id, scenario_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS run_hook_results_run_idx
  ON run_hook_results (run_id, scenario_id, ordinal);

CREATE TABLE IF NOT EXISTS run_variables (
  run_id TEXT NOT NULL,
  variable_name TEXT NOT NULL,
  scope INTEGER NOT NULL CHECK (scope BETWEEN 0 AND 2),
  value_json TEXT NOT NULL,
  PRIMARY KEY (run_id, variable_name, scope),
  FOREIGN KEY (run_id) REFERENCES runs(run_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run_events (
  run_id TEXT NOT NULL,
  sequence INTEGER NOT NULL,
  observed_at TEXT NOT NULL,
  event_type INTEGER NOT NULL CHECK (event_type BETWEEN 0 AND 6),
  terminal INTEGER NOT NULL CHECK (terminal IN (0, 1)),
  metadata_json TEXT,
  payload_run_status INTEGER,
  payload_scenario_id TEXT,
  payload_step_id TEXT,
  payload_hook_id TEXT,
  payload_diagnostic_code INTEGER,
  payload_diagnostic_message TEXT,
  payload_diagnostic_details_json TEXT,
  payload_summary_total_scenarios INTEGER,
  payload_summary_passed_scenarios INTEGER,
  payload_summary_failed_scenarios INTEGER,
  payload_summary_skipped_scenarios INTEGER,
  payload_summary_duration_nanos INTEGER,
  PRIMARY KEY (run_id, sequence),
  FOREIGN KEY (run_id) REFERENCES runs(run_id) ON DELETE CASCADE,
  FOREIGN KEY (run_id, payload_scenario_id) REFERENCES run_scenarios(run_id, scenario_id) ON DELETE SET NULL,
  FOREIGN KEY (run_id, payload_scenario_id, payload_step_id) REFERENCES run_steps(run_id, scenario_id, step_id) ON DELETE SET NULL,
  FOREIGN KEY (run_id, payload_hook_id) REFERENCES run_hook_results(run_id, hook_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS run_events_stream_idx
  ON run_events (run_id, sequence);

CREATE INDEX IF NOT EXISTS run_events_observed_idx
  ON run_events (observed_at);

CREATE TABLE IF NOT EXISTS execution_plans (
  plan_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  profile_name TEXT NOT NULL,
  selector_feature_path TEXT,
  selector_tag_expression TEXT,
  selector_profile_name TEXT,
  estimated_duration_nanos INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS execution_plans_workspace_created_idx
  ON execution_plans (workspace_id, created_at DESC, plan_id DESC);

CREATE TABLE IF NOT EXISTS execution_plan_scenarios (
  plan_id TEXT NOT NULL,
  scenario_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  feature_name TEXT NOT NULL,
  scenario_name TEXT NOT NULL,
  estimated_duration_nanos INTEGER NOT NULL DEFAULT 0,
  deterministic_feature_name TEXT,
  deterministic_scenario_name TEXT,
  deterministic_example_row_index INTEGER,
  deterministic_normalization_version TEXT,
  deterministic_stable_hash TEXT,
  PRIMARY KEY (plan_id, scenario_id),
  UNIQUE (plan_id, ordinal),
  FOREIGN KEY (plan_id) REFERENCES execution_plans(plan_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS execution_plan_scenarios_det_key_idx
  ON execution_plan_scenarios (
    deterministic_feature_name,
    deterministic_scenario_name,
    deterministic_example_row_index,
    deterministic_normalization_version
  );

CREATE TABLE IF NOT EXISTS execution_plan_scenario_tags (
  plan_id TEXT NOT NULL,
  scenario_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  tag TEXT NOT NULL,
  PRIMARY KEY (plan_id, scenario_id, ordinal),
  FOREIGN KEY (plan_id, scenario_id) REFERENCES execution_plan_scenarios(plan_id, scenario_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS plan_artifacts (
  plan_id TEXT PRIMARY KEY,
  artifact_id TEXT NOT NULL UNIQUE,
  storage_path TEXT NOT NULL,
  envelope_schema_version TEXT,
  envelope_schema_revision INTEGER,
  envelope_media_type TEXT,
  envelope_json_export TEXT,
  envelope_created_at TEXT,
  envelope_payload_sha256 TEXT,
  envelope_compression_type INTEGER CHECK (envelope_compression_type BETWEEN 0 AND 3),
  envelope_signature TEXT,
  FOREIGN KEY (plan_id) REFERENCES execution_plans(plan_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS stacks (
  stack_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL UNIQUE,
  profile_name TEXT NOT NULL,
  project_name TEXT NOT NULL,
  status INTEGER NOT NULL CHECK (status BETWEEN 0 AND 5),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS stacks_status_updated_idx
  ON stacks (status, updated_at DESC);

CREATE TABLE IF NOT EXISTS stack_components (
  stack_id TEXT NOT NULL,
  name TEXT NOT NULL,
  image TEXT,
  container_id TEXT,
  status TEXT,
  health INTEGER NOT NULL CHECK (health BETWEEN 0 AND 4),
  updated_at TEXT NOT NULL,
  PRIMARY KEY (stack_id, name),
  FOREIGN KEY (stack_id) REFERENCES stacks(stack_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS stack_components_health_idx
  ON stack_components (health, updated_at DESC);

CREATE TABLE IF NOT EXISTS integration_adapters (
  adapter_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  state INTEGER NOT NULL CHECK (state BETWEEN 0 AND 4),
  config_extension_type_url TEXT,
  config_extension_value BLOB,
  state_extension_type_url TEXT,
  state_extension_value BLOB,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS integration_adapters_state_idx
  ON integration_adapters (state, updated_at DESC, adapter_id ASC);

CREATE TABLE IF NOT EXISTS integration_adapter_capabilities (
  adapter_id TEXT NOT NULL,
  name TEXT NOT NULL,
  enabled INTEGER NOT NULL CHECK (enabled IN (0, 1)),
  PRIMARY KEY (adapter_id, name),
  FOREIGN KEY (adapter_id) REFERENCES integration_adapters(adapter_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS integration_adapter_state_events (
  adapter_id TEXT NOT NULL,
  sequence INTEGER NOT NULL,
  previous_state INTEGER CHECK (previous_state BETWEEN 0 AND 4),
  next_state INTEGER NOT NULL CHECK (next_state BETWEEN 0 AND 4),
  reason TEXT,
  changed_at TEXT NOT NULL,
  PRIMARY KEY (adapter_id, sequence),
  FOREIGN KEY (adapter_id) REFERENCES integration_adapters(adapter_id) ON DELETE CASCADE
);

COMMIT;

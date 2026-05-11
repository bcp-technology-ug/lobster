BEGIN;

DROP TABLE IF EXISTS integration_adapter_state_events;
DROP TABLE IF EXISTS integration_adapter_capabilities;
DROP TABLE IF EXISTS integration_adapters;

DROP TABLE IF EXISTS stack_components;
DROP TABLE IF EXISTS stacks;

DROP TABLE IF EXISTS plan_artifacts;
DROP TABLE IF EXISTS execution_plan_scenario_tags;
DROP TABLE IF EXISTS execution_plan_scenarios;
DROP TABLE IF EXISTS execution_plans;

DROP TABLE IF EXISTS run_events;
DROP TABLE IF EXISTS run_variables;
DROP TABLE IF EXISTS run_hook_results;
DROP TABLE IF EXISTS run_step_doc_strings;
DROP TABLE IF EXISTS run_step_data_table_cells;
DROP TABLE IF EXISTS run_step_data_table_rows;
DROP TABLE IF EXISTS run_step_data_table_headers;
DROP TABLE IF EXISTS run_step_assertion_failures;
DROP TABLE IF EXISTS run_steps;
DROP TABLE IF EXISTS run_scenario_example_values;
DROP TABLE IF EXISTS run_scenario_examples;
DROP TABLE IF EXISTS run_scenario_tags;
DROP TABLE IF EXISTS run_scenarios;
DROP TABLE IF EXISTS run_background_step_data_table_cells;
DROP TABLE IF EXISTS run_background_step_data_table_rows;
DROP TABLE IF EXISTS run_background_step_data_table_headers;
DROP TABLE IF EXISTS run_background_steps;
DROP TABLE IF EXISTS run_feature_tags;
DROP TABLE IF EXISTS runs;

DROP TABLE IF EXISTS lobster_schema_meta;

COMMIT;

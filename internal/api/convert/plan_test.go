package convert

import (
	"testing"

	planstore "github.com/bcp-technology/lobster/gen/sqlc/plan"
)

func TestExecutionPlanFromDB_basic(t *testing.T) {
	t.Parallel()
	now := NowDB()
	row := planstore.ExecutionPlan{
		PlanID:      "plan-001",
		WorkspaceID: "ws-001",
		ProfileName: "default",
		CreatedAt:   now,
	}
	plan := ExecutionPlanFromDB(row)
	if plan.PlanId != "plan-001" {
		t.Errorf("PlanId: got %q want %q", plan.PlanId, "plan-001")
	}
	if plan.WorkspaceId != "ws-001" {
		t.Errorf("WorkspaceId: got %q want %q", plan.WorkspaceId, "ws-001")
	}
	if plan.ProfileName != "default" {
		t.Errorf("ProfileName: got %q want %q", plan.ProfileName, "default")
	}
	if plan.CreatedAt == nil {
		t.Error("CreatedAt should not be nil")
	}
}

func TestExecutionPlanFromDB_withSelector(t *testing.T) {
	t.Parallel()
	now := NowDB()
	fp := "features/**/*.feature"
	tag := "@smoke"
	row := planstore.ExecutionPlan{
		PlanID:                "plan-002",
		WorkspaceID:           "ws-002",
		ProfileName:           "ci",
		CreatedAt:             now,
		SelectorFeaturePath:   &fp,
		SelectorTagExpression: &tag,
	}
	plan := ExecutionPlanFromDB(row)
	if plan.Selector == nil {
		t.Fatal("Selector should not be nil")
	}
	if plan.Selector.FeaturePath != fp {
		t.Errorf("FeaturePath: got %q want %q", plan.Selector.FeaturePath, fp)
	}
	if plan.Selector.TagExpression != tag {
		t.Errorf("TagExpression: got %q want %q", plan.Selector.TagExpression, tag)
	}
}

func TestExecutionPlanFromDB_estimatedDuration(t *testing.T) {
	t.Parallel()
	now := NowDB()
	row := planstore.ExecutionPlan{
		PlanID:                 "plan-003",
		WorkspaceID:            "ws-003",
		ProfileName:            "default",
		CreatedAt:              now,
		EstimatedDurationNanos: 2_000_000_000, // 2s
	}
	plan := ExecutionPlanFromDB(row)
	if plan.EstimatedDuration == nil {
		t.Fatal("EstimatedDuration should not be nil")
	}
	if plan.EstimatedDuration.AsDuration().Seconds() != 2 {
		t.Errorf("EstimatedDuration: got %v want 2s", plan.EstimatedDuration.AsDuration())
	}
}

func TestScenarioPlanFromDB_basic(t *testing.T) {
	t.Parallel()
	row := planstore.ExecutionPlanScenario{
		PlanID:       "plan-001",
		ScenarioID:   "scen-001",
		Ordinal:      1,
		FeatureName:  "Login Feature",
		ScenarioName: "User can log in",
	}
	sp := ScenarioPlanFromDB(row)
	if sp.ScenarioId != "scen-001" {
		t.Errorf("ScenarioId: got %q want %q", sp.ScenarioId, "scen-001")
	}
	if sp.FeatureName != "Login Feature" {
		t.Errorf("FeatureName: got %q want %q", sp.FeatureName, "Login Feature")
	}
	if sp.ScenarioName != "User can log in" {
		t.Errorf("ScenarioName: got %q want %q", sp.ScenarioName, "User can log in")
	}
}

func TestScenarioPlanFromDB_deterministicKey(t *testing.T) {
	t.Parallel()
	featName := "Login"
	scenName := "Successful login"
	hash := "abc123"
	row := planstore.ExecutionPlanScenario{
		PlanID:                    "plan-004",
		ScenarioID:                "scen-004",
		FeatureName:               "Login",
		ScenarioName:              "Successful login",
		DeterministicFeatureName:  &featName,
		DeterministicScenarioName: &scenName,
		DeterministicStableHash:   &hash,
	}
	sp := ScenarioPlanFromDB(row)
	if sp.DeterministicKey == nil {
		t.Fatal("DeterministicKey should not be nil")
	}
	if sp.DeterministicKey.FeatureName != "Login" {
		t.Errorf("FeatureName: got %q want %q", sp.DeterministicKey.FeatureName, "Login")
	}
	if sp.DeterministicKey.StableHash != "abc123" {
		t.Errorf("StableHash: got %q want %q", sp.DeterministicKey.StableHash, "abc123")
	}
}

package convert

import (
	"testing"
	"time"

	commonv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/common"
	runv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/run"
	runstore "github.com/bcp-technology-ug/lobster/gen/sqlc/run"
)

func TestRunFromDB_basic(t *testing.T) {
	t.Parallel()
	now := NowDB()
	row := runstore.Run{
		RunID:       "run-001",
		WorkspaceID: "ws-001",
		ProfileName: "default",
		Status:      int64(commonv1.RunStatus_RUN_STATUS_PASSED),
		CreatedAt:   now,
	}
	run := RunFromDB(row)
	if run.RunId != "run-001" {
		t.Errorf("RunId: got %q want %q", run.RunId, "run-001")
	}
	if run.WorkspaceId != "ws-001" {
		t.Errorf("WorkspaceId: got %q want %q", run.WorkspaceId, "ws-001")
	}
	if run.ProfileName != "default" {
		t.Errorf("ProfileName: got %q want %q", run.ProfileName, "default")
	}
	if run.Status != commonv1.RunStatus_RUN_STATUS_PASSED {
		t.Errorf("Status: got %v want PASSED", run.Status)
	}
	if run.CreatedAt == nil {
		t.Error("CreatedAt should not be nil")
	}
}

func TestRunFromDB_withFeature(t *testing.T) {
	t.Parallel()
	now := NowDB()
	name := "login"
	desc := "login feature"
	row := runstore.Run{
		RunID:              "run-002",
		WorkspaceID:        "ws-002",
		ProfileName:        "ci",
		Status:             int64(commonv1.RunStatus_RUN_STATUS_FAILED),
		CreatedAt:          now,
		FeatureName:        &name,
		FeatureDescription: &desc,
	}
	run := RunFromDB(row)
	if run.Feature == nil {
		t.Fatal("Feature should not be nil when FeatureName is set")
	}
	if run.Feature.Name != "login" {
		t.Errorf("Feature.Name: got %q want %q", run.Feature.Name, "login")
	}
	if run.Feature.Description != "login feature" {
		t.Errorf("Feature.Description: got %q want %q", run.Feature.Description, "login feature")
	}
}

func TestRunFromDB_summaryFields(t *testing.T) {
	t.Parallel()
	now := NowDB()
	row := runstore.Run{
		RunID:                   "run-003",
		WorkspaceID:             "ws-003",
		ProfileName:             "default",
		Status:                  int64(commonv1.RunStatus_RUN_STATUS_PASSED),
		CreatedAt:               now,
		SummaryTotalScenarios:   5,
		SummaryPassedScenarios:  4,
		SummaryFailedScenarios:  1,
		SummarySkippedScenarios: 0,
		SummaryDurationNanos:    int64(2 * time.Second),
	}
	run := RunFromDB(row)
	if run.Summary == nil {
		t.Fatal("Summary should not be nil")
	}
	if run.Summary.TotalScenarios != 5 {
		t.Errorf("TotalScenarios: got %d want 5", run.Summary.TotalScenarios)
	}
	if run.Summary.PassedScenarios != 4 {
		t.Errorf("PassedScenarios: got %d want 4", run.Summary.PassedScenarios)
	}
}

func TestRunEventFromDB_runStatus(t *testing.T) {
	t.Parallel()
	now := NowDB()
	status := int64(commonv1.RunStatus_RUN_STATUS_RUNNING)
	e := runstore.RunEvent{
		RunID:            "run-001",
		Sequence:         1,
		ObservedAt:       now,
		EventType:        int64(runv1.RunEventType_RUN_EVENT_TYPE_RUN_STATUS),
		Terminal:         0,
		PayloadRunStatus: &status,
	}
	evt := RunEventFromDB(e)
	if evt.RunId != "run-001" {
		t.Errorf("RunId: got %q want %q", evt.RunId, "run-001")
	}
	if evt.Sequence != 1 {
		t.Errorf("Sequence: got %d want 1", evt.Sequence)
	}
	if evt.Terminal {
		t.Error("Terminal should be false")
	}
	payload, ok := evt.Payload.(*runv1.RunEvent_RunStatus)
	if !ok {
		t.Fatalf("Payload type: got %T want *RunEvent_RunStatus", evt.Payload)
	}
	if payload.RunStatus != commonv1.RunStatus_RUN_STATUS_RUNNING {
		t.Errorf("RunStatus: got %v want RUNNING", payload.RunStatus)
	}
}

func TestRunEventFromDB_summary(t *testing.T) {
	t.Parallel()
	now := NowDB()
	total := int64(10)
	passed := int64(8)
	failed := int64(2)
	skipped := int64(0)
	durNs := int64(5_000_000_000) // 5s
	e := runstore.RunEvent{
		RunID:                          "run-002",
		Sequence:                       10,
		ObservedAt:                     now,
		EventType:                      int64(runv1.RunEventType_RUN_EVENT_TYPE_SUMMARY),
		Terminal:                       0,
		PayloadSummaryTotalScenarios:   &total,
		PayloadSummaryPassedScenarios:  &passed,
		PayloadSummaryFailedScenarios:  &failed,
		PayloadSummarySkippedScenarios: &skipped,
		PayloadSummaryDurationNanos:    &durNs,
	}
	evt := RunEventFromDB(e)
	payload, ok := evt.Payload.(*runv1.RunEvent_Summary)
	if !ok {
		t.Fatalf("Payload type: got %T want *RunEvent_Summary", evt.Payload)
	}
	if payload.Summary.TotalScenarios != 10 {
		t.Errorf("TotalScenarios: got %d want 10", payload.Summary.TotalScenarios)
	}
	if payload.Summary.PassedScenarios != 8 {
		t.Errorf("PassedScenarios: got %d want 8", payload.Summary.PassedScenarios)
	}
}

func TestRunEventFromDB_terminal(t *testing.T) {
	t.Parallel()
	now := NowDB()
	status := int64(commonv1.RunStatus_RUN_STATUS_PASSED)
	e := runstore.RunEvent{
		RunID:            "run-003",
		Sequence:         99,
		ObservedAt:       now,
		EventType:        int64(runv1.RunEventType_RUN_EVENT_TYPE_RUN_STATUS),
		Terminal:         1,
		PayloadRunStatus: &status,
	}
	evt := RunEventFromDB(e)
	if !evt.Terminal {
		t.Error("Terminal should be true when Terminal==1 in DB row")
	}
}

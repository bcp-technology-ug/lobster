package reports_test

import (
	"testing"
	"time"

	"github.com/bcp-technology-ug/lobster/internal/reports"
)

func TestStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status reports.Status
		want   string
	}{
		{reports.StatusPassed, "passed"},
		{reports.StatusFailed, "failed"},
		{reports.StatusSkipped, "skipped"},
		{reports.StatusUndefined, "undefined"},
		{reports.StatusPending, "pending"},
		{reports.StatusUnknown, "unknown"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()

			got := tc.status.String()
			if got != tc.want {
				t.Errorf("Status(%d).String() = %q, want %q", int(tc.status), got, tc.want)
			}
		})
	}
}

func TestRunResult_Finalise_AllPassed(t *testing.T) {
	t.Parallel()

	rr := &reports.RunResult{
		Scenarios: []*reports.ScenarioResult{
			{Status: reports.StatusPassed},
			{Status: reports.StatusPassed},
		},
	}
	rr.Finalise()

	if rr.Status != reports.StatusPassed {
		t.Errorf("Status = %v, want passed", rr.Status)
	}
	if rr.Total != 2 {
		t.Errorf("Total = %d, want 2", rr.Total)
	}
	if rr.Passed != 2 {
		t.Errorf("Passed = %d, want 2", rr.Passed)
	}
	if rr.Failed != 0 {
		t.Errorf("Failed = %d, want 0", rr.Failed)
	}
}

func TestRunResult_Finalise_OneFailed(t *testing.T) {
	t.Parallel()

	rr := &reports.RunResult{
		Scenarios: []*reports.ScenarioResult{
			{Status: reports.StatusPassed},
			{Status: reports.StatusFailed},
			{Status: reports.StatusPassed},
		},
	}
	rr.Finalise()

	if rr.Status != reports.StatusFailed {
		t.Errorf("Status = %v, want failed", rr.Status)
	}
	if rr.Total != 3 {
		t.Errorf("Total = %d, want 3", rr.Total)
	}
	if rr.Passed != 2 {
		t.Errorf("Passed = %d, want 2", rr.Passed)
	}
	if rr.Failed != 1 {
		t.Errorf("Failed = %d, want 1", rr.Failed)
	}
}

func TestRunResult_Finalise_Undefined(t *testing.T) {
	t.Parallel()

	rr := &reports.RunResult{
		Scenarios: []*reports.ScenarioResult{
			{Status: reports.StatusPassed},
			{Status: reports.StatusUndefined},
		},
	}
	rr.Finalise()

	// Undefined steps count as failures.
	if rr.Status != reports.StatusFailed {
		t.Errorf("Status = %v, want failed (undefined should fail the run)", rr.Status)
	}
	if rr.Undefined != 1 {
		t.Errorf("Undefined = %d, want 1", rr.Undefined)
	}
}

func TestRunResult_Finalise_Empty(t *testing.T) {
	t.Parallel()

	rr := &reports.RunResult{}
	rr.Finalise()

	if rr.Total != 0 {
		t.Errorf("Total = %d, want 0", rr.Total)
	}
	// With no scenarios, status stays unknown (no evidence of pass or fail).
	if rr.Status != reports.StatusUnknown {
		t.Errorf("Status = %v, want unknown for empty run", rr.Status)
	}
}

func TestRunResult_Finalise_Skipped(t *testing.T) {
	t.Parallel()

	rr := &reports.RunResult{
		Scenarios: []*reports.ScenarioResult{
			{Status: reports.StatusPassed},
			{Status: reports.StatusSkipped},
		},
	}
	rr.Finalise()

	if rr.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", rr.Skipped)
	}
	// Skipped alone should not mark run as failed.
	if rr.Status != reports.StatusPassed {
		t.Errorf("Status = %v, want passed (skipped does not fail)", rr.Status)
	}
}

func TestScenarioResult_Defaults(t *testing.T) {
	t.Parallel()

	sc := &reports.ScenarioResult{
		DeterministicID: "abc123",
		Name:            "My scenario",
		Tags:            []string{"@smoke"},
		Duration:        200 * time.Millisecond,
	}

	if sc.DeterministicID != "abc123" {
		t.Errorf("DeterministicID = %q", sc.DeterministicID)
	}
	if sc.Status != reports.StatusUnknown {
		t.Errorf("default Status = %v, want unknown", sc.Status)
	}
}

package ui

import "github.com/bcp-technology/lobster/internal/reports"

// CaptureReporter is a no-op reporter that stores the final RunResult so
// callers can inspect it after a run (e.g. to print undefined steps).
type CaptureReporter struct {
	Result *reports.RunResult
}

func (c *CaptureReporter) RunStarted(_ *reports.RunResult)                               {}
func (c *CaptureReporter) ScenarioStarted(_ *reports.ScenarioResult)                     {}
func (c *CaptureReporter) StepFinished(_ *reports.ScenarioResult, _ *reports.StepResult) {}
func (c *CaptureReporter) ScenarioFinished(_ *reports.ScenarioResult)                    {}
func (c *CaptureReporter) RunFinished(r *reports.RunResult)                              { c.Result = r }

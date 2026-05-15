// Package reports provides result types and reporter implementations for
// Lobster run output. The Reporter interface is satisfied by console, JSON,
// and JUnit XML reporters.
package reports

import (
	"time"
)

// Status represents the execution outcome of a step, scenario, or run.
type Status int

const (
	StatusUnknown   Status = iota
	StatusPassed           // all steps passed
	StatusFailed           // at least one step failed
	StatusSkipped          // step was skipped due to earlier failure
	StatusUndefined        // no matching step definition found
	StatusPending          // step definition present but not yet implemented
)

func (s Status) String() string {
	switch s {
	case StatusPassed:
		return "passed"
	case StatusFailed:
		return "failed"
	case StatusSkipped:
		return "skipped"
	case StatusUndefined:
		return "undefined"
	case StatusPending:
		return "pending"
	default:
		return "unknown"
	}
}

// StepResult holds the outcome of a single step execution.
type StepResult struct {
	Keyword  string
	Text     string
	Status   Status
	Err      error
	Duration time.Duration
}

// ScenarioResult holds the outcome of a single scenario execution.
type ScenarioResult struct {
	// DeterministicID matches parser.Scenario.DeterministicID.
	DeterministicID string
	Name            string
	FeatureName     string
	FeatureURI      string
	Tags            []string
	Steps           []*StepResult
	Status          Status
	Duration        time.Duration
	// Err is set when the scenario fails with a non-step error (e.g. hook failure).
	Err error
}

// RunResult holds the aggregate outcome of a complete Lobster run.
type RunResult struct {
	RunID     string
	Profile   string
	StartedAt time.Time
	Duration  time.Duration
	Scenarios []*ScenarioResult
	Status    Status

	// Counters (derived from Scenarios on Finalise).
	Total     int
	Passed    int
	Failed    int
	Skipped   int
	Undefined int
}

// Finalise computes aggregate counters and the top-level Status from the
// scenario results. Call once after all scenarios have been added.
func (r *RunResult) Finalise() {
	r.Total = len(r.Scenarios)
	r.Passed = 0
	r.Failed = 0
	r.Skipped = 0
	r.Undefined = 0
	for _, sc := range r.Scenarios {
		switch sc.Status {
		case StatusPassed:
			r.Passed++
		case StatusFailed:
			r.Failed++
		case StatusSkipped:
			r.Skipped++
		case StatusUndefined:
			r.Undefined++
		}
	}
	if r.Failed > 0 || r.Undefined > 0 {
		r.Status = StatusFailed
	} else if r.Passed > 0 {
		r.Status = StatusPassed
	} else {
		r.Status = StatusUnknown
	}
}

// Reporter is the observer interface for run lifecycle events. Each reporter
// receives the same events and is responsible for its own output.
type Reporter interface {
	// RunStarted is called once before any scenario runs.
	RunStarted(result *RunResult)
	// ScenarioStarted is called before each scenario's steps execute.
	ScenarioStarted(scenario *ScenarioResult)
	// StepFinished is called after each step completes.
	StepFinished(scenario *ScenarioResult, step *StepResult)
	// ScenarioFinished is called after all steps in a scenario complete.
	ScenarioFinished(scenario *ScenarioResult)
	// RunFinished is called once after all scenarios complete.
	// result.Finalise() has already been called before this event.
	RunFinished(result *RunResult)
}

// MultiReporter fans out all events to a list of child reporters.
type MultiReporter struct {
	reporters []Reporter
}

// NewMultiReporter creates a MultiReporter from the supplied reporters.
func NewMultiReporter(reporters ...Reporter) *MultiReporter {
	return &MultiReporter{reporters: reporters}
}

func (m *MultiReporter) RunStarted(r *RunResult) {
	for _, rep := range m.reporters {
		rep.RunStarted(r)
	}
}
func (m *MultiReporter) ScenarioStarted(sc *ScenarioResult) {
	for _, rep := range m.reporters {
		rep.ScenarioStarted(sc)
	}
}
func (m *MultiReporter) StepFinished(sc *ScenarioResult, step *StepResult) {
	for _, rep := range m.reporters {
		rep.StepFinished(sc, step)
	}
}
func (m *MultiReporter) ScenarioFinished(sc *ScenarioResult) {
	for _, rep := range m.reporters {
		rep.ScenarioFinished(sc)
	}
}
func (m *MultiReporter) RunFinished(r *RunResult) {
	for _, rep := range m.reporters {
		rep.RunFinished(r)
	}
}

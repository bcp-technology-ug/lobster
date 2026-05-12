package reports

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// jsonRunResult mirrors RunResult for JSON serialisation with snake_case keys.
type jsonRunResult struct {
	RunID     string                `json:"run_id"`
	Profile   string                `json:"profile"`
	Status    string                `json:"status"`
	StartedAt time.Time             `json:"started_at"`
	Duration  float64               `json:"duration_ms"`
	Total     int                   `json:"total"`
	Passed    int                   `json:"passed"`
	Failed    int                   `json:"failed"`
	Skipped   int                   `json:"skipped"`
	Undefined int                   `json:"undefined"`
	Scenarios []*jsonScenarioResult `json:"scenarios"`
}

type jsonScenarioResult struct {
	DeterministicID string            `json:"id"`
	Name            string            `json:"name"`
	Feature         string            `json:"feature"`
	FeatureURI      string            `json:"feature_uri"`
	Tags            []string          `json:"tags"`
	Status          string            `json:"status"`
	Duration        float64           `json:"duration_ms"`
	Steps           []*jsonStepResult `json:"steps"`
	Error           string            `json:"error,omitempty"`
}

type jsonStepResult struct {
	Keyword  string  `json:"keyword"`
	Text     string  `json:"text"`
	Status   string  `json:"status"`
	Duration float64 `json:"duration_ms"`
	Error    string  `json:"error,omitempty"`
}

// JSONReporter writes a structured JSON report to a file after the run
// completes. It is a no-op for all intermediate events.
type JSONReporter struct {
	// Path is the output file path. Parent directories are created if missing.
	Path string
}

// NewJSONReporter creates a JSONReporter writing to path.
func NewJSONReporter(path string) *JSONReporter {
	return &JSONReporter{Path: path}
}

func (j *JSONReporter) RunStarted(_ *RunResult)                       {}
func (j *JSONReporter) ScenarioStarted(_ *ScenarioResult)             {}
func (j *JSONReporter) StepFinished(_ *ScenarioResult, _ *StepResult) {}
func (j *JSONReporter) ScenarioFinished(_ *ScenarioResult)            {}

// RunFinished serialises the result to j.Path.
func (j *JSONReporter) RunFinished(r *RunResult) {
	if j.Path == "" {
		return
	}
	if err := j.write(r); err != nil {
		fmt.Fprintf(os.Stderr, "lobster: json reporter: %v\n", err)
	}
}

func (j *JSONReporter) write(r *RunResult) error {
	if err := os.MkdirAll(filepath.Dir(j.Path), 0o755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}
	f, err := os.Create(j.Path)
	if err != nil {
		return fmt.Errorf("create json report %q: %w", j.Path, err)
	}
	defer f.Close()

	jr := toJSONResult(r)
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(jr)
}

func toJSONResult(r *RunResult) *jsonRunResult {
	jr := &jsonRunResult{
		RunID:     r.RunID,
		Profile:   r.Profile,
		Status:    r.Status.String(),
		StartedAt: r.StartedAt,
		Duration:  float64(r.Duration) / float64(time.Millisecond),
		Total:     r.Total,
		Passed:    r.Passed,
		Failed:    r.Failed,
		Skipped:   r.Skipped,
		Undefined: r.Undefined,
		Scenarios: make([]*jsonScenarioResult, len(r.Scenarios)),
	}
	for i, sc := range r.Scenarios {
		jsc := &jsonScenarioResult{
			DeterministicID: sc.DeterministicID,
			Name:            sc.Name,
			Feature:         sc.FeatureName,
			FeatureURI:      sc.FeatureURI,
			Tags:            sc.Tags,
			Status:          sc.Status.String(),
			Duration:        float64(sc.Duration) / float64(time.Millisecond),
			Steps:           make([]*jsonStepResult, len(sc.Steps)),
		}
		if sc.Err != nil {
			jsc.Error = sc.Err.Error()
		}
		for k, step := range sc.Steps {
			jst := &jsonStepResult{
				Keyword:  step.Keyword,
				Text:     step.Text,
				Status:   step.Status.String(),
				Duration: float64(step.Duration) / float64(time.Millisecond),
			}
			if step.Err != nil {
				jst.Error = step.Err.Error()
			}
			jsc.Steps[k] = jst
		}
		jr.Scenarios[i] = jsc
	}
	return jr
}

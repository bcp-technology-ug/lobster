package reports

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// JUnit XML schema compatible with Jenkins, CircleCI, GitLab CI, and GitHub Actions.

type junitTestsuites struct {
	XMLName    xml.Name          `xml:"testsuites"`
	Name       string            `xml:"name,attr"`
	Tests      int               `xml:"tests,attr"`
	Failures   int               `xml:"failures,attr"`
	Errors     int               `xml:"errors,attr"`
	Skipped    int               `xml:"skipped,attr"`
	Time       string            `xml:"time,attr"`
	Testsuites []*junitTestsuite `xml:"testsuite"`
}

type junitTestsuite struct {
	XMLName   xml.Name         `xml:"testsuite"`
	Name      string           `xml:"name,attr"`
	Tests     int              `xml:"tests,attr"`
	Failures  int              `xml:"failures,attr"`
	Errors    int              `xml:"errors,attr"`
	Skipped   int              `xml:"skipped,attr"`
	Time      string           `xml:"time,attr"`
	Timestamp string           `xml:"timestamp,attr"`
	Testcases []*junitTestcase `xml:"testcase"`
}

type junitTestcase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Skipped   *junitSkipped `xml:"skipped,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

type junitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

// JUnitReporter writes a JUnit XML report file after the run completes.
type JUnitReporter struct {
	Path string
}

// NewJUnitReporter creates a JUnitReporter writing to path.
func NewJUnitReporter(path string) *JUnitReporter {
	return &JUnitReporter{Path: path}
}

func (j *JUnitReporter) RunStarted(_ *RunResult)                       {}
func (j *JUnitReporter) ScenarioStarted(_ *ScenarioResult)             {}
func (j *JUnitReporter) StepFinished(_ *ScenarioResult, _ *StepResult) {}
func (j *JUnitReporter) ScenarioFinished(_ *ScenarioResult)            {}

// RunFinished writes the JUnit XML report to j.Path.
func (j *JUnitReporter) RunFinished(r *RunResult) {
	if j.Path == "" {
		return
	}
	if err := j.write(r); err != nil {
		fmt.Fprintf(os.Stderr, "lobster: junit reporter: %v\n", err)
	}
}

func (j *JUnitReporter) write(r *RunResult) error {
	if err := os.MkdirAll(filepath.Dir(j.Path), 0o755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}
	f, err := os.Create(j.Path)
	if err != nil {
		return fmt.Errorf("create junit report %q: %w", j.Path, err)
	}
	defer f.Close()

	suites := buildJUnitSuites(r)
	f.WriteString(xml.Header) //nolint:errcheck,gosec
	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	if err := enc.Encode(suites); err != nil {
		return fmt.Errorf("encode junit xml: %w", err)
	}
	return enc.Close()
}

func buildJUnitSuites(r *RunResult) *junitTestsuites {
	// Group scenarios by feature name.
	type group struct {
		name      string
		uri       string
		scenarios []*ScenarioResult
	}
	order := []string{}
	groups := map[string]*group{}
	for _, sc := range r.Scenarios {
		key := sc.FeatureURI
		if _, ok := groups[key]; !ok {
			order = append(order, key)
			groups[key] = &group{name: sc.FeatureName, uri: sc.FeatureURI}
		}
		groups[key].scenarios = append(groups[key].scenarios, sc)
	}

	suites := &junitTestsuites{
		Name:     r.RunID,
		Tests:    r.Total,
		Failures: r.Failed,
		Time:     formatSeconds(r.Duration),
	}

	for _, key := range order {
		g := groups[key]
		suite := &junitTestsuite{
			Name:      g.name,
			Tests:     len(g.scenarios),
			Timestamp: r.StartedAt.UTC().Format(time.RFC3339),
			Time:      formatSeconds(r.Duration),
		}
		for _, sc := range g.scenarios {
			tc := &junitTestcase{
				Name:      sc.Name,
				Classname: g.name,
				Time:      formatSeconds(sc.Duration),
			}
			switch sc.Status {
			case StatusFailed:
				suite.Failures++
				body := buildFailureBody(sc)
				msg := "scenario failed"
				if sc.Err != nil {
					msg = sc.Err.Error()
				}
				tc.Failure = &junitFailure{
					Message: msg,
					Type:    "failure",
					Body:    body,
				}
			case StatusSkipped, StatusUndefined:
				suite.Skipped++
				tc.Skipped = &junitSkipped{Message: sc.Status.String()}
			}
			suite.Testcases = append(suite.Testcases, tc)
		}
		suites.Skipped += suite.Skipped
		suites.Testsuites = append(suites.Testsuites, suite)
	}
	return suites
}

func buildFailureBody(sc *ScenarioResult) string {
	var sb []byte
	for _, step := range sc.Steps {
		if step.Status == StatusFailed && step.Err != nil {
			sb = append(sb, fmt.Sprintf("%s%s\n  → %s\n", step.Keyword, step.Text, step.Err.Error())...)
		} else if step.Status == StatusUndefined {
			sb = append(sb, fmt.Sprintf("%s%s\n  → undefined step\n", step.Keyword, step.Text)...)
		}
	}
	return string(sb)
}

func formatSeconds(d time.Duration) string {
	return fmt.Sprintf("%.3f", d.Seconds())
}

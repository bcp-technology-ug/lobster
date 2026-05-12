package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/bcp-technology/lobster/internal/reports"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- TUI reporter ---------------------------------------------------------

// TUIReporter implements reports.Reporter and forwards events to a Bubbletea
// program via Send(). Create with NewTUIReporter and pass to runner.WithReporter.
type TUIReporter struct {
	program *tea.Program
}

// NewTUIReporter creates a TUIReporter wired to the given program.
func NewTUIReporter(p *tea.Program) *TUIReporter {
	return &TUIReporter{program: p}
}

func (r *TUIReporter) RunStarted(result *reports.RunResult) {
	r.program.Send(RunStartedMsg{RunID: result.RunID, StartedAt: result.StartedAt})
}

func (r *TUIReporter) ScenarioStarted(sc *reports.ScenarioResult) {
	r.program.Send(ScenarioStartedMsg{
		Name:    sc.Name,
		Feature: sc.FeatureName,
		Tags:    sc.Tags,
	})
}

func (r *TUIReporter) StepFinished(sc *reports.ScenarioResult, step *reports.StepResult) {
	r.program.Send(StepFinishedMsg{
		ScenarioName: sc.Name,
		Keyword:      step.Keyword,
		Text:         step.Text,
		Status:       step.Status,
		Err:          step.Err,
	})
}

func (r *TUIReporter) ScenarioFinished(sc *reports.ScenarioResult) {
	r.program.Send(ScenarioFinishedMsg{
		Name:     sc.Name,
		Feature:  sc.FeatureName,
		Tags:     sc.Tags,
		Status:   sc.Status,
		Duration: sc.Duration,
		Err:      sc.Err,
	})
}

func (r *TUIReporter) RunFinished(result *reports.RunResult) {
	r.program.Send(RunFinishedMsg{
		RunID:    result.RunID,
		Total:    result.Total,
		Passed:   result.Passed,
		Failed:   result.Failed,
		Skipped:  result.Skipped,
		Duration: result.Duration,
		Status:   result.Status,
	})
}

// --- Message types ---------------------------------------------------------

// RunStartedMsg is sent when the run begins.
type RunStartedMsg struct {
	RunID     string
	StartedAt time.Time
}

// ScenarioStartedMsg is sent when a scenario begins executing.
type ScenarioStartedMsg struct {
	Name    string
	Feature string
	Tags    []string
}

// StepFinishedMsg is sent when a single step completes.
type StepFinishedMsg struct {
	ScenarioName string
	Keyword      string
	Text         string
	Status       reports.Status
	Err          error
}

// ScenarioFinishedMsg is sent when a scenario finishes.
type ScenarioFinishedMsg struct {
	Name     string
	Feature  string
	Tags     []string
	Status   reports.Status
	Duration time.Duration
	Err      error
}

// RunFinishedMsg is sent when the entire run finishes.
type RunFinishedMsg struct {
	RunID    string
	Total    int
	Passed   int
	Failed   int
	Skipped  int
	Duration time.Duration
	Status   reports.Status
}

// RunnerErrMsg is sent when RunSync returns an error.
type RunnerErrMsg struct{ Err error }

// --- Internal model types --------------------------------------------------

type scenarioRow struct {
	name     string
	feature  string
	tags     []string
	status   reports.Status
	duration time.Duration
	err      error
	steps    []stepRow
	running  bool
}

type stepRow struct {
	keyword string
	text    string
	status  reports.Status
	err     error
}

// --- Bubbletea model -------------------------------------------------------

// RunModel is the Bubbletea model for a live run TUI.
type RunModel struct {
	runID     string
	startedAt time.Time
	scenarios []scenarioRow
	current   int // index of the currently executing scenario
	done      bool
	runErr    error
	spinner   spinner.Model
	summary   *RunFinishedMsg

	// Terminal width for layout.
	width int
}

// NewRunModel creates an initial RunModel.
func NewRunModel() RunModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorPrimary)
	return RunModel{
		spinner: sp,
		current: -1,
		width:   80,
	}
}

func (m RunModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m RunModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case RunStartedMsg:
		m.runID = msg.RunID
		m.startedAt = msg.StartedAt

	case ScenarioStartedMsg:
		m.scenarios = append(m.scenarios, scenarioRow{
			name:    msg.Name,
			feature: msg.Feature,
			tags:    msg.Tags,
			status:  reports.StatusUnknown,
			running: true,
		})
		m.current = len(m.scenarios) - 1

	case StepFinishedMsg:
		if m.current >= 0 && m.current < len(m.scenarios) {
			m.scenarios[m.current].steps = append(m.scenarios[m.current].steps, stepRow{
				keyword: msg.Keyword,
				text:    msg.Text,
				status:  msg.Status,
				err:     msg.Err,
			})
		}

	case ScenarioFinishedMsg:
		if m.current >= 0 && m.current < len(m.scenarios) {
			sc := &m.scenarios[m.current]
			sc.status = msg.Status
			sc.duration = msg.Duration
			sc.err = msg.Err
			sc.running = false
		}

	case RunFinishedMsg:
		m.done = true
		m.summary = &msg

	case RunnerErrMsg:
		m.done = true
		m.runErr = msg.Err
		return m, tea.Quit
	}

	if m.done {
		return m, tea.Quit
	}

	return m, nil
}

func (m RunModel) View() string {
	var b strings.Builder

	// Header
	header := StyleHeading.Render(IconRocket + " lobster run")
	if m.runID != "" {
		header += "  " + StyleMuted.Render("run/"+m.runID[:8])
	}
	b.WriteString(header + "\n\n")

	// Scenario list
	for i, sc := range m.scenarios {
		icon, iconStyle := scenarioStatusIcon(sc.status, sc.running)
		if sc.running {
			icon = m.spinner.View()
		}

		name := StyleBold.Render(sc.feature + " › " + sc.name)
		tags := ""
		if len(sc.tags) > 0 {
			tags = " " + StyleMuted.Render(strings.Join(sc.tags, " "))
		}
		dur := ""
		if !sc.running && sc.duration > 0 {
			dur = "  " + StyleMuted.Render(formatRunDuration(sc.duration))
		}
		b.WriteString(fmt.Sprintf("  %s  %s%s%s\n", iconStyle.Render(icon), name, tags, dur))

		// Show steps for current running scenario or any failed one
		if sc.running || (sc.status == reports.StatusFailed && i == m.current) {
			for _, step := range sc.steps {
				sIcon, sStyle := stepStatusIcon(step.status)
				stepText := StyleMuted.Render(step.keyword) + " " + step.text
				b.WriteString(fmt.Sprintf("       %s  %s\n", sStyle.Render(sIcon), stepText))
				if step.err != nil {
					b.WriteString("          " + StyleError.Render(step.err.Error()) + "\n")
				}
			}
		}
	}

	// Summary
	if m.done && m.summary != nil {
		b.WriteString("\n")
		b.WriteString(m.renderSummary())
	}

	return b.String()
}

// Summary returns the run summary once the run is finished, or nil.
func (m RunModel) Summary() *RunFinishedMsg {
	return m.summary
}

func (m RunModel) renderSummary() string {
	s := m.summary
	dur := formatRunDuration(s.Duration)

	if s.Failed == 0 {
		line := fmt.Sprintf("%s  %d/%d scenarios passed  %s",
			StyleSuccess.Render(IconCheck+" Passed"),
			s.Passed, s.Total,
			StyleMuted.Render(dur),
		)
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSuccess).
			Padding(0, 2).
			Render(line) + "\n"
	}

	line := fmt.Sprintf("%s  %d passed  %s  %d failed  %s",
		StyleError.Render(IconCross+" Failed"),
		s.Passed,
		StyleError.Render(fmt.Sprintf("%d failed", s.Failed)),
		s.Failed,
		StyleMuted.Render(dur),
	)
	// Simpler failure summary
	line = fmt.Sprintf("%s  %d/%d passed · %d failed · %s",
		StyleError.Render(IconCross+" Failed"),
		s.Passed, s.Total, s.Failed,
		StyleMuted.Render(dur),
	)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorError).
		Padding(0, 2).
		Render(line) + "\n"
}

// scenarioStatusIcon returns icon text and style based on status.
func scenarioStatusIcon(status reports.Status, running bool) (string, lipgloss.Style) {
	if running {
		return IconDot, lipgloss.NewStyle().Foreground(colorPrimary)
	}
	switch status {
	case reports.StatusPassed:
		return IconCheck, StyleSuccess
	case reports.StatusFailed:
		return IconCross, StyleError
	case reports.StatusSkipped:
		return "-", StyleMuted
	case reports.StatusUndefined:
		return "?", StyleWarning
	default:
		return IconDot, StyleMuted
	}
}

// stepStatusIcon returns icon and style for a step result.
func stepStatusIcon(status reports.Status) (string, lipgloss.Style) {
	switch status {
	case reports.StatusPassed:
		return IconCheck, StyleSuccess
	case reports.StatusFailed:
		return IconCross, StyleError
	case reports.StatusUndefined:
		return "?", StyleWarning
	case reports.StatusSkipped:
		return "-", StyleMuted
	default:
		return IconDot, StyleMuted
	}
}

func formatRunDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

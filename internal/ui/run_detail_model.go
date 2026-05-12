package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	commonv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/common"
	runv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/run"
	"github.com/bcp-technology/lobster/internal/reports"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// RunDetailModel renders a scrollable detail view for a single Run.
type RunDetailModel struct {
	run      *runv1.Run
	viewport viewport.Model
	width    int
	height   int
	ready    bool
}

// NewRunDetailModel creates a RunDetailModel for the given run.
func NewRunDetailModel(run *runv1.Run) RunDetailModel {
	return RunDetailModel{run: run, width: 80, height: 24}
}

// SetSize updates the terminal dimensions.
func (m *RunDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	if m.ready {
		m.viewport.Width = w
		m.viewport.Height = h - 4
	}
}

func (m RunDetailModel) Init() tea.Cmd {
	return func() tea.Msg { return runDetailReadyMsg{} }
}

type runDetailReadyMsg struct{}

func (m RunDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case runDetailReadyMsg:
		m.viewport = viewport.New(m.width, m.height-4)
		m.viewport.SetContent(m.buildContent())
		m.ready = true

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.ready {
			m.viewport.Width = m.width
			m.viewport.Height = m.height - 4
			m.viewport.SetContent(m.buildContent())
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "b":
			// Caller handles 'b' to go back.
			return m, nil
		}
	}

	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m RunDetailModel) View() string {
	if !m.ready {
		return StyleMuted.Render("  loading…")
	}

	header := StyleHeading.Render("Run Detail  "+shortID(m.run.GetRunId())) +
		"  " + StyleMuted.Render(m.run.GetRunId())
	footer := StyleMuted.Render("[↑/↓] scroll  [b] back  [q] quit")
	return header + "\n" + m.viewport.View() + "\n" + footer
}

func (m RunDetailModel) buildContent() string {
	var b strings.Builder

	// Metadata section.
	b.WriteString(StyleSubheading.Render("Run") + "\n")
	kvRows := [][2]string{
		{"ID", m.run.GetRunId()},
		{"Workspace", m.run.GetWorkspaceId()},
		{"Profile", m.run.GetProfileName()},
		{"Status", runStatusLabel(m.run.GetStatus().String())},
	}
	if ts := m.run.GetCreatedAt(); ts != nil {
		kvRows = append(kvRows, [2]string{"Created", ts.AsTime().Format("2006-01-02 15:04:05 UTC")})
	}
	if ts := m.run.GetStartedAt(); ts != nil {
		kvRows = append(kvRows, [2]string{"Started", ts.AsTime().Format("2006-01-02 15:04:05 UTC")})
	}
	if ts := m.run.GetEndedAt(); ts != nil {
		kvRows = append(kvRows, [2]string{"Ended", ts.AsTime().Format("2006-01-02 15:04:05 UTC")})
	}
	if s := m.run.GetSummary(); s != nil {
		if d := s.GetDuration(); d != nil {
			kvRows = append(kvRows, [2]string{"Duration", formatRunDuration(d.AsDuration())})
		}
		kvRows = append(kvRows, [2]string{"Scenarios",
			fmt.Sprintf("%d total  %d passed  %d failed  %d skipped",
				s.GetTotalScenarios(), s.GetPassedScenarios(),
				s.GetFailedScenarios(), s.GetSkippedScenarios())})
	}
	b.WriteString(RenderKeyValueTable("", kvRows))
	b.WriteString("\n")

	// Scenario results.
	results := m.run.GetScenarioResults()
	if len(results) == 0 {
		b.WriteString(StyleMuted.Render("  No scenario results.") + "\n")
		return b.String()
	}

	b.WriteString(StyleSubheading.Render("Scenarios") + "\n\n")
	for _, sc := range results {
		icon, iconStyle := scenarioStatusIcon(protoScenarioStatusToReports(sc.GetStatus()), false)
		dur := ""
		if d := sc.GetDuration(); d != nil {
			dur = "  " + StyleMuted.Render(formatRunDuration(d.AsDuration()))
		}
		b.WriteString(fmt.Sprintf("  %s  %s%s\n",
			iconStyle.Render(icon),
			StyleBold.Render(sc.GetScenarioId()),
			dur))

		for _, step := range sc.GetStepResults() {
			sIcon, sStyle := stepStatusIcon(stepStatusFromProto(step.GetStatus()))
			stepDur := ""
			if d := step.GetDuration(); d != nil {
				stepDur = "  " + StyleMuted.Render(formatRunDuration(d.AsDuration()))
			}
			b.WriteString(fmt.Sprintf("       %s  %s%s\n",
				sStyle.Render(sIcon),
				StyleMuted.Render("step/"+shortID(step.GetStepId())),
				stepDur))
			for _, af := range step.GetAssertionFailures() {
				b.WriteString("          " + StyleError.Render(af.GetMessage()) + "\n")
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// protoScenarioStatusToReports converts a commonv1.ScenarioStatus to reports.Status.
func protoScenarioStatusToReports(s commonv1.ScenarioStatus) reports.Status {
	switch s {
	case commonv1.ScenarioStatus_SCENARIO_STATUS_PASSED:
		return reports.StatusPassed
	case commonv1.ScenarioStatus_SCENARIO_STATUS_FAILED:
		return reports.StatusFailed
	case commonv1.ScenarioStatus_SCENARIO_STATUS_SKIPPED:
		return reports.StatusSkipped
	default:
		return reports.StatusUnknown
	}
}

// contextWithTimeout creates a context with the given timeout duration.
// Shared by TUI models that need short-lived request contexts.
func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

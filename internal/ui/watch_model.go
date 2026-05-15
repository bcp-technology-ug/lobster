package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	commonv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/common"
	runv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/run"
	"github.com/bcp-technology-ug/lobster/internal/reports"
)

// WatchModel is a Bubbletea model that streams live run events from a
// gRPC StreamRunEvents call and renders a live TUI.
// It is used by `lobster run watch` when attached to a TTY.
type WatchModel struct {
	runID    string
	rows     []watchRow
	done     bool
	exitCode int
	spinner  spinner.Model
	summary  *watchSummary
	width    int
	stream   runv1.RunService_StreamRunEventsClient
}

type watchRow struct {
	scenarioID string
	status     reports.Status
	duration   string
	errText    string
}

type watchSummary struct {
	total   uint32
	passed  uint32
	failed  uint32
	skipped uint32
}

// NewWatchModel creates a WatchModel bound to an already-open gRPC stream.
func NewWatchModel(runID string, stream runv1.RunService_StreamRunEventsClient) WatchModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorPrimary)
	return WatchModel{
		runID:   runID,
		stream:  stream,
		spinner: sp,
		width:   80,
	}
}

// watchEventMsg carries a received RunEvent from the stream goroutine.
type watchEventMsg struct{ event *runv1.RunEvent }

// watchDoneMsg signals the stream has ended.
type watchDoneMsg struct{}

// listenCmd reads the next event from the stream and sends it as a tea.Msg.
func (m WatchModel) listenCmd() tea.Cmd {
	return func() tea.Msg {
		ev, err := m.stream.Recv()
		if err != nil {
			return watchDoneMsg{}
		}
		return watchEventMsg{event: ev}
	}
}

func (m WatchModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.listenCmd())
}

func (m WatchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case watchEventMsg:
		m.applyEvent(msg.event)
		if m.done {
			return m, tea.Quit
		}
		return m, m.listenCmd()

	case watchDoneMsg:
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

// ExitCode returns 1 if any scenarios failed, 0 otherwise.
func (m WatchModel) ExitCode() int { return m.exitCode }

func (m *WatchModel) applyEvent(ev *runv1.RunEvent) {
	switch p := ev.GetPayload().(type) {
	case *runv1.RunEvent_ScenarioResult:
		sr := p.ScenarioResult
		row := watchRow{scenarioID: sr.GetScenarioId()}
		if d := sr.GetDuration(); d != nil {
			row.duration = formatRunDuration(d.AsDuration())
		}
		switch sr.GetStatus() {
		case commonv1.ScenarioStatus_SCENARIO_STATUS_PASSED:
			row.status = reports.StatusPassed
		case commonv1.ScenarioStatus_SCENARIO_STATUS_FAILED:
			row.status = reports.StatusFailed
			m.exitCode = 1
			// Collect first assertion failure as error text.
			for _, st := range sr.GetStepResults() {
				for _, af := range st.GetAssertionFailures() {
					if row.errText == "" {
						row.errText = af.GetMessage()
					}
				}
			}
		case commonv1.ScenarioStatus_SCENARIO_STATUS_SKIPPED:
			row.status = reports.StatusSkipped
		default:
			row.status = reports.StatusUnknown
		}
		m.rows = append(m.rows, row)

	case *runv1.RunEvent_Summary:
		s := p.Summary
		m.summary = &watchSummary{
			total:   s.GetTotalScenarios(),
			passed:  s.GetPassedScenarios(),
			failed:  s.GetFailedScenarios(),
			skipped: s.GetSkippedScenarios(),
		}
		if s.GetFailedScenarios() > 0 {
			m.exitCode = 1
		}
	}

	if ev.GetTerminal() {
		m.done = true
	}
}

func (m WatchModel) View() string {
	var b strings.Builder

	header := StyleHeading.Render(IconRocket + " lobster run watch")
	if m.runID != "" {
		header += "  " + StyleMuted.Render("run/"+shortID(m.runID))
	}
	b.WriteString(header + "\n\n")

	for _, row := range m.rows {
		icon, iconStyle := scenarioStatusIcon(row.status, false)
		dur := ""
		if row.duration != "" {
			dur = "  " + StyleMuted.Render(row.duration)
		}
		fmt.Fprintf(&b, "  %s  %s%s\n",
			iconStyle.Render(icon),
			StyleBold.Render(row.scenarioID),
			dur)
		if row.errText != "" {
			b.WriteString("       " + StyleError.Render(row.errText) + "\n")
		}
	}

	if !m.done {
		fmt.Fprintf(&b, "\n  %s streaming…  press q to detach\n",
			m.spinner.View())
	}

	if m.done && m.summary != nil {
		b.WriteString("\n")
		b.WriteString(m.renderWatchSummary())
	}

	return b.String()
}

func (m WatchModel) renderWatchSummary() string {
	s := m.summary
	if s.failed == 0 {
		return StyleSuccess.Render(fmt.Sprintf("  %s  %d/%d scenarios passed",
			IconCheck+" Passed", s.passed, s.total)) + "\n"
	}
	return StyleError.Render(fmt.Sprintf("  %s  %d failed  %d/%d passed",
		IconCross+" Failed", s.failed, s.passed, s.total)) + "\n"
}

// shortID returns the first 8 characters of an ID string.
func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

// stepStatusFromProto maps proto StepStatus to reports.Status.
func stepStatusFromProto(s commonv1.StepStatus) reports.Status {
	switch s {
	case commonv1.StepStatus_STEP_STATUS_PASSED:
		return reports.StatusPassed
	case commonv1.StepStatus_STEP_STATUS_FAILED:
		return reports.StatusFailed
	case commonv1.StepStatus_STEP_STATUS_SKIPPED:
		return reports.StatusSkipped
	default:
		return reports.StatusUnknown
	}
}

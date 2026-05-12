package ui

import (
	"fmt"
	"strings"
	"time"

	runv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/run"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RunsListModel is a Bubbletea model for browsing the run history.
// It pages through runs from the daemon and optionally opens a detail view.
type RunsListModel struct {
	client    runv1.RunServiceClient
	workspace string
	rows      []*runv1.Run
	table     table.Model
	filter    textinput.Model
	filtering bool
	nextToken string
	prevToken string
	loading   bool
	err       error
	detail    *RunDetailModel
	width     int
	height    int
}

// runsLoadMsg is sent when a page of runs arrives.
type runsLoadMsg struct {
	runs      []*runv1.Run
	nextToken string
	err       error
}

// NewRunsListModel creates a RunsListModel with the given gRPC client.
func NewRunsListModel(client runv1.RunServiceClient, workspace string) RunsListModel {
	cols := []table.Column{
		{Title: "ID", Width: 8},
		{Title: "Status", Width: 12},
		{Title: "Workspace", Width: 16},
		{Title: "Scenarios", Width: 10},
		{Title: "Duration", Width: 10},
		{Title: "Created", Width: 20},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(tableStyles())

	ti := textinput.New()
	ti.Placeholder = "filter…"
	ti.Width = 30

	return RunsListModel{
		client:    client,
		workspace: workspace,
		table:     t,
		filter:    ti,
		width:     80,
		height:    24,
	}
}

func (m RunsListModel) Init() tea.Cmd {
	return m.loadPage("")
}

func (m RunsListModel) loadPage(token string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout(5 * time.Second)
		defer cancel()
		resp, err := m.client.ListRuns(ctx, &runv1.ListRunsRequest{
			WorkspaceId: m.workspace,
			PageSize:    25,
			PageToken:   token,
		})
		if err != nil {
			return runsLoadMsg{err: err}
		}
		runs := resp.GetRuns()
		out := make([]*runv1.Run, len(runs))
		copy(out, runs)
		return runsLoadMsg{runs: out, nextToken: resp.GetNextPageToken()}
	}
}

func (m RunsListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If we have a detail view open, delegate to it.
	if m.detail != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "b" || msg.String() == "backspace" {
				m.detail = nil
				return m, nil
			}
		}
		newDetail, cmd := m.detail.Update(msg)
		if nd, ok := newDetail.(RunDetailModel); ok {
			m.detail = &nd
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetHeight(m.height - 6)

	case tea.KeyMsg:
		if m.filtering {
			switch msg.String() {
			case "esc", "enter":
				m.filtering = false
				m.filter.Blur()
				m.applyRebuild()
				return m, nil
			default:
				var cmd tea.Cmd
				m.filter, cmd = m.filter.Update(msg)
				m.applyRebuild()
				return m, cmd
			}
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "/":
			m.filtering = true
			m.filter.Focus()
			return m, textinput.Blink
		case "enter":
			if row := m.table.SelectedRow(); len(row) > 0 {
				return m, m.openDetail(row[0])
			}
		case "r":
			m.loading = true
			return m, m.loadPage("")
		case "right", "n":
			if m.nextToken != "" {
				m.loading = true
				return m, m.loadPage(m.nextToken)
			}
		}

	case runsLoadMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.rows = make([]*runv1.Run, len(msg.runs))
		for i := range msg.runs {
			m.rows[i] = msg.runs[i]
		}
		m.nextToken = msg.nextToken
		m.applyRebuild()
		return m, nil

	case runsDetailLoadMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		d := NewRunDetailModel(msg.run)
		d.SetSize(m.width, m.height)
		m.detail = &d
		return m, d.Init()
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// runsDetailLoadMsg is sent when a detail run is loaded.
type runsDetailLoadMsg struct {
	run *runv1.Run
	err error
}

func (m RunsListModel) openDetail(runID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout(5 * time.Second)
		defer cancel()
		resp, err := m.client.GetRun(ctx, &runv1.GetRunRequest{RunId: runID})
		if err != nil {
			return runsDetailLoadMsg{err: err}
		}
		return runsDetailLoadMsg{run: resp.GetRun()}
	}
}

func (m *RunsListModel) applyRebuild() {
	filterText := strings.ToLower(m.filter.Value())
	var rows []table.Row
	for _, r := range m.rows {
		idShort := shortID(r.GetRunId())
		status := runStatusLabel(r.GetStatus().String())
		workspace := r.GetWorkspaceId()
		dur := ""
		scenarios := ""
		if s := r.GetSummary(); s != nil {
			if d := s.GetDuration(); d != nil {
				dur = formatRunDuration(d.AsDuration())
			}
			scenarios = fmt.Sprintf("%d/%d", s.GetPassedScenarios(), s.GetTotalScenarios())
		}
		created := ""
		if ts := r.GetCreatedAt(); ts != nil {
			created = ts.AsTime().Format("2006-01-02 15:04:05")
		}
		row := table.Row{idShort, status, workspace, scenarios, dur, created}
		if filterText == "" || strings.Contains(strings.ToLower(workspace), filterText) ||
			strings.Contains(strings.ToLower(idShort), filterText) {
			rows = append(rows, row)
		}
	}
	m.table.SetRows(rows)
}

func (m RunsListModel) View() string {
	if m.detail != nil {
		return m.detail.View()
	}

	var b strings.Builder
	b.WriteString(StyleHeading.Render("Run History") + "\n")

	if m.err != nil {
		b.WriteString(StyleError.Render("  "+m.err.Error()) + "\n")
	}

	if m.filtering {
		b.WriteString("  Filter: " + m.filter.View() + "\n")
	}

	b.WriteString(m.table.View() + "\n\n")

	// Footer keys
	help := []string{"[enter] detail", "[/] filter", "[r] refresh", "[→] next page", "[q] quit"}
	if m.nextToken != "" {
		help = append(help, "[→] next")
	}
	b.WriteString(StyleMuted.Render(strings.Join(help, "  ")) + "\n")

	return b.String()
}

// runStatusLabel returns a short display string for a run status.
func runStatusLabel(s string) string {
	switch s {
	case "RUN_STATUS_RUNNING":
		return "running"
	case "RUN_STATUS_PASSED":
		return "passed"
	case "RUN_STATUS_FAILED":
		return "failed"
	case "RUN_STATUS_CANCELLED":
		return "cancelled"
	case "RUN_STATUS_PENDING":
		return "pending"
	default:
		return strings.ToLower(strings.TrimPrefix(s, "RUN_STATUS_"))
	}
}

// tableStyles returns a styled table.Styles for the TUI.
func tableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorMuted).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(colorPrimary).
		Bold(false)
	return s
}

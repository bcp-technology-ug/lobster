package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	runv1 "github.com/bcp-technology-ug/lobster/gen/go/lobster/v1/run"
)

// RunsListModel is a Bubbletea model for browsing the run history.
// It pages through runs from the daemon and optionally opens a detail view.
type RunsListModel struct {
	client    runv1.RunServiceClient
	workspace string
	rows      []*runv1.Run
	filtered  []*runv1.Run // mirrors table rows after filtering
	table     table.Model
	filter    textinput.Model
	filtering bool
	nextToken string
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
		{Title: "ID", Width: 10},
		{Title: "Status", Width: 12},
		{Title: "Workspace", Width: 18},
		{Title: "Scenarios", Width: 12},
		{Title: "Duration", Width: 11},
		{Title: "Created", Width: 14},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(3),
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
		cw := m.cardWidth()
		tableInner := cw - 6 // subtract border(2) + padding(4)
		// Workspace is capped at 24; any leftover space is distributed evenly
		// across the other 5 columns so the table always fills the card.
		const idW, stW, scW, durW, crtW = 10, 12, 12, 11, 14
		wsWidth := tableInner - (idW + stW + scW + durW + crtW)
		if wsWidth < 12 {
			wsWidth = 12
		}
		if wsWidth > 24 {
			wsWidth = 24
		}
		perCol := (tableInner - (idW + stW + wsWidth + scW + durW + crtW)) / 5
		if perCol < 0 {
			perCol = 0
		}
		m.table.SetColumns([]table.Column{
			{Title: "ID", Width: idW + perCol},
			{Title: "Status", Width: stW + perCol},
			{Title: "Workspace", Width: wsWidth},
			{Title: "Scenarios", Width: scW + perCol},
			{Title: "Duration", Width: durW + perCol},
			{Title: "Created", Width: crtW + perCol},
		})
		m.table.SetHeight(max(3, m.height-10))
		return m, nil

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
			if idx := m.table.Cursor(); idx >= 0 && idx < len(m.filtered) {
				return m, m.openDetail(m.filtered[idx].GetRunId())
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
		copy(m.rows, msg.runs)
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
	var filtered []*runv1.Run
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
			created = ts.AsTime().Format("01/02 15:04")
		}
		row := table.Row{idShort, status, workspace, scenarios, dur, created}
		if filterText == "" || strings.Contains(strings.ToLower(workspace), filterText) ||
			strings.Contains(strings.ToLower(idShort), filterText) {
			rows = append(rows, row)
			filtered = append(filtered, r)
		}
	}
	m.filtered = filtered
	m.table.SetRows(rows)
	// Cap the table height to actual row count so empty/sparse lists don't
	// expand to fill all available space with blank rows.
	maxH := max(3, m.height-10)
	wantH := max(3, len(rows)+2)
	m.table.SetHeight(min(wantH, maxH))
}

func (m RunsListModel) View() string {
	if m.detail != nil {
		return m.detail.View()
	}

	// ── Card header ──────────────────────────────────────────────────────────
	title := TUICardHeaderStyle.Render("Run History")

	var infoLine string
	switch {
	case m.loading:
		infoLine = "\n" + StyleMuted.Render("  loading…")
	case m.err != nil:
		infoLine = "\n" + StyleError.Render("  "+m.err.Error())
	case m.filtering:
		infoLine = "\n" + StyleMuted.Render("  filter: ") + m.filter.View()
	}

	// ── Footer hints ─────────────────────────────────────────────────────────
	hints := []string{
		renderKeyHint("↵", "detail"),
		renderKeyHint("/", "filter"),
		renderKeyHint("r", "refresh"),
	}
	if m.nextToken != "" {
		hints = append(hints, renderKeyHint("→", "next page"))
	}
	footerHints := strings.Join(hints, "   ")

	content := title + infoLine + "\n\n" + m.table.View() + "\n\n" + StyleMuted.Render(footerHints)

	cw := m.cardWidth()
	card := TUICardStyle.Width(cw - 6).Render(content)
	return TUICenter(m.width, card)
}

// cardWidth returns the outer width of the centred content card.
func (m RunsListModel) cardWidth() int {
	w := m.width - 2
	if w > 200 {
		w = 200
	}
	if w < 60 {
		w = 60
	}
	return w
}

// runStatusLabel returns a plain-text status label safe for bubbles table cells.
// Bubbles uses runewidth.Truncate internally, which is not ANSI-aware; any
// escape code in a cell value can be mis-measured and truncated mid-sequence,
// corrupting every column to the right. Plain text avoids this entirely.
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
	s.Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorPrimary).
		BorderBottom(true)
	s.Selected = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(colorPrimary).
		Bold(true)
	s.Cell = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#D1D5DB"})
	return s
}

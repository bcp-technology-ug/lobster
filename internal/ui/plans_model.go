package ui

import (
	"fmt"
	"strings"
	"time"

	planv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/plan"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// PlansListModel is a Bubbletea model for browsing execution plans.
type PlansListModel struct {
	client    planv1.PlanServiceClient
	workspace string
	plans     []*planv1.ExecutionPlan
	table     table.Model
	filter    textinput.Model
	filtering bool
	nextToken string
	loading   bool
	err       error
	detail    *planDetailModel
	width     int
	height    int
}

type planDetailModel struct {
	plan     *planv1.ExecutionPlan
	viewport viewport.Model
	ready    bool
}

// plansLoadMsg is sent when a page of plans arrives.
type plansLoadMsg struct {
	plans     []*planv1.ExecutionPlan
	nextToken string
	err       error
}

type plansDetailLoadMsg struct {
	plan *planv1.ExecutionPlan
	err  error
}

// NewPlansListModel creates a PlansListModel backed by the given gRPC client.
func NewPlansListModel(client planv1.PlanServiceClient, workspace string) PlansListModel {
	cols := []table.Column{
		{Title: "ID", Width: 8},
		{Title: "Workspace", Width: 16},
		{Title: "Profile", Width: 12},
		{Title: "Scenarios", Width: 10},
		{Title: "Est. Duration", Width: 14},
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

	return PlansListModel{
		client:    client,
		workspace: workspace,
		table:     t,
		filter:    ti,
		width:     80,
		height:    24,
	}
}

func (m PlansListModel) Init() tea.Cmd {
	return m.loadPage("")
}

func (m PlansListModel) loadPage(token string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout(5 * time.Second)
		defer cancel()
		resp, err := m.client.ListPlans(ctx, &planv1.ListPlansRequest{
			WorkspaceId: m.workspace,
			PageSize:    25,
			PageToken:   token,
		})
		if err != nil {
			return plansLoadMsg{err: err}
		}
		return plansLoadMsg{plans: resp.GetPlans(), nextToken: resp.GetNextPageToken()}
	}
}

func (m PlansListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.detail != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "b" || msg.String() == "backspace" {
				m.detail = nil
				return m, nil
			}
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			if m.detail.ready {
				m.detail.viewport.Width = msg.Width
				m.detail.viewport.Height = msg.Height - 4
			}
		}
		if m.detail.ready {
			var cmd tea.Cmd
			m.detail.viewport, cmd = m.detail.viewport.Update(msg)
			return m, cmd
		}
		return m, nil
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
				m.applyFilter()
				return m, nil
			default:
				var cmd tea.Cmd
				m.filter, cmd = m.filter.Update(msg)
				m.applyFilter()
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

	case plansLoadMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.plans = msg.plans
		m.nextToken = msg.nextToken
		m.applyFilter()
		return m, nil

	case plansDetailLoadMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		vp := viewport.New(m.width, m.height-4)
		vp.SetContent(buildPlanContent(msg.plan))
		m.detail = &planDetailModel{plan: msg.plan, viewport: vp, ready: true}
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *PlansListModel) openDetail(planID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout(5 * time.Second)
		defer cancel()
		resp, err := m.client.GetPlan(ctx, &planv1.GetPlanRequest{PlanId: planID})
		if err != nil {
			return plansDetailLoadMsg{err: err}
		}
		return plansDetailLoadMsg{plan: resp.GetPlan()}
	}
}

func (m *PlansListModel) applyFilter() {
	filterText := strings.ToLower(m.filter.Value())
	var rows []table.Row
	for _, p := range m.plans {
		idShort := shortID(p.GetPlanId())
		workspace := p.GetWorkspaceId()
		profile := p.GetProfileName()
		scenarios := ""
		if sc := p.GetScenarios(); len(sc) > 0 {
			scenarios = fmt.Sprintf("%d", len(sc))
		}
		dur := ""
		if d := p.GetEstimatedDuration(); d != nil {
			dur = formatRunDuration(d.AsDuration())
		}
		created := ""
		if ts := p.GetCreatedAt(); ts != nil {
			created = ts.AsTime().Format("2006-01-02 15:04:05")
		}
		row := table.Row{idShort, workspace, profile, scenarios, dur, created}
		if filterText == "" ||
			strings.Contains(strings.ToLower(workspace), filterText) ||
			strings.Contains(strings.ToLower(profile), filterText) {
			rows = append(rows, row)
		}
	}
	m.table.SetRows(rows)
}

func (m PlansListModel) View() string {
	if m.detail != nil {
		header := StyleHeading.Render("Plan Detail  "+shortID(m.detail.plan.GetPlanId())) +
			"  " + StyleMuted.Render(m.detail.plan.GetPlanId())
		footer := StyleMuted.Render("[↑/↓] scroll  [b] back  [q] quit")
		return header + "\n" + m.detail.viewport.View() + "\n" + footer
	}

	var b strings.Builder
	b.WriteString(StyleHeading.Render("Execution Plans") + "\n")
	if m.err != nil {
		b.WriteString(StyleError.Render("  "+m.err.Error()) + "\n")
	}
	if m.filtering {
		b.WriteString("  Filter: " + m.filter.View() + "\n")
	}
	b.WriteString(m.table.View() + "\n\n")
	b.WriteString(StyleMuted.Render("[enter] detail  [/] filter  [r] refresh  [→] next page  [q] quit") + "\n")
	return b.String()
}

func buildPlanContent(p *planv1.ExecutionPlan) string {
	var b strings.Builder
	kvRows := [][2]string{
		{"Plan ID", p.GetPlanId()},
		{"Workspace", p.GetWorkspaceId()},
		{"Profile", p.GetProfileName()},
	}
	if d := p.GetEstimatedDuration(); d != nil {
		kvRows = append(kvRows, [2]string{"Est. Duration", formatRunDuration(d.AsDuration())})
	}
	if ts := p.GetCreatedAt(); ts != nil {
		kvRows = append(kvRows, [2]string{"Created", ts.AsTime().Format("2006-01-02 15:04:05 UTC")})
	}
	b.WriteString(RenderKeyValueTable("Plan", kvRows))
	b.WriteString("\n")

	scenarios := p.GetScenarios()
	if len(scenarios) == 0 {
		b.WriteString(StyleMuted.Render("  No scenarios.") + "\n")
		return b.String()
	}

	b.WriteString(StyleSubheading.Render("Scenarios") + "\n\n")
	for _, sc := range scenarios {
		tags := ""
		if len(sc.GetTags()) > 0 {
			tags = "  " + StyleMuted.Render(strings.Join(sc.GetTags(), " "))
		}
		b.WriteString("  " + StyleBold.Render(sc.GetFeatureName()+" › "+sc.GetScenarioName()) + tags + "\n")
	}
	return b.String()
}

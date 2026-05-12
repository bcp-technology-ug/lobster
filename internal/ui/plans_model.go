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
	filtered  []*planv1.ExecutionPlan // mirrors table rows after filtering
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
		{Title: "Workspace", Width: 8},
		{Title: "Profile", Width: 10},
		{Title: "Scenarios", Width: 8},
		{Title: "Est. Duration", Width: 10},
		{Title: "Created", Width: 10},
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
				m.detail.viewport.Width = m.cardWidth() - 10
				m.detail.viewport.Height = m.height - 12
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
		cw := m.cardWidth()
		tableInner := cw - 6
		// fixed cols: ID(8)+Profile(10)+Scenarios(8)+EstDur(10)+Created(10) = 46
		// workspace absorbs the slack; cardWidth() floor ensures wsWidth >= 8.
		wsWidth := tableInner - 46
		if wsWidth < 8 {
			wsWidth = 8
		}
		m.table.SetColumns([]table.Column{
			{Title: "ID", Width: 8},
			{Title: "Workspace", Width: wsWidth},
			{Title: "Profile", Width: 10},
			{Title: "Scenarios", Width: 8},
			{Title: "Est. Duration", Width: 10},
			{Title: "Created", Width: 10},
		})
		m.table.SetHeight(max(3, m.height-10))

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
			if idx := m.table.Cursor(); idx >= 0 && idx < len(m.filtered) {
				return m, m.openDetail(m.filtered[idx].GetPlanId())
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
		vp := viewport.New(m.cardWidth()-10, max(3, m.height-12))
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
	var filtered []*planv1.ExecutionPlan
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
			created = ts.AsTime().Format("01/02 15:04")
		}
		row := table.Row{idShort, workspace, profile, scenarios, dur, created}
		if filterText == "" ||
			strings.Contains(strings.ToLower(workspace), filterText) ||
			strings.Contains(strings.ToLower(profile), filterText) {
			rows = append(rows, row)
			filtered = append(filtered, p)
		}
	}
	m.filtered = filtered
	m.table.SetRows(rows)
}

func (m PlansListModel) View() string {
	if m.detail != nil {
		header := TUICardHeaderStyle.Render("Plan  "+shortID(m.detail.plan.GetPlanId())) +
			"  " + StyleMuted.Render(m.detail.plan.GetPlanId())
		hints := renderKeyHint("↑/↓", "scroll") + "   " + renderKeyHint("b", "back")
		content := header + "\n\n" + m.detail.viewport.View() + "\n\n" + StyleMuted.Render(hints)
		cw := m.cardWidth()
		card := TUICardStyle.Width(cw - 6).Render(content)
		return TUICenter(m.width, card)
	}

	title := TUICardHeaderStyle.Render("Execution Plans")
	var infoLine string
	switch {
	case m.loading:
		infoLine = "\n" + StyleMuted.Render("  loading…")
	case m.err != nil:
		infoLine = "\n" + StyleError.Render("  "+m.err.Error())
	case m.filtering:
		infoLine = "\n" + StyleMuted.Render("  filter: ") + m.filter.View()
	}

	hints := []string{
		renderKeyHint("↵", "detail"),
		renderKeyHint("/", "filter"),
		renderKeyHint("r", "refresh"),
	}
	if m.nextToken != "" {
		hints = append(hints, renderKeyHint("→", "next page"))
	}

	content := title + infoLine + "\n\n" + m.table.View() + "\n\n" + StyleMuted.Render(strings.Join(hints, "   "))
	cw := m.cardWidth()
	card := TUICardStyle.Width(cw - 6).Render(content)
	return TUICenter(m.width, card)
}

// cardWidth returns the outer width of the centered content card.
func (m PlansListModel) cardWidth() int {
	w := m.width - 6
	if w > 128 {
		w = 128
	}
	if w < 60 {
		w = 60
	}
	return w
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

package ui

import (
	"fmt"
	"strings"
	"time"

	stackv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/stack"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// StackStatusModel is a Bubbletea model that displays the current stack status.
// When watch=true it auto-refreshes every 2 seconds.
type StackStatusModel struct {
	client    stackv1.StackServiceClient
	workspace string
	stack     *stackv1.Stack
	table     table.Model
	watch     bool
	err       error
	width     int
	height    int
}

// stackRefreshMsg is sent periodically in watch mode.
type stackRefreshMsg struct{}

// stackLoadMsg is sent when a stack status response arrives.
type stackLoadMsg struct {
	stack *stackv1.Stack
	err   error
}

// NewStackStatusModel creates a StackStatusModel.
// Set watch=true to enable auto-refresh.
func NewStackStatusModel(client stackv1.StackServiceClient, workspace string, watch bool) StackStatusModel {
	cols := []table.Column{
		{Title: "Service", Width: 20},
		{Title: "Container ID", Width: 14},
		{Title: "Status", Width: 10},
		{Title: "Health", Width: 12},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(tableStyles())
	return StackStatusModel{
		client:    client,
		workspace: workspace,
		watch:     watch,
		table:     t,
		width:     80,
		height:    24,
	}
}

func (m StackStatusModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.fetchCmd()}
	if m.watch {
		cmds = append(cmds, tickAfter(2*time.Second))
	}
	return tea.Batch(cmds...)
}

func tickAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return stackRefreshMsg{} })
}

func (m StackStatusModel) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout(5 * time.Second)
		defer cancel()
		resp, err := m.client.GetStackStatus(ctx, &stackv1.GetStackStatusRequest{
			WorkspaceId: m.workspace,
		})
		if err != nil {
			return stackLoadMsg{err: err}
		}
		return stackLoadMsg{stack: resp.GetStack()}
	}
}

func (m StackStatusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetHeight(m.height - 8)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			return m, m.fetchCmd()
		}

	case stackRefreshMsg:
		var cmds []tea.Cmd
		cmds = append(cmds, m.fetchCmd())
		if m.watch {
			cmds = append(cmds, tickAfter(2*time.Second))
		}
		return m, tea.Batch(cmds...)

	case stackLoadMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.stack = msg.stack
		m.rebuildTable()
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *StackStatusModel) rebuildTable() {
	if m.stack == nil {
		m.table.SetRows(nil)
		return
	}
	var rows []table.Row
	for _, svc := range m.stack.GetServices() {
		cid := shortID(svc.GetContainerId())
		if cid == "" {
			cid = "-"
		}
		health := serviceHealthLabel(svc.GetHealth())
		rows = append(rows, table.Row{
			svc.GetName(),
			cid,
			svc.GetStatus(),
			health,
		})
	}
	m.table.SetRows(rows)
}

func (m StackStatusModel) View() string {
	var b strings.Builder

	stackStatus := StyleMuted.Render("unknown")
	projectName := ""
	if m.stack != nil {
		stackStatus = stackStatusLabel(m.stack.GetStatus())
		projectName = m.stack.GetProjectName()
	}

	title := StyleHeading.Render("Stack Status")
	if projectName != "" {
		title += "  " + StyleMuted.Render(projectName)
	}
	b.WriteString(title + "\n")
	b.WriteString(StyleLabel.Render("status: ") + stackStatus + "\n\n")

	if m.err != nil {
		b.WriteString(StyleError.Render("  "+m.err.Error()) + "\n")
		return b.String()
	}

	if m.stack == nil || len(m.stack.GetServices()) == 0 {
		b.WriteString(StyleMuted.Render("  No services.") + "\n")
	} else {
		b.WriteString(m.table.View() + "\n")
	}

	b.WriteString("\n")
	help := "[r] refresh  [q] quit"
	if m.watch {
		help = "[r] refresh  auto-refresh on  [q] quit"
	}
	b.WriteString(StyleMuted.Render(help) + "\n")
	return b.String()
}

// stackStatusLabel returns a human-readable coloured label for a StackStatus.
func stackStatusLabel(s stackv1.StackStatus) string {
	switch s {
	case stackv1.StackStatus_STACK_STATUS_HEALTHY:
		return StyleSuccess.Render("healthy")
	case stackv1.StackStatus_STACK_STATUS_DEGRADED:
		return StyleWarning.Render("degraded")
	case stackv1.StackStatus_STACK_STATUS_UNHEALTHY:
		return StyleError.Render("unhealthy")
	case stackv1.StackStatus_STACK_STATUS_PROVISIONING:
		return StyleMuted.Render("provisioning")
	case stackv1.StackStatus_STACK_STATUS_TEARDOWN:
		return StyleMuted.Render("teardown")
	default:
		return StyleMuted.Render(fmt.Sprintf("status/%d", s))
	}
}

// serviceHealthLabel returns a human-readable coloured label for ServiceHealth.
func serviceHealthLabel(h stackv1.ServiceHealth) string {
	switch h {
	case stackv1.ServiceHealth_SERVICE_HEALTH_HEALTHY:
		return StyleSuccess.Render("healthy")
	case stackv1.ServiceHealth_SERVICE_HEALTH_STARTING:
		return StyleWarning.Render("starting")
	case stackv1.ServiceHealth_SERVICE_HEALTH_UNHEALTHY:
		return StyleError.Render("unhealthy")
	case stackv1.ServiceHealth_SERVICE_HEALTH_UNKNOWN:
		return StyleMuted.Render("unknown")
	default:
		return StyleMuted.Render("unspecified")
	}
}

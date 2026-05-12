package ui

import (
	"fmt"
	"strings"
	"time"

	adminv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/admin"
	planv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/plan"
	runv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/run"
	stackv1 "github.com/bcp-technology/lobster/gen/go/lobster/v1/stack"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/grpc"
)

// LobbyTab identifies each tab in the tabbed lobby view.
type LobbyTab int

const (
	TabRuns LobbyTab = iota
	TabPlans
	TabStack
	TabAdmin
	tabCount
)

var tabNames = [tabCount]string{
	"Live Runs",
	"History",
	"Stack",
	"Admin",
}

// LobbyModel is the top-level tabbed TUI model.
// It is served by the Wish SSH server and optionally via `lobster tui`.
type LobbyModel struct {
	tab       LobbyTab
	runs      RunsListModel
	plans     PlansListModel
	stack     StackStatusModel
	admin     adminModel
	width     int
	height    int
	workspace string
}

// adminModel holds a simple read-only admin pane.
type adminModel struct {
	client  adminv1.AdminServiceClient
	content string
	loaded  bool
	err     error
}

type adminLoadMsg struct {
	content string
	err     error
}

// NewLobbyModel creates a LobbyModel using the given gRPC connection.
func NewLobbyModel(conn *grpc.ClientConn, workspace string) LobbyModel {
	runClient := runv1.NewRunServiceClient(conn)
	planClient := planv1.NewPlanServiceClient(conn)
	stackClient := stackv1.NewStackServiceClient(conn)
	adminClient := adminv1.NewAdminServiceClient(conn)

	return LobbyModel{
		tab:       TabRuns,
		runs:      NewRunsListModel(runClient, workspace),
		plans:     NewPlansListModel(planClient, workspace),
		stack:     NewStackStatusModel(stackClient, workspace, true),
		admin:     adminModel{client: adminClient},
		workspace: workspace,
		width:     80,
		height:    24,
	}
}

func (m LobbyModel) Init() tea.Cmd {
	return tea.Batch(
		m.runs.Init(),
		m.plans.Init(),
		m.stack.Init(),
		m.loadAdminCmd(),
	)
}

func (m LobbyModel) loadAdminCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := contextWithTimeout(5 * time.Second)
		defer cancel()
		resp, err := m.admin.client.GetHealth(ctx, &adminv1.GetHealthRequest{})
		if err != nil {
			return adminLoadMsg{err: err}
		}
		h := resp.GetHealth()
		live := StyleError.Render("no")
		if h.GetLive() {
			live = StyleSuccess.Render("yes")
		}
		ready := StyleError.Render("no")
		if h.GetReady() {
			ready = StyleSuccess.Render("yes")
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("Live:    %s\n", live))
		b.WriteString(fmt.Sprintf("Ready:   %s\n", ready))
		b.WriteString(fmt.Sprintf("Version: %s\n", h.GetVersion()))
		return adminLoadMsg{content: b.String()}
	}
}

func (m LobbyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		inner := tea.WindowSizeMsg{Width: m.width, Height: m.height - 3}
		// Propagate resize through each sub-model's Update so viewports/tables
		// recalculate their internal dimensions correctly.
		runsUpdated, _ := m.runs.Update(inner)
		if r, ok := runsUpdated.(RunsListModel); ok {
			m.runs = r
		}
		plansUpdated, _ := m.plans.Update(inner)
		if p, ok := plansUpdated.(PlansListModel); ok {
			m.plans = p
		}
		stackUpdated, _ := m.stack.Update(inner)
		if s, ok := stackUpdated.(StackStatusModel); ok {
			m.stack = s
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab", "l":
			m.tab = (m.tab + 1) % tabCount
			return m, nil
		case "shift+tab", "h":
			m.tab = (m.tab + tabCount - 1) % tabCount
			return m, nil
		case "1":
			m.tab = TabRuns
			return m, nil
		case "2":
			m.tab = TabPlans
			return m, nil
		case "3":
			m.tab = TabStack
			return m, nil
		case "4":
			m.tab = TabAdmin
			return m, nil
		}

	case adminLoadMsg:
		if msg.err != nil {
			m.admin.err = msg.err
		} else {
			m.admin.content = msg.content
			m.admin.loaded = true
		}
		return m, nil
	}

	// Delegate to active tab's model.
	var cmd tea.Cmd
	switch m.tab {
	case TabRuns:
		updated, c := m.runs.Update(msg)
		if r, ok := updated.(RunsListModel); ok {
			m.runs = r
		}
		cmd = c
	case TabPlans:
		updated, c := m.plans.Update(msg)
		if p, ok := updated.(PlansListModel); ok {
			m.plans = p
		}
		cmd = c
	case TabStack:
		updated, c := m.stack.Update(msg)
		if s, ok := updated.(StackStatusModel); ok {
			m.stack = s
		}
		cmd = c
	}
	return m, cmd
}

func (m LobbyModel) View() string {
	var b strings.Builder

	// Tab bar.
	b.WriteString(m.renderTabBar())
	b.WriteString("\n")

	// Active pane content.
	switch m.tab {
	case TabRuns:
		b.WriteString(m.runs.View())
	case TabPlans:
		b.WriteString(m.plans.View())
	case TabStack:
		b.WriteString(m.stack.View())
	case TabAdmin:
		b.WriteString(m.adminView())
	}

	return b.String()
}

func (m LobbyModel) renderTabBar() string {
	var parts []string
	for i := LobbyTab(0); i < tabCount; i++ {
		label := fmt.Sprintf(" %s ", tabNames[i])
		if i == m.tab {
			parts = append(parts, lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(colorPrimary).
				Padding(0, 1).
				Render(label))
		} else {
			parts = append(parts, lipgloss.NewStyle().
				Foreground(colorMuted).
				Padding(0, 1).
				Render(label))
		}
	}
	help := StyleMuted.Render("  [tab/1-4] switch tab  [ctrl+c] quit")
	return strings.Join(parts, " ") + help
}

func (m LobbyModel) adminView() string {
	if m.admin.err != nil {
		return StyleError.Render("  "+m.admin.err.Error()) + "\n"
	}
	if !m.admin.loaded {
		return StyleMuted.Render("  Loading admin info…") + "\n"
	}
	return m.admin.content
}

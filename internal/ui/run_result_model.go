package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/bcp-technology-ug/lobster/internal/reports"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ResultAction is the action the user chose on the result screen.
type ResultAction int

const (
	ResultActionQuit  ResultAction = iota
	ResultActionRerun              // user pressed [r] to re-run
)

// clipboardResultMsg is returned by the clipboard copy command.
type clipboardResultMsg struct{ ok bool }

// RunResultModel is the full-screen result page shown after a run finishes.
// It replaces the alt-screen with a centred status card and key-shortcut bar.
type RunResultModel struct {
	result   *reports.RunResult
	runErr   error
	width    int
	height   int
	action   ResultAction
	done     bool
	copied   bool
	showList bool // true when the scenario list overlay is active
	scrollY  int  // scroll offset for the list view
}

// NewRunResultModel creates a RunResultModel from the finished run.
func NewRunResultModel(r *reports.RunResult, runErr error) RunResultModel {
	return RunResultModel{
		result: r,
		runErr: runErr,
		width:  80,
		height: 24,
	}
}

// Action returns the action chosen by the user once the model has exited.
func (m RunResultModel) Action() ResultAction { return m.action }

func (m RunResultModel) Init() tea.Cmd { return nil }

func (m RunResultModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.scrollY > m.maxScroll() {
			m.scrollY = m.maxScroll()
		}

	case tea.KeyMsg:
		// In list view, esc returns to the card; arrow keys scroll.
		if m.showList {
			switch msg.String() {
			case "esc":
				m.showList = false
			case "up", "k":
				if m.scrollY > 0 {
					m.scrollY--
				}
			case "down", "j":
				if m.scrollY < m.maxScroll() {
					m.scrollY++
				}
			case "q", "ctrl+c":
				m.action = ResultActionQuit
				m.done = true
				return m, tea.Quit
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.action = ResultActionQuit
			m.done = true
			return m, tea.Quit
		case "r":
			m.action = ResultActionRerun
			m.done = true
			return m, tea.Quit
		case "l":
			if m.result != nil && len(m.result.Scenarios) > 0 {
				m.showList = true
				m.scrollY = 0
			}
		case "c":
			if m.result != nil && m.result.RunID != "" {
				return m, copyToClipboard(m.result.RunID)
			}
		}

	case clipboardResultMsg:
		m.copied = msg.ok
	}

	return m, nil
}

// ── View ─────────────────────────────────────────────────────────────────────

func (m RunResultModel) View() string {
	if m.showList {
		return m.listView()
	}
	return m.cardView()
}

// listView renders the full-screen scrollable scenario list.
func (m RunResultModel) listView() string {
	var b strings.Builder

	b.WriteString(StyleHeading.Render(IconRocket+" Scenario Results") + "  ")
	if m.result != nil {
		b.WriteString(StyleMuted.Render(fmt.Sprintf("%d total · %d passed · %d failed",
			m.result.Total, m.result.Passed, m.result.Failed)))
	}
	b.WriteString("\n\n")

	lines := m.scenarioLines()
	visible := m.listVisibleRows()
	end := m.scrollY + visible
	if end > len(lines) {
		end = len(lines)
	}
	for _, line := range lines[m.scrollY:end] {
		b.WriteString(line + "\n")
	}

	if len(lines) > visible {
		b.WriteString("\n" + StyleMuted.Render(fmt.Sprintf(
			"↑/↓  row %d–%d of %d", m.scrollY+1, end, len(lines),
		)) + "\n")
	}

	b.WriteString("\n" + StyleMuted.Render("[esc]") + " back  " +
		StyleMuted.Render("[q]") + " quit")

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (m RunResultModel) cardView() string {
	cw := m.cardWidth()
	divider := StyleMuted.Render(strings.Repeat("─", cw))

	var inner strings.Builder

	// ── Status ───────────────────────────────────────────────────────────
	var statusLine string
	switch {
	case m.runErr != nil:
		statusLine = StyleError.Render(IconCross + "  Run Error")
	case m.result == nil:
		statusLine = StyleMuted.Render(IconDot + "  No result")
	case m.result.Failed > 0:
		statusLine = StyleError.Render(IconCross + "  Failed")
	default:
		statusLine = StyleSuccess.Render(IconCheck + "  Passed")
	}
	inner.WriteString(StyleBold.Render(statusLine) + "\n\n")

	// ── Meta ─────────────────────────────────────────────────────────────
	if m.result != nil {
		runIDShort := m.result.RunID
		if len(runIDShort) > 8 {
			runIDShort = runIDShort[:8]
		}
		inner.WriteString(StyleLabel.Render("Run ID:   ") + runIDShort + "\n")
		if !m.result.StartedAt.IsZero() {
			inner.WriteString(StyleLabel.Render("Started:  ") + m.result.StartedAt.Format("02 Jan 2006 15:04:05") + "\n")
			inner.WriteString(StyleLabel.Render("Ended:    ") + m.result.StartedAt.Add(m.result.Duration).Format("02 Jan 2006 15:04:05") + "\n")
		}
		inner.WriteString(StyleLabel.Render("Duration: ") + formatRunDuration(m.result.Duration) + "\n")
		inner.WriteString("\n")

		// ── Stats ─────────────────────────────────────────────────────────
		inner.WriteString(divider + "\n")
		inner.WriteString(fmt.Sprintf(
			"%s  %s  %s  %s",
			StyleBold.Render(fmt.Sprintf("%d total", m.result.Total)),
			StyleSuccess.Render(fmt.Sprintf("%d passed", m.result.Passed)),
			resultFailStyle(m.result.Failed, fmt.Sprintf("%d failed", m.result.Failed)),
			StyleMuted.Render(fmt.Sprintf("%d skipped", m.result.Skipped)),
		) + "\n")
		inner.WriteString(divider + "\n")
	} else if m.runErr != nil {
		inner.WriteString(StyleError.Render(m.runErr.Error()) + "\n")
	}

	// ── Card border ───────────────────────────────────────────────────────
	borderColor := colorSuccess
	if m.result == nil || m.result.Failed > 0 || m.runErr != nil {
		borderColor = colorError
	}
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(cw).
		Render(inner.String())

	// ── Action bar ────────────────────────────────────────────────────────
	copiedHint := ""
	if m.copied {
		copiedHint = " " + StyleSuccess.Render("(copied!)")
	}
	listHint := ""
	if m.result != nil && len(m.result.Scenarios) > 0 {
		listHint = "  " + StyleMuted.Render("[l]") + " list"
	}
	actionBar := StyleMuted.Render("[r]") + " re-run" +
		listHint + "  " +
		StyleMuted.Render("[c]") + " copy run id" + copiedHint + "  " +
		StyleMuted.Render("[q]") + " quit"

	content := card + "\n\n" + actionBar

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// listVisibleRows returns how many scenario lines fit in the list view.
func (m RunResultModel) listVisibleRows() int {
	// header(2) + blank(1) + scroll hint(2) + blank(1) + action bar(1) + padding(2)
	const reserved = 9
	n := m.height - reserved
	if n < 3 {
		return 3
	}
	return n
}

// maxScroll returns the maximum scroll offset for the list view.
func (m RunResultModel) maxScroll() int {
	n := len(m.scenarioLines()) - m.listVisibleRows()
	if n < 0 {
		return 0
	}
	return n
}

// scenarioLines builds the flat list of lines for the scenario list view.
func (m RunResultModel) scenarioLines() []string {
	if m.result == nil {
		return nil
	}
	var lines []string
	for _, sc := range m.result.Scenarios {
		icon, style := scenarioStatusIcon(sc.Status, false)
		dur := ""
		if sc.Duration > 0 {
			dur = "  " + StyleMuted.Render(formatRunDuration(sc.Duration))
		}
		lines = append(lines, fmt.Sprintf("  %s  %s%s",
			style.Render(icon),
			StyleBold.Render(sc.FeatureName+" › "+sc.Name),
			dur,
		))
		if sc.Status == reports.StatusFailed {
			for _, step := range sc.Steps {
				if step.Status == reports.StatusFailed {
					lines = append(lines, "       "+StyleMuted.Render(step.Keyword+" "+step.Text))
					if step.Err != nil {
						lines = append(lines, "          "+StyleError.Render(step.Err.Error()))
					}
				}
			}
			if sc.Err != nil {
				lines = append(lines, "       "+StyleError.Render(sc.Err.Error()))
			}
		}
	}
	return lines
}

// cardWidth returns the inner width of the card, clamped to the terminal.
func (m RunResultModel) cardWidth() int {
	w := m.width - 8
	if w > 88 {
		w = 88
	}
	if w < 40 {
		w = 40
	}
	return w
}

// resultFailStyle applies error styling only when the count is non-zero.
func resultFailStyle(count int, text string) string {
	if count > 0 {
		return StyleError.Render(text)
	}
	return StyleMuted.Render(text)
}

// copyToClipboard returns a tea.Cmd that writes text to the system clipboard
// using platform-native tools (pbcopy / clip / xclip / xsel).
func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("pbcopy")
		case "windows":
			cmd = exec.Command("clip")
		default:
			if _, err := exec.LookPath("xclip"); err == nil {
				cmd = exec.Command("xclip", "-selection", "c")
			} else if _, err := exec.LookPath("xsel"); err == nil {
				cmd = exec.Command("xsel", "--clipboard", "--input")
			} else {
				return clipboardResultMsg{ok: false}
			}
		}
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err != nil {
			return clipboardResultMsg{ok: false}
		}
		return clipboardResultMsg{ok: true}
	}
}

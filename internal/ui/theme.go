package ui

import "github.com/charmbracelet/lipgloss"

// Adaptive colors: look great on both dark and light terminals.
var (
	colorPrimary   = lipgloss.AdaptiveColor{Light: "#6D28D9", Dark: "#A78BFA"}
	colorSecondary = lipgloss.AdaptiveColor{Light: "#4F46E5", Dark: "#818CF8"}
	colorSuccess   = lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"}
	colorWarning   = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FCD34D"}
	colorError     = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#FCA5A5"}
	colorMuted     = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	colorBorder    = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#374151"}
	colorHighlight = lipgloss.AdaptiveColor{Light: "#EDE9FE", Dark: "#1E1B4B"}
)

// Text styles — compose these when building command output.
var (
	StyleHeading = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	StyleSubheading = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSecondary)

	StyleMuted = lipgloss.NewStyle().
			Foreground(colorMuted)

	StyleSuccess = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSuccess)

	StyleWarning = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWarning)

	StyleError = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorError)

	StyleCode = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Background(colorHighlight).
			Padding(0, 1)

	StyleBold = lipgloss.NewStyle().Bold(true)

	StyleLabel = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorMuted)
)

// Icon constants — unicode symbols used consistently across commands.
const (
	IconCheck   = "✓"
	IconCross   = "✗"
	IconWarning = "⚠"
	IconInfo    = "ℹ"
	IconArrow   = "→"
	IconDot     = "·"
	IconRocket  = "🚀"
)

// ─────────────────────────────────────────────────────────────────────────────
// TUI design tokens — used exclusively by the lobby and pane models.
// ─────────────────────────────────────────────────────────────────────────────

// colorAccent is a cyan complement to the violet primary, used in the TUI.
var colorAccent = lipgloss.AdaptiveColor{Light: "#0891B2", Dark: "#22D3EE"} //nolint:unused

// TUI layout styles.
var (
	// TUITabActive is the filled-pill style for the currently selected tab.
	TUITabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorPrimary).
			Padding(0, 2)

	// TUITabInactive is the plain-text style for non-selected tabs.
	TUITabInactive = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 2)

	// TUICardStyle wraps the main content area of each TUI pane.
	TUICardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#6D28D9"}).
			Padding(1, 2)

	// TUICardHeaderStyle renders the pane title inside a card.
	TUICardHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary)

	// TUILogoStyle renders the top lobster wordmark.
	TUILogoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	// TUIFooterKeyStyle highlights a key name in footer hint lines.
	TUIFooterKeyStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorSecondary)
)

// Status badge styles — colored background pills used in run/service status cells.
var (
	BadgeRunning = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "#0E7490", Dark: "#155E75"}).
			Foreground(lipgloss.Color("#E0FFFF")).
			Bold(true).Padding(0, 1)

	BadgePassed = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "#166534", Dark: "#14532D"}).
			Foreground(lipgloss.Color("#D1FAE5")).
			Bold(true).Padding(0, 1)

	BadgeFailed = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "#991B1B", Dark: "#7F1D1D"}).
			Foreground(lipgloss.Color("#FEE2E2")).
			Bold(true).Padding(0, 1)

	BadgePending = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "#92400E", Dark: "#78350F"}).
			Foreground(lipgloss.Color("#FEF3C7")).
			Bold(true).Padding(0, 1)

	BadgeCancelled = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "#374151", Dark: "#1F2937"}).
			Foreground(lipgloss.Color("#F3F4F6")).
			Bold(true).Padding(0, 1)
)

// TUICenter centers s horizontally inside a rendered block of totalWidth chars.
func TUICenter(totalWidth int, s string) string {
	return lipgloss.NewStyle().Width(totalWidth).Align(lipgloss.Center).Render(s)
}

// renderKeyHint returns a styled "key  action" string for footer hint bars.
func renderKeyHint(key, action string) string {
	return TUIFooterKeyStyle.Render(key) + " " + StyleMuted.Render(action)
}

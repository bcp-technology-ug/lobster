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

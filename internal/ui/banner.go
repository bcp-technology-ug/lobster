package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func bannerStyle(border lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 2).
		Width(70)
}

// RenderSuccess renders a green success panel.
func RenderSuccess(title, body string) string {
	return renderBanner(lipgloss.Color("#4ADE80"), StyleSuccess, IconCheck, title, body)
}

// RenderInfo renders a blue/indigo info panel.
func RenderInfo(title, body string) string {
	return renderBanner(lipgloss.Color("#818CF8"), StyleSubheading, IconInfo, title, body)
}

// RenderWarning renders an amber warning panel.
func RenderWarning(title, body string) string {
	return renderBanner(lipgloss.Color("#FCD34D"), StyleWarning, IconWarning, title, body)
}

// RenderComingSoon renders a muted panel for reserved/not-yet-implemented commands.
func RenderComingSoon(command string) string {
	title := command + " — coming soon"
	body := "This command is not yet available in the current release.\n" +
		StyleMuted.Render("Check the docs or run "+StyleCode.Render("lobster --help")+" for available commands.")
	return renderBanner(lipgloss.Color("#9CA3AF"), StyleMuted, IconRocket, title, body)
}

func renderBanner(borderColor lipgloss.Color, titleStyle lipgloss.Style, icon, title, body string) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(icon + "  " + title))
	if body != "" {
		b.WriteString("\n")
		b.WriteString(body)
	}
	return fmt.Sprintf("\n%s\n", bannerStyle(borderColor).Render(b.String()))
}

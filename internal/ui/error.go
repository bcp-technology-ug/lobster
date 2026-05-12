package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var errorPanel = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorError).
	Padding(1, 2).
	Width(70)

var errorTitle = lipgloss.NewStyle().
	Bold(true).
	Foreground(colorError)

var hintStyle = lipgloss.NewStyle().
	Foreground(colorMuted).
	Italic(true)

var docLinkStyle = lipgloss.NewStyle().
	Foreground(colorSecondary).
	Underline(true)

// RenderError renders a rich diagnostic error panel.
//   - title:  short error type / command name
//   - msg:    full error message (may be multi-line)
//   - hint:   optional suggestion to resolve (empty to omit)
//   - docURL: optional documentation link (empty to omit)
func RenderError(title, msg, hint, docURL string) string {
	var b strings.Builder

	b.WriteString(errorTitle.Render(IconCross + "  " + title))
	b.WriteString("\n\n")
	b.WriteString(msg)

	if hint != "" {
		b.WriteString("\n\n")
		b.WriteString(StyleLabel.Render("Hint  "))
		b.WriteString(hintStyle.Render(hint))
	}
	if docURL != "" {
		b.WriteString("\n")
		b.WriteString(StyleLabel.Render("Docs  "))
		b.WriteString(docLinkStyle.Render(docURL))
	}

	return fmt.Sprintf("\n%s\n", errorPanel.Render(b.String()))
}

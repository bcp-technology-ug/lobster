package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	tableOuterStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1).
			Width(70)

	tableTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	tableKeyStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(34)

	tableValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#111827", Dark: "#F9FAFB"})
)

// RenderKeyValueTable renders a titled two-column table inside a rounded border.
// rows is a slice of [key, value] pairs. Pass an empty title to omit the header.
func RenderKeyValueTable(title string, rows [][2]string) string {
	var b strings.Builder

	if title != "" {
		b.WriteString(tableTitleStyle.Render(title))
		b.WriteString("\n")
	}

	for i, row := range rows {
		k, v := row[0], row[1]
		b.WriteString(tableKeyStyle.Render(k))
		b.WriteString(tableValueStyle.Render(v))
		if i < len(rows)-1 {
			b.WriteString("\n")
		}
	}

	return fmt.Sprintf("\n%s\n", tableOuterStyle.Render(b.String()))
}

// RenderSectionedTable renders multiple named sections, each a key-value table,
// separated by a blank line — useful for the config command.
func RenderSectionedTable(sections []Section) string {
	var parts []string
	for _, s := range sections {
		parts = append(parts, RenderKeyValueTable(s.Title, s.Rows))
	}
	return strings.Join(parts, "\n")
}

// Section is a titled group of key-value rows.
type Section struct {
	Title string
	Rows  [][2]string
}

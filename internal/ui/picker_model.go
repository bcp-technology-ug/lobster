package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bcp-technology-ug/lobster/internal/parser"
)

// ScenarioPickerModel is a Bubbletea model for interactively selecting
// which scenarios to run. It accepts a flat list of parser.Scenario entries
// and lets the user toggle individual items before confirming.
type ScenarioPickerModel struct {
	allItems  []pickerItem
	filtered  []pickerItem
	cursor    int
	filter    textinput.Model
	filtering bool
	done      bool
	width     int
	height    int
}

type pickerItem struct {
	id       string // DeterministicID
	feature  string // feature URI / name
	name     string // scenario name
	tags     []string
	selected bool
}

// NewScenarioPickerModel builds a picker from a set of parsed features.
func NewScenarioPickerModel(features []*parser.Feature) ScenarioPickerModel {
	var items []pickerItem
	for _, f := range features {
		featureLabel := f.Name
		if featureLabel == "" {
			featureLabel = f.URI
		}
		for _, sc := range f.Scenarios {
			items = append(items, pickerItem{
				id:      sc.DeterministicID,
				feature: featureLabel,
				name:    sc.Name,
				tags:    sc.Tags,
			})
		}
	}

	ti := textinput.New()
	ti.Placeholder = "fuzzy filter…"
	ti.Width = 40

	m := ScenarioPickerModel{
		allItems: items,
		filter:   ti,
		width:    80,
		height:   24,
	}
	m.applyFilter()
	return m
}

// SelectedIDs returns the DeterministicIDs of all selected scenarios.
// Returns nil (meaning "run all") if nothing is selected.
func (m ScenarioPickerModel) SelectedIDs() []string {
	var ids []string
	for _, it := range m.allItems {
		if it.selected {
			ids = append(ids, it.id)
		}
	}
	return ids
}

// Done returns true when the user has confirmed their selection.
func (m ScenarioPickerModel) Done() bool { return m.done }

func (m ScenarioPickerModel) Init() tea.Cmd {
	return nil
}

func (m ScenarioPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.filtering {
			switch msg.String() {
			case "esc":
				m.filtering = false
				m.filter.Blur()
			case "enter":
				m.filtering = false
				m.filter.Blur()
			default:
				var cmd tea.Cmd
				m.filter, cmd = m.filter.Update(msg)
				m.applyFilter()
				m.cursor = 0
				return m, cmd
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.done = true
			return m, tea.Quit

		case "enter":
			// Confirm selection.
			m.done = true
			return m, tea.Quit

		case " ":
			// Toggle current item.
			if m.cursor < len(m.filtered) {
				id := m.filtered[m.cursor].id
				for i := range m.allItems {
					if m.allItems[i].id == id {
						m.allItems[i].selected = !m.allItems[i].selected
						break
					}
				}
				m.applyFilter()
			}

		case "a":
			// Select all visible.
			for _, fi := range m.filtered {
				for i := range m.allItems {
					if m.allItems[i].id == fi.id {
						m.allItems[i].selected = true
					}
				}
			}
			m.applyFilter()

		case "n":
			// Deselect all.
			for i := range m.allItems {
				m.allItems[i].selected = false
			}
			m.applyFilter()

		case "/":
			m.filtering = true
			m.filter.Focus()
			return m, textinput.Blink

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m *ScenarioPickerModel) applyFilter() {
	filterText := strings.ToLower(m.filter.Value())
	m.filtered = m.filtered[:0]
	for _, it := range m.allItems {
		if filterText == "" ||
			strings.Contains(strings.ToLower(it.name), filterText) ||
			strings.Contains(strings.ToLower(it.feature), filterText) {
			// Sync selection state.
			m.filtered = append(m.filtered, it)
		}
	}
}

func (m ScenarioPickerModel) View() string {
	var b strings.Builder

	b.WriteString(StyleHeading.Render("Select Scenarios") + "\n")
	b.WriteString(StyleMuted.Render(fmt.Sprintf(
		"  %d/%d selected  [space] toggle  [a] all  [n] none  [/] filter  [enter] run  [q] cancel",
		m.countSelected(), len(m.allItems))) + "\n\n")

	if m.filtering {
		b.WriteString("  Filter: " + m.filter.View() + "\n\n")
	}

	// Visible window.
	visibleLines := m.height - 6
	if visibleLines < 5 {
		visibleLines = 5
	}
	start := 0
	if m.cursor >= visibleLines {
		start = m.cursor - visibleLines + 1
	}
	end := start + visibleLines
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	lastFeature := ""
	for i := start; i < end; i++ {
		it := m.filtered[i]

		if it.feature != lastFeature {
			b.WriteString("  " + StyleSubheading.Render(it.feature) + "\n")
			lastFeature = it.feature
		}

		checkbox := "[ ]"
		checkStyle := lipgloss.NewStyle()
		if it.selected {
			checkbox = "[" + StyleSuccess.Render("✓") + "]"
		}

		cursor := "  "
		if i == m.cursor {
			cursor = StyleBold.Render("▸ ")
			checkStyle = StyleBold
		}

		tags := ""
		if len(it.tags) > 0 {
			tags = " " + StyleMuted.Render(strings.Join(it.tags, " "))
		}

		fmt.Fprintf(&b, "%s%s %s%s\n",
			cursor,
			checkbox,
			checkStyle.Render(it.name),
			tags)
	}

	if len(m.filtered) == 0 {
		b.WriteString(StyleMuted.Render("  No matching scenarios.") + "\n")
	}

	return b.String()
}

func (m ScenarioPickerModel) countSelected() int {
	n := 0
	for _, it := range m.allItems {
		if it.selected {
			n++
		}
	}
	return n
}

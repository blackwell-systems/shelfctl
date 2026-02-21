package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// EditFormData holds the metadata collected from the user.
type EditFormData struct {
	Title  string
	Author string
	Year   int
	Tags   string // Comma-separated
}

// EditFormDefaults provides default values for form fields.
type EditFormDefaults struct {
	BookID string
	Title  string
	Author string
	Year   int
	Tags   []string
}

type editFormModel struct {
	inputs   []textinput.Model
	focused  int
	defaults EditFormDefaults
	result   *EditFormData
	err      error
	canceled bool
}

const (
	editFieldTitle = iota
	editFieldAuthor
	editFieldYear
	editFieldTags
)

func newEditForm(defaults EditFormDefaults) editFormModel {
	m := editFormModel{
		inputs:   make([]textinput.Model, 4),
		defaults: defaults,
	}

	// Title field
	m.inputs[editFieldTitle] = textinput.New()
	m.inputs[editFieldTitle].Placeholder = defaults.Title
	m.inputs[editFieldTitle].SetValue(defaults.Title)
	m.inputs[editFieldTitle].Focus()
	m.inputs[editFieldTitle].CharLimit = 200
	m.inputs[editFieldTitle].Width = 50

	// Author field
	m.inputs[editFieldAuthor] = textinput.New()
	m.inputs[editFieldAuthor].Placeholder = "Author name"
	m.inputs[editFieldAuthor].SetValue(defaults.Author)
	m.inputs[editFieldAuthor].CharLimit = 100
	m.inputs[editFieldAuthor].Width = 50

	// Year field
	m.inputs[editFieldYear] = textinput.New()
	m.inputs[editFieldYear].Placeholder = "Publication year (e.g., 2023)"
	if defaults.Year > 0 {
		m.inputs[editFieldYear].SetValue(strconv.Itoa(defaults.Year))
	}
	m.inputs[editFieldYear].CharLimit = 4
	m.inputs[editFieldYear].Width = 50

	// Tags field
	m.inputs[editFieldTags] = textinput.New()
	m.inputs[editFieldTags].Placeholder = "comma,separated,tags"
	if len(defaults.Tags) > 0 {
		m.inputs[editFieldTags].SetValue(strings.Join(defaults.Tags, ","))
	}
	m.inputs[editFieldTags].CharLimit = 200
	m.inputs[editFieldTags].Width = 50

	return m
}

func (m editFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m editFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			return m, tea.Quit

		case "enter":
			// Parse year
			yearVal := 0
			if yearStr := m.inputs[editFieldYear].Value(); yearStr != "" {
				year, err := strconv.Atoi(yearStr)
				if err != nil || year < 0 || year > 9999 {
					m.err = fmt.Errorf("invalid year (must be 0-9999)")
					return m, nil
				}
				yearVal = year
			}

			// Submit the form
			m.result = &EditFormData{
				Title:  m.inputs[editFieldTitle].Value(),
				Author: m.inputs[editFieldAuthor].Value(),
				Year:   yearVal,
				Tags:   m.inputs[editFieldTags].Value(),
			}
			return m, tea.Quit

		case "tab", "shift+tab", "up", "down":
			// Navigate between fields
			if msg.String() == "up" || msg.String() == "shift+tab" {
				m.focused--
			} else {
				m.focused++
			}

			if m.focused < 0 {
				m.focused = len(m.inputs) - 1
			} else if m.focused >= len(m.inputs) {
				m.focused = 0
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focused {
					cmds[i] = m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}
			return m, tea.Batch(cmds...)
		}
	}

	// Update the focused input
	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *editFormModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m editFormModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(StyleHeader.Render(fmt.Sprintf("Edit Book: %s", m.defaults.BookID)))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(StyleNormal.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	// Form fields
	fields := []string{"Title", "Author", "Year", "Tags"}
	for i, label := range fields {
		if i == m.focused {
			b.WriteString(StyleHighlight.Render("› " + label + ":"))
		} else {
			b.WriteString(StyleNormal.Render("  " + label + ":"))
		}
		b.WriteString("\n  ")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}

	// Help text
	b.WriteString("\n")
	b.WriteString(StyleHelp.Render("Tab/↑↓: Navigate  Enter: Submit  Esc: Cancel"))
	b.WriteString("\n")

	return b.String()
}

// RunEditForm launches an interactive form for editing book metadata.
// Returns the filled form data, or error if canceled.
func RunEditForm(defaults EditFormDefaults) (*EditFormData, error) {
	m := newEditForm(defaults)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("running form: %w", err)
	}

	fm, ok := finalModel.(editFormModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	if fm.canceled {
		return nil, fmt.Errorf("canceled")
	}

	if fm.result == nil {
		return nil, fmt.Errorf("no data collected")
	}

	return fm.result, nil
}

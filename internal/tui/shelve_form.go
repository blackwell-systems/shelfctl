package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ShelveFormData holds the metadata collected from the user.
type ShelveFormData struct {
	Title  string
	Author string
	Tags   string // Comma-separated
	ID     string
}

// ShelveFormDefaults provides default values for form fields.
type ShelveFormDefaults struct {
	Filename string
	Title    string
	Author   string
	ID       string
}

type shelveFormModel struct {
	inputs   []textinput.Model
	focused  int
	defaults ShelveFormDefaults
	result   *ShelveFormData
	err      error
	canceled bool
}

const (
	fieldTitle = iota
	fieldAuthor
	fieldTags
	fieldID
)

func newShelveForm(defaults ShelveFormDefaults) shelveFormModel {
	m := shelveFormModel{
		inputs:   make([]textinput.Model, 4),
		defaults: defaults,
	}

	// Title field
	m.inputs[fieldTitle] = textinput.New()
	m.inputs[fieldTitle].Placeholder = defaults.Title
	m.inputs[fieldTitle].Focus()
	m.inputs[fieldTitle].CharLimit = 200
	m.inputs[fieldTitle].Width = 50

	// Author field
	m.inputs[fieldAuthor] = textinput.New()
	if defaults.Author != "" {
		m.inputs[fieldAuthor].Placeholder = defaults.Author
	} else {
		m.inputs[fieldAuthor].Placeholder = "Author name"
	}
	m.inputs[fieldAuthor].CharLimit = 100
	m.inputs[fieldAuthor].Width = 50

	// Tags field
	m.inputs[fieldTags] = textinput.New()
	m.inputs[fieldTags].Placeholder = "comma,separated,tags"
	m.inputs[fieldTags].CharLimit = 200
	m.inputs[fieldTags].Width = 50

	// ID field
	m.inputs[fieldID] = textinput.New()
	m.inputs[fieldID].Placeholder = defaults.ID
	m.inputs[fieldID].CharLimit = 63
	m.inputs[fieldID].Width = 50

	return m
}

func (m shelveFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m shelveFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			return m, tea.Quit

		case "enter":
			// Submit the form
			m.result = &ShelveFormData{
				Title:  m.getValue(fieldTitle),
				Author: m.inputs[fieldAuthor].Value(),
				Tags:   m.inputs[fieldTags].Value(),
				ID:     m.getValue(fieldID),
			}
			return m, tea.Quit

		case "tab", "down":
			// Move to next field
			m.inputs[m.focused].Blur()
			m.focused = (m.focused + 1) % len(m.inputs)
			m.inputs[m.focused].Focus()
			return m, m.inputs[m.focused].Focus()

		case "shift+tab", "up":
			// Move to previous field
			m.inputs[m.focused].Blur()
			m.focused--
			if m.focused < 0 {
				m.focused = len(m.inputs) - 1
			}
			m.inputs[m.focused].Focus()
			return m, m.inputs[m.focused].Focus()
		}
	}

	// Update the focused input
	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

func (m shelveFormModel) View() string {
	if m.canceled {
		return ""
	}

	var b strings.Builder

	// Header
	b.WriteString(StyleHeader.Render("Add Book to Library"))
	b.WriteString("\n\n")

	// File info
	b.WriteString(StyleHelp.Render(fmt.Sprintf("File: %s", m.defaults.Filename)))
	b.WriteString("\n\n")

	// Form fields
	fields := []string{"Title", "Author", "Tags", "ID"}
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

// getValue returns the input value or the placeholder default if empty.
func (m shelveFormModel) getValue(field int) string {
	val := m.inputs[field].Value()
	if val != "" {
		return val
	}
	return m.inputs[field].Placeholder
}

// RunShelveForm launches an interactive form for book metadata entry.
// Returns the filled form data, or error if canceled.
func RunShelveForm(defaults ShelveFormDefaults) (*ShelveFormData, error) {
	m := newShelveForm(defaults)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("running form: %w", err)
	}

	fm, ok := finalModel.(shelveFormModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	if fm.canceled {
		return nil, fmt.Errorf("canceled by user")
	}

	return fm.result, nil
}

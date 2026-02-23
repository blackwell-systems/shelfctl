package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ShelveFormData holds the metadata collected from the user.
type ShelveFormData struct {
	Title  string
	Author string
	Tags   string // Comma-separated
	ID     string
	Cache  bool // Whether to cache locally after upload
}

// ShelveFormDefaults provides default values for form fields.
type ShelveFormDefaults struct {
	Filename string
	Title    string
	Author   string
	ID       string
}

type shelveFormModel struct {
	inputs       []textinput.Model
	focused      int
	defaults     ShelveFormDefaults
	result       *ShelveFormData
	err          error
	canceled     bool
	width        int
	height       int
	cacheLocally bool // Checkbox state
}

const (
	fieldTitle = iota
	fieldAuthor
	fieldTags
	fieldID
	fieldCache // Checkbox position
)

func newShelveForm(defaults ShelveFormDefaults) shelveFormModel {
	m := shelveFormModel{
		inputs:       make([]textinput.Model, 4), // Still 4 text inputs, checkbox is separate
		defaults:     defaults,
		cacheLocally: true, // Default to caching
	}

	const inputWidth = 50
	const maxPlaceholderWidth = 45 // Leave room for borders and padding

	// Truncate long placeholders to prevent overflow
	truncate := func(s string, max int) string {
		if len(s) > max {
			return s[:max-1] + "…"
		}
		return s
	}

	// Title field
	m.inputs[fieldTitle] = textinput.New()
	m.inputs[fieldTitle].Placeholder = truncate(defaults.Title, maxPlaceholderWidth)
	m.inputs[fieldTitle].Focus()
	m.inputs[fieldTitle].CharLimit = 200
	m.inputs[fieldTitle].Width = inputWidth
	m.inputs[fieldTitle].Prompt = ""

	// Author field
	m.inputs[fieldAuthor] = textinput.New()
	if defaults.Author != "" {
		m.inputs[fieldAuthor].Placeholder = truncate(defaults.Author, maxPlaceholderWidth)
	} else {
		m.inputs[fieldAuthor].Placeholder = "Author name"
	}
	m.inputs[fieldAuthor].CharLimit = 100
	m.inputs[fieldAuthor].Width = inputWidth
	m.inputs[fieldAuthor].Prompt = ""

	// Tags field
	m.inputs[fieldTags] = textinput.New()
	m.inputs[fieldTags].Placeholder = "comma,separated,tags"
	m.inputs[fieldTags].CharLimit = 200
	m.inputs[fieldTags].Width = inputWidth
	m.inputs[fieldTags].Prompt = ""

	// ID field
	m.inputs[fieldID] = textinput.New()
	m.inputs[fieldID].Placeholder = truncate(defaults.ID, maxPlaceholderWidth)
	m.inputs[fieldID].CharLimit = 63
	m.inputs[fieldID].Width = inputWidth
	m.inputs[fieldID].Prompt = ""

	return m
}

func (m shelveFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m shelveFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

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
				Cache:  m.cacheLocally,
			}
			return m, tea.Quit

		case " ":
			// Toggle checkbox if focused on it
			if m.focused == fieldCache {
				m.cacheLocally = !m.cacheLocally
				return m, nil
			}

		case "tab", "down":
			// Move to next field (including checkbox)
			if m.focused < fieldID {
				m.inputs[m.focused].Blur()
			}
			m.focused = (m.focused + 1) % (fieldCache + 1)
			if m.focused < fieldID+1 {
				m.inputs[m.focused].Focus()
				return m, m.inputs[m.focused].Focus()
			}
			return m, nil

		case "shift+tab", "up":
			// Move to previous field (including checkbox)
			if m.focused < fieldID+1 && m.focused >= 0 {
				m.inputs[m.focused].Blur()
			}
			m.focused--
			if m.focused < 0 {
				m.focused = fieldCache
			}
			if m.focused < fieldID+1 {
				m.inputs[m.focused].Focus()
				return m, m.inputs[m.focused].Focus()
			}
			return m, nil
		}
	}

	// Update the focused input (only if it's a text input field, not the checkbox)
	var cmd tea.Cmd
	if m.focused < fieldID+1 {
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	}
	return m, cmd
}

func (m shelveFormModel) View() string {
	if m.canceled {
		return ""
	}

	// Outer container for centering - same as hub
	outerStyle := lipgloss.NewStyle().
		Padding(2, 4) // top/bottom: 2 lines, left/right: 4 chars

	var b strings.Builder

	// Header
	b.WriteString(StyleHeader.Render("Add Book to Library"))
	b.WriteString("\n\n")

	// File info - truncate long filenames to fit within border
	filename := m.defaults.Filename
	const maxFilenameWidth = 60 // Keep it under the input width + padding
	if len(filename) > maxFilenameWidth {
		filename = filename[:maxFilenameWidth-1] + "…"
	}
	b.WriteString(StyleHelp.Render(fmt.Sprintf("File: %s", filename)))
	b.WriteString("\n\n")

	// Form fields
	fields := []string{"Title", "Author", "Tags", "ID"}
	for i, label := range fields {
		if i == m.focused {
			b.WriteString(StyleHighlight.Render("> " + label + ":"))
		} else {
			b.WriteString(StyleNormal.Render("  " + label + ":"))
		}
		b.WriteString("\n  ")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}

	// Checkbox for caching
	checkboxLabel := "Cache locally"
	checkbox := "[ ]"
	if m.cacheLocally {
		checkbox = "[✓]"
	}
	if m.focused == fieldCache {
		b.WriteString(StyleHighlight.Render("> " + checkboxLabel + ": " + checkbox))
	} else {
		b.WriteString(StyleNormal.Render("  " + checkboxLabel + ": " + checkbox))
	}
	b.WriteString("\n")
	b.WriteString(StyleHelp.Render("  (Space to toggle)"))
	b.WriteString("\n")

	// Help text
	b.WriteString("\n")
	b.WriteString(StyleHelp.Render("Tab/↑↓: Navigate  Space: Toggle  Enter: Submit  Esc: Cancel"))
	b.WriteString("\n")

	content := b.String()

	// Add inner padding inside border (same as hub)
	innerPadding := lipgloss.NewStyle().
		Padding(0, 2, 0, 1) // top, right, bottom, left

	// Apply inner padding, then border, then outer padding
	return outerStyle.Render(StyleBorder.Render(innerPadding.Render(content)))
}

// getValue returns the input value or the original default if empty.
func (m shelveFormModel) getValue(field int) string {
	val := m.inputs[field].Value()
	if val != "" {
		return val
	}
	// Return original defaults, not the truncated placeholder
	switch field {
	case fieldTitle:
		return m.defaults.Title
	case fieldAuthor:
		return m.defaults.Author
	case fieldID:
		return m.defaults.ID
	default:
		return ""
	}
}

// RunShelveForm launches an interactive form for book metadata entry.
// Returns the filled form data, or error if canceled.
func RunShelveForm(defaults ShelveFormDefaults) (*ShelveFormData, error) {
	m := newShelveForm(defaults)
	p := tea.NewProgram(m, tea.WithAltScreen())

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

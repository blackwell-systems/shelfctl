package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	inputs      []textinput.Model
	focused     int
	defaults    EditFormDefaults
	result      *EditFormData
	err         error
	canceled    bool
	width       int
	height      int
	confirming  bool
	confirmResp string
	activeCmd   string
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

	const fieldWidth = 42

	// Title field
	m.inputs[editFieldTitle] = textinput.New()
	m.inputs[editFieldTitle].Placeholder = "Book title"
	m.inputs[editFieldTitle].SetValue(defaults.Title)
	m.inputs[editFieldTitle].Focus()
	m.inputs[editFieldTitle].CharLimit = 200
	m.inputs[editFieldTitle].Width = fieldWidth
	m.inputs[editFieldTitle].Prompt = "│ "

	// Author field
	m.inputs[editFieldAuthor] = textinput.New()
	m.inputs[editFieldAuthor].Placeholder = "Author name"
	m.inputs[editFieldAuthor].SetValue(defaults.Author)
	m.inputs[editFieldAuthor].CharLimit = 100
	m.inputs[editFieldAuthor].Width = fieldWidth
	m.inputs[editFieldAuthor].Prompt = "│ "

	// Year field
	m.inputs[editFieldYear] = textinput.New()
	m.inputs[editFieldYear].Placeholder = "2024"
	if defaults.Year > 0 {
		m.inputs[editFieldYear].SetValue(strconv.Itoa(defaults.Year))
	}
	m.inputs[editFieldYear].CharLimit = 4
	m.inputs[editFieldYear].Width = 8
	m.inputs[editFieldYear].Prompt = "│ "

	// Tags field
	m.inputs[editFieldTags] = textinput.New()
	m.inputs[editFieldTags].Placeholder = "comma,separated,tags"
	if len(defaults.Tags) > 0 {
		m.inputs[editFieldTags].SetValue(strings.Join(defaults.Tags, ","))
	}
	m.inputs[editFieldTags].CharLimit = 200
	m.inputs[editFieldTags].Width = fieldWidth
	m.inputs[editFieldTags].Prompt = "│ "

	return m
}

func (m editFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m editFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ClearActiveCmdMsg:
		m.activeCmd = ""
		return m, nil

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
			if m.confirming {
				// Treat enter as "y" (yes)
				m.confirmResp = "y"

				// Parse year
				yearVal := 0
				if yearStr := m.inputs[editFieldYear].Value(); yearStr != "" {
					year, err := strconv.Atoi(yearStr)
					if err != nil || year < 0 || year > 9999 {
						m.err = fmt.Errorf("invalid year (must be 0-9999)")
						m.confirming = false
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
			}

			// Show confirmation prompt
			m.confirming = true
			return m, nil

		case "y", "Y":
			if m.confirming {
				m.confirmResp = "y"

				// Parse year
				yearVal := 0
				if yearStr := m.inputs[editFieldYear].Value(); yearStr != "" {
					year, err := strconv.Atoi(yearStr)
					if err != nil || year < 0 || year > 9999 {
						m.err = fmt.Errorf("invalid year (must be 0-9999)")
						m.confirming = false
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
			}

		case "n", "N":
			if m.confirming {
				m.canceled = true
				return m, tea.Quit
			}

		case "tab", "shift+tab", "up", "down":
			// Don't allow navigation during confirmation
			if m.confirming {
				return m, nil
			}

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

			cmds := make([]tea.Cmd, len(m.inputs)+1)
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focused {
					cmds[i] = m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}
			m.activeCmd = "tab"
			cmds[len(m.inputs)] = HighlightCmd()
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
	outerStyle := lipgloss.NewStyle().Padding(2, 4)

	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#D0D0D0", Dark: "#444444"})
	formLabel := lipgloss.NewStyle().
		Foreground(ColorGray).
		Width(10).
		Align(lipgloss.Right).
		PaddingRight(1)
	formLabelActive := lipgloss.NewStyle().
		Foreground(ColorYellow).
		Bold(true).
		Width(10).
		Align(lipgloss.Right).
		PaddingRight(1)

	const w = 54
	sep := sepStyle.Render(strings.Repeat("─", w))

	var b strings.Builder

	// ── Header ──
	b.WriteString(StyleHeader.Render("Edit Book"))
	b.WriteString("\n")
	b.WriteString(StyleHelp.Render(m.defaults.BookID))
	b.WriteString("\n")
	if len(m.defaults.Tags) > 0 {
		b.WriteString(StyleTag.Render(strings.Join(m.defaults.Tags, ", ")))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(sep)
	b.WriteString("\n\n")

	// ── Error ──
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	// ── Form fields ──
	fields := []string{"Title", "Author", "Year", "Tags"}
	for i, label := range fields {
		if i == m.focused && !m.confirming {
			b.WriteString(formLabelActive.Render("› " + label))
		} else {
			b.WriteString(formLabel.Render(label))
		}
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}

	b.WriteString(sep)
	b.WriteString("\n")

	// ── Footer ──
	if m.confirming {
		b.WriteString(StyleHighlight.Render("  Apply changes? "))
		b.WriteString(StyleHelp.Render("Y/n"))
	} else {
		b.WriteString(RenderFooterBar([]ShortcutEntry{
			{Key: "tab", Label: "Tab/↑↓ navigate"},
			{Key: "enter", Label: "enter submit"},
			{Key: "", Label: "esc cancel"},
		}, m.activeCmd))
	}
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return outerStyle.Render(StyleBorder.Render(innerPadding.Render(b.String())))
}

// RunEditForm launches an interactive form for editing book metadata.
// Returns the filled form data, or error if canceled.
func RunEditForm(defaults EditFormDefaults) (*EditFormData, error) {
	m := newEditForm(defaults)
	p := tea.NewProgram(m, tea.WithAltScreen())

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

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ShelfCreateFormData holds the shelf configuration collected from the user
type ShelfCreateFormData struct {
	ShelfName     string
	RepoName      string
	CreateRepo    bool
	Private       bool
	CreateRelease bool
}

type shelfCreateFormModel struct {
	inputs         []textinput.Model
	focused        int
	result         *ShelfCreateFormData
	err            error
	canceled       bool
	width          int
	height         int
	createRepoFlag bool // Checkbox state
	privateFlag    bool // Checkbox state
	releaseFlag    bool // Checkbox state
}

const (
	shelfFieldName = iota
	shelfFieldRepo
	shelfFieldCreateRepo
	shelfFieldPrivate
	shelfFieldRelease
)

func newShelfCreateForm() shelfCreateFormModel {
	m := shelfCreateFormModel{
		inputs:         make([]textinput.Model, 2), // Only name and repo are text inputs
		createRepoFlag: true,                       // Default: create repo
		privateFlag:    true,                       // Default: private
		releaseFlag:    true,                       // Default: create release
	}

	const inputWidth = 50

	// Shelf name field
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "e.g., programming, history, fiction"
	m.inputs[0].Focus()
	m.inputs[0].CharLimit = 50
	m.inputs[0].Width = inputWidth
	m.inputs[0].Prompt = ""

	// Repo name field
	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "e.g., shelf-programming (include 'shelf-' prefix)"
	m.inputs[1].CharLimit = 100
	m.inputs[1].Width = inputWidth
	m.inputs[1].Prompt = ""

	return m
}

func (m shelfCreateFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m shelfCreateFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			shelfName := strings.TrimSpace(m.inputs[0].Value())
			repoName := strings.TrimSpace(m.inputs[1].Value())

			if shelfName == "" {
				m.err = fmt.Errorf("shelf name is required")
				return m, nil
			}
			if repoName == "" {
				m.err = fmt.Errorf("repo name is required")
				return m, nil
			}

			m.result = &ShelfCreateFormData{
				ShelfName:     shelfName,
				RepoName:      repoName,
				CreateRepo:    m.createRepoFlag,
				Private:       m.privateFlag,
				CreateRelease: m.releaseFlag,
			}
			return m, tea.Quit

		case "tab", "shift+tab", "up", "down":
			// Navigate between fields
			if msg.String() == "up" || msg.String() == "shift+tab" {
				m.focused--
			} else {
				m.focused++
			}

			totalFields := 5 // 2 text inputs + 3 checkboxes
			if m.focused < 0 {
				m.focused = totalFields - 1
			} else if m.focused >= totalFields {
				m.focused = 0
			}

			// Focus/blur text inputs
			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focused {
					cmds[i] = m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}
			return m, tea.Batch(cmds...)

		case " ", "space":
			// Toggle checkboxes when focused
			switch m.focused {
			case shelfFieldCreateRepo:
				m.createRepoFlag = !m.createRepoFlag
				return m, nil
			case shelfFieldPrivate:
				m.privateFlag = !m.privateFlag
				return m, nil
			case shelfFieldRelease:
				m.releaseFlag = !m.releaseFlag
				return m, nil
			}
		}
	}

	// Update text inputs
	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *shelfCreateFormModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m shelfCreateFormModel) View() string {
	// Outer container for centering
	outerStyle := lipgloss.NewStyle().
		Padding(2, 4)

	var b strings.Builder

	// Title
	b.WriteString(StyleHeader.Render("Create New Shelf"))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(StyleNormal.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	// Text input fields
	textFields := []string{"Shelf Name", "Repository Name"}
	for i, label := range textFields {
		if i == m.focused {
			b.WriteString(StyleHighlight.Render("› " + label + ":"))
		} else {
			b.WriteString(StyleNormal.Render("  " + label + ":"))
		}
		b.WriteString("\n  ")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}

	// Checkbox fields
	checkboxes := []struct {
		label   string
		field   int
		checked bool
		help    string
	}{
		{"Create GitHub repository", shelfFieldCreateRepo, m.createRepoFlag, "Create repo via GitHub API"},
		{"Make repository private", shelfFieldPrivate, m.privateFlag, "Private repos are only visible to you"},
		{"Create 'library' release", shelfFieldRelease, m.releaseFlag, "Required for storing book files"},
	}

	for _, cb := range checkboxes {
		checkbox := "[ ]"
		if cb.checked {
			checkbox = "[✓]"
		}

		prefix := "  "
		if cb.field == m.focused {
			prefix = "› "
			b.WriteString(StyleHighlight.Render(fmt.Sprintf("%s%s %s", prefix, checkbox, cb.label)))
		} else {
			b.WriteString(StyleNormal.Render(fmt.Sprintf("%s%s %s", prefix, checkbox, cb.label)))
		}
		b.WriteString("\n")
		b.WriteString(StyleHelp.Render(fmt.Sprintf("    %s", cb.help)))
		b.WriteString("\n\n")
	}

	// Help text
	b.WriteString("\n")
	b.WriteString(StyleHelp.Render("Tab/↑↓: Navigate  Space: Toggle  Enter: Create  Esc: Cancel"))
	b.WriteString("\n")

	content := b.String()

	// Add inner padding inside border
	innerPadding := lipgloss.NewStyle().
		Padding(0, 2, 0, 1)

	// Apply inner padding, then border, then outer padding
	return outerStyle.Render(StyleBorder.Render(innerPadding.Render(content)))
}

// RunShelfCreateForm launches an interactive form for creating a new shelf
// Returns the filled form data, or error if canceled
func RunShelfCreateForm() (*ShelfCreateFormData, error) {
	m := newShelfCreateForm()
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("running form: %w", err)
	}

	fm, ok := finalModel.(shelfCreateFormModel)
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

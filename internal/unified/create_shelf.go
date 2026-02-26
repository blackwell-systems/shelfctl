package unified

import (
	"fmt"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/operations"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	createShelfFieldName = iota
	createShelfFieldRepo
	createShelfFieldCreateRepo
	createShelfFieldPrivate
)

// CreateShelfModel is the unified view for creating a new shelf
type CreateShelfModel struct {
	inputs         []textinput.Model
	focused        int
	createRepoFlag bool
	privateFlag    bool
	width          int
	height         int
	err            error
	processing     bool
	activeCmd      string
	statusMsg      string
	gh             *github.Client
	cfg            *config.Config
}

// NewCreateShelfModel creates a new create-shelf view
func NewCreateShelfModel(gh *github.Client, cfg *config.Config) CreateShelfModel {
	m := CreateShelfModel{
		inputs:         make([]textinput.Model, 2),
		createRepoFlag: true,
		privateFlag:    true,
		gh:             gh,
		cfg:            cfg,
	}

	const inputWidth = 50

	// Shelf name field
	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "e.g., programming, history, fiction"
	m.inputs[0].Focus()
	m.inputs[0].CharLimit = 50
	m.inputs[0].Width = inputWidth
	m.inputs[0].Prompt = ""

	// Repo name field (suffix after "shelf-" prefix)
	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "e.g., programming, history"
	m.inputs[1].CharLimit = 100
	m.inputs[1].Width = inputWidth
	m.inputs[1].Prompt = ""

	return m
}

// Init initializes the create-shelf view
func (m CreateShelfModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the create-shelf view
func (m CreateShelfModel) Update(msg tea.Msg) (CreateShelfModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tui.ClearActiveCmdMsg:
		m.activeCmd = ""
		return m, nil

	case tea.KeyMsg:
		if m.processing {
			// Ignore input while processing
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return m, func() tea.Msg { return QuitAppMsg{} }

		case "esc":
			// Return to hub (no terminal drop)
			return m, func() tea.Msg {
				return NavigateMsg{Target: "hub"}
			}

		case "enter":
			// Validate and submit
			shelfName := strings.TrimSpace(m.inputs[0].Value())
			repoSuffix := strings.TrimSpace(m.inputs[1].Value())

			if shelfName == "" {
				m.err = fmt.Errorf("shelf name is required")
				return m, nil
			}
			if repoSuffix == "" {
				m.err = fmt.Errorf("repo name is required")
				return m, nil
			}

			repoName := "shelf-" + repoSuffix

			// Start async creation
			m.activeCmd = "enter"
			highlightCmd := tui.HighlightCmd()
			m.processing = true
			m.statusMsg = "Creating shelf..."
			m.err = nil
			return m, tea.Batch(m.createShelfAsync(shelfName, repoName), highlightCmd)

		case "tab", "shift+tab", "up", "down":
			// Navigate between fields
			m.activeCmd = "tab"
			highlightCmd := tui.HighlightCmd()

			if msg.String() == "up" || msg.String() == "shift+tab" {
				m.focused--
			} else {
				m.focused++
			}

			totalFields := 4
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
			return m, tea.Batch(append(cmds, highlightCmd)...)

		case " ", "space":
			// Toggle checkboxes when focused
			m.activeCmd = " "
			highlightCmd := tui.HighlightCmd()
			switch m.focused {
			case createShelfFieldCreateRepo:
				m.createRepoFlag = !m.createRepoFlag
				return m, highlightCmd
			case createShelfFieldPrivate:
				m.privateFlag = !m.privateFlag
				return m, highlightCmd
			}
		}

	case CreateShelfCompleteMsg:
		// Shelf creation completed
		m.processing = false
		m.statusMsg = ""

		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}

		// Success - return to hub
		return m, func() tea.Msg {
			return NavigateMsg{Target: "hub"}
		}
	}

	// Update text inputs
	if !m.processing {
		cmd := m.updateInputs(msg)
		return m, cmd
	}

	return m, nil
}

func (m *CreateShelfModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

// View renders the create-shelf form
func (m CreateShelfModel) View() string {
	outerStyle := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder

	// Title
	b.WriteString(tui.StyleHeader.Render("Create New Shelf"))
	b.WriteString("\n\n")

	// Error message
	if m.err != nil {
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	// Processing indicator
	if m.processing {
		b.WriteString(tui.StyleHighlight.Render(fmt.Sprintf("⏳ %s", m.statusMsg)))
		b.WriteString("\n\n")
	}

	// Shelf Name field
	if m.focused == 0 {
		b.WriteString(tui.StyleHighlight.Render("› Shelf Name:"))
	} else {
		b.WriteString(tui.StyleNormal.Render("  Shelf Name:"))
	}
	b.WriteString("\n  ")
	b.WriteString(m.inputs[0].View())
	b.WriteString("\n\n")

	// Repository Name field (with shelf- prefix shown)
	if m.focused == 1 {
		b.WriteString(tui.StyleHighlight.Render("› Repository Name:"))
	} else {
		b.WriteString(tui.StyleNormal.Render("  Repository Name:"))
	}
	b.WriteString("\n  ")
	b.WriteString(tui.StyleHelp.Render("shelf-"))
	b.WriteString(m.inputs[1].View())
	b.WriteString("\n\n")

	// Checkbox fields
	checkboxes := []struct {
		label   string
		field   int
		checked bool
		help    string
	}{
		{"Create GitHub repository", createShelfFieldCreateRepo, m.createRepoFlag, "Create repo via GitHub API"},
		{"Make repository private", createShelfFieldPrivate, m.privateFlag, "Private repos are only visible to you"},
	}

	for _, cb := range checkboxes {
		checkbox := "[ ]"
		if cb.checked {
			checkbox = "[✓]"
		}

		prefix := "  "
		if cb.field == m.focused {
			prefix = "› "
			b.WriteString(tui.StyleHighlight.Render(fmt.Sprintf("%s%s %s", prefix, checkbox, cb.label)))
		} else {
			b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("%s%s %s", prefix, checkbox, cb.label)))
		}
		b.WriteString("\n")
		b.WriteString(tui.StyleHelp.Render(fmt.Sprintf("    %s", cb.help)))
		b.WriteString("\n\n")
	}

	// Help text
	b.WriteString("\n")
	if m.processing {
		b.WriteString(tui.StyleHelp.Render("Please wait..."))
	} else {
		b.WriteString(tui.RenderFooterBar([]tui.ShortcutEntry{
			{Key: "tab", Label: "Tab/↑↓ Navigate"},
			{Key: "space", Label: "Space Toggle"},
			{Key: "enter", Label: "Enter Create"},
			{Key: "q", Label: "Esc Cancel"},
		}, m.activeCmd))
	}
	b.WriteString("\n")

	content := b.String()

	// Add inner padding inside border
	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)

	// Apply inner padding, then border, then outer padding
	return outerStyle.Render(tui.StyleBorder.Render(innerPadding.Render(content)))
}

// createShelfAsync creates the shelf in the background
func (m CreateShelfModel) createShelfAsync(shelfName, repoName string) tea.Cmd {
	return func() tea.Msg {
		// Call shared creation logic from operations package
		err := operations.CreateShelf(m.gh, m.cfg, shelfName, repoName, m.createRepoFlag, m.privateFlag)

		return CreateShelfCompleteMsg{
			ShelfName: shelfName,
			RepoName:  repoName,
			Err:       err,
		}
	}
}

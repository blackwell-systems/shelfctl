package unified

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BrowseModel is a placeholder browse model for testing view switching
// TODO: Integrate full list_browser.go functionality
type BrowseModel struct {
	width  int
	height int
}

type browseKeys struct {
	quit key.Binding
}

var browseKeyMap = browseKeys{
	quit: key.NewBinding(
		key.WithKeys("q", "esc"),
		key.WithHelp("q/esc", "back to menu"),
	),
}

// NewBrowseModel creates a new browse model
func NewBrowseModel() BrowseModel {
	return BrowseModel{}
}

func (m BrowseModel) Init() tea.Cmd {
	return nil
}

func (m BrowseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, browseKeyMap.quit):
			// Navigate back to hub
			return m, func() tea.Msg {
				return NavigateMsg{Target: "hub"}
			}
		}
	}

	return m, nil
}

func (m BrowseModel) View() string {
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(2, 4)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")).
			Render("Browse Library"),
		"",
		"This is a placeholder browse view",
		"",
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("Press q or ESC to return to menu"),
		"",
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("(Full browse functionality coming next)"),
	)

	return style.Render(content)
}

package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ClearActiveCmdMsg clears the active command highlight in the footer.
type ClearActiveCmdMsg struct{}

// ShortcutEntry pairs a trigger key with the display label for footer highlighting.
type ShortcutEntry struct {
	Key   string // trigger key to match against activeCmd (empty = no highlight)
	Label string // display text
}

// SetActiveCmd sets an active command string and returns a 500ms tick to clear it.
func SetActiveCmd(activeCmd *string, key string) tea.Cmd {
	*activeCmd = key
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return ClearActiveCmdMsg{}
	})
}

// RenderFooterBar renders a footer bar with shortcut labels.
// The shortcut matching activeCmd is rendered with StyleHighlight; others are dim.
func RenderFooterBar(shortcuts []ShortcutEntry, activeCmd string) string {
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	parts := make([]string, len(shortcuts))
	for i, sc := range shortcuts {
		if activeCmd != "" && sc.Key == activeCmd {
			parts[i] = StyleHighlight.Render(sc.Label)
		} else {
			parts[i] = dimStyle.Render(sc.Label)
		}
	}

	return lipgloss.NewStyle().Padding(0, 1).Render(strings.Join(parts, dimStyle.Render(" • ")))
}

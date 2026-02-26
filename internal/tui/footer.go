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

// HighlightCmd returns a 500ms tick command to clear the active command highlight.
// Callers must set activeCmd on the model directly before returning:
//
//	m.activeCmd = "key"
//	return m, tui.HighlightCmd()
func HighlightCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return ClearActiveCmdMsg{}
	})
}

// SetActiveCmd sets an active command string and returns a 500ms tick to clear it.
//
// WARNING: Do NOT use in "return m, SetActiveCmd(...)". Go evaluates left-to-right,
// so m is copied before the pointer write takes effect. Use HighlightCmd instead:
//
//	m.activeCmd = "key"
//	return m, tui.HighlightCmd()
//
// Deprecated: Use HighlightCmd and direct field assignment instead.
func SetActiveCmd(activeCmd *string, key string) tea.Cmd {
	*activeCmd = key
	return HighlightCmd()
}

// RenderFooterBar renders a footer bar with shortcut labels.
// The shortcut matching activeCmd is rendered with StyleHighlight; others are dim.
func RenderFooterBar(shortcuts []ShortcutEntry, activeCmd string) string {
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	parts := make([]string, len(shortcuts))
	for i, sc := range shortcuts {
		if activeCmd != "" && sc.Key == activeCmd {
			parts[i] = StyleHighlight.Render("[ " + sc.Label + " ]")
		} else {
			parts[i] = dimStyle.Render(sc.Label)
		}
	}

	return lipgloss.NewStyle().Padding(0, 1).Render(strings.Join(parts, dimStyle.Render(" • ")))
}

// RenderWithFooter wraps a component view (list, multiselect) inside a border
// and appends a footer bar with shortcut highlights below the component's own help.
func RenderWithFooter(componentView string, shortcuts []ShortcutEntry, activeCmd string) string {
	footer := RenderFooterBar(shortcuts, activeCmd)
	content := componentView + "\n" + footer
	return StyleBorder.Render(content)
}

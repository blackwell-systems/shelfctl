package unified

import (
	"fmt"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/charmbracelet/lipgloss"
)

func (m ImportShelfModel) View() string {
	if m.empty {
		return m.renderMessage("No shelves configured", "Run 'shelfctl init' to create a shelf first.\nPress Enter to return to menu.")
	}

	if m.err != nil && m.phase == importShelfDone {
		return m.renderMessage(fmt.Sprintf("Error: %v", m.err), "Press Enter to return to menu")
	}

	switch m.phase {
	case importShelfSourceInput:
		return m.renderSourceInput()
	case importShelfShelfPicking:
		return tui.StyleBorder.Render(m.shelfList.View())
	case importShelfScanning:
		return m.renderMessage("Scanning source shelf...", "Loading catalogs and checking for duplicates")
	case importShelfBookPicking:
		return tui.StyleBorder.Render(m.ms.View())
	case importShelfProcessing:
		return m.renderMessage(m.statusMsg, "Please wait")
	case importShelfCommitting:
		return m.renderMessage(m.statusMsg, "Please wait")
	case importShelfDone:
		return m.renderDone()
	}

	return ""
}

func (m ImportShelfModel) renderMessage(title, help string) string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render(title))
	b.WriteString("\n\n")
	b.WriteString(tui.StyleHelp.Render(help))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

func (m ImportShelfModel) renderSourceInput() string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render("Import from Shelf"))
	b.WriteString("\n\n")

	b.WriteString(tui.StyleNormal.Render("Source shelf (owner/repo):"))
	b.WriteString("\n  ")
	b.WriteString(m.sourceInput.View())
	b.WriteString("\n\n")

	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	b.WriteString(tui.RenderFooterBar([]tui.ShortcutEntry{
		{Key: "enter", Label: "Enter Submit"},
		{Key: "", Label: "Esc Back"},
	}, ""))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

func (m ImportShelfModel) renderDone() string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render("Import Complete"))
	b.WriteString("\n\n")

	if m.successCount > 0 {
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  Imported: %d book(s)", m.successCount)))
		b.WriteString("\n")
	}
	if m.failCount > 0 {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render(fmt.Sprintf("  Failed:   %d book(s)", m.failCount)))
		b.WriteString("\n")
	}
	if m.dupeCount > 0 {
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		b.WriteString(dim.Render(fmt.Sprintf("  Skipped:  %d duplicate(s)", m.dupeCount)))
		b.WriteString("\n")
	}
	if m.successCount == 0 && m.failCount == 0 {
		b.WriteString(tui.StyleHelp.Render("  Nothing to import — all books already exist in destination"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(tui.StyleHelp.Render("Press Enter to return to menu"))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

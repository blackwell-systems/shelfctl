package unified

import (
	"fmt"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/charmbracelet/lipgloss"
)

func (m ImportRepoModel) View() string {
	if m.empty {
		return m.renderMessage("No shelves configured", "Run 'shelfctl init' to create a shelf first.\nPress Enter to return to menu.")
	}

	if m.err != nil && m.phase == importRepoDone {
		return m.renderMessage(fmt.Sprintf("Error: %v", m.err), "Press Enter to return to menu")
	}

	switch m.phase {
	case importRepoSourceInput:
		return m.renderSourceInput()
	case importRepoShelfPicking:
		return tui.StyleBorder.Render(m.shelfList.View())
	case importRepoScanning:
		return m.renderMessage("Scanning repository...", "Searching for book files (pdf, epub, mobi, djvu, azw3, cbz, cbr)")
	case importRepoFilePicking:
		return tui.StyleBorder.Render(m.ms.View())
	case importRepoProcessing:
		return m.renderMessage(m.statusMsg, "Please wait")
	case importRepoCommitting:
		return m.renderMessage(m.statusMsg, "Please wait")
	case importRepoDone:
		return m.renderDone()
	}

	return ""
}

func (m ImportRepoModel) renderMessage(title, help string) string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render(title))
	b.WriteString("\n\n")
	b.WriteString(tui.StyleHelp.Render(help))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

func (m ImportRepoModel) renderSourceInput() string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render("Import from Repo"))
	b.WriteString("\n\n")

	b.WriteString(tui.StyleNormal.Render("Source repository (owner/repo):"))
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

func (m ImportRepoModel) renderDone() string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render("Migration Complete"))
	b.WriteString("\n\n")

	if m.successCount > 0 {
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  Migrated: %d file(s)", m.successCount)))
		b.WriteString("\n")
	}
	if m.failCount > 0 {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render(fmt.Sprintf("  Failed:   %d file(s)", m.failCount)))
		b.WriteString("\n")
	}
	if m.successCount == 0 && m.failCount == 0 {
		b.WriteString(tui.StyleHelp.Render("  No files found to migrate"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(tui.StyleHelp.Render("Press Enter to return to menu"))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

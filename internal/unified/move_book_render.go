package unified

import (
	"fmt"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/charmbracelet/lipgloss"
)

func (m MoveBookModel) View() string {
	if m.empty {
		return m.renderMessage("No books in library", "Press Enter to return to menu")
	}

	if m.err != nil && m.phase == moveBookPicking {
		return m.renderMessage(fmt.Sprintf("Error: %v", m.err), "Press Enter to return to menu")
	}

	switch m.phase {
	case moveBookPicking:
		return tui.StyleBorder.Render(m.ms.View())

	case moveTypePicking:
		return m.renderTypePicker()

	case moveDestPicking:
		if m.moveType == moveToShelf {
			return tui.StyleBorder.Render(m.destShelfList.View())
		}
		return m.renderReleaseInput()

	case moveConfirming:
		return m.renderConfirmation()

	case moveProcessing:
		return m.renderMessage("Moving books...", "Please wait")
	}

	return ""
}

func (m MoveBookModel) renderMessage(title, help string) string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render(title))
	b.WriteString("\n\n")
	b.WriteString(tui.StyleHelp.Render(help))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

func (m MoveBookModel) renderTypePicker() string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render("Move To"))
	b.WriteString("\n\n")

	// Show selected books summary
	if len(m.toMove) == 1 {
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("Book: %s (%s)", m.toMove[0].Book.ID, m.toMove[0].Book.Title)))
	} else {
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("%d books selected", len(m.toMove))))
	}
	b.WriteString("\n\n")

	// Error display
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	// Options
	options := []string{
		"Different shelf (different repository)",
		"Different release (same shelf)",
	}

	for i, opt := range options {
		if i == m.typeSelected {
			b.WriteString(tui.StyleHighlight.Render(fmt.Sprintf("› %d. %s", i+1, opt)))
		} else {
			b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  %d. %s", i+1, opt)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(tui.RenderFooterBar([]tui.ShortcutEntry{
		{Key: "1", Label: "↑↓/1-2 Select"},
		{Key: "enter", Label: "Enter Confirm"},
		{Key: "", Label: "Esc Back"},
	}, m.activeCmd))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

func (m MoveBookModel) renderReleaseInput() string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render("Destination Release"))
	b.WriteString("\n\n")

	// Show current release
	b.WriteString(tui.StyleHelp.Render(fmt.Sprintf("Current release: %s", m.toMove[0].Book.Source.Release)))
	b.WriteString("\n\n")

	// Error display
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	b.WriteString(tui.StyleNormal.Render("Release tag:"))
	b.WriteString("\n  ")
	b.WriteString(m.releaseInput.View())
	b.WriteString("\n\n")

	b.WriteString(tui.RenderFooterBar([]tui.ShortcutEntry{
		{Key: "enter", Label: "Enter Confirm"},
		{Key: "", Label: "Esc Back"},
	}, m.activeCmd))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

func (m MoveBookModel) renderConfirmation() string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render("Confirm Move"))
	b.WriteString("\n\n")

	// Show books
	if len(m.toMove) == 1 {
		item := m.toMove[0]
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  Book:  %s (%s)", item.Book.ID, item.Book.Title)))
		b.WriteString("\n")
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  From:  %s/%s@%s", item.Owner, item.Repo, item.Book.Source.Release)))
		b.WriteString("\n")
	} else {
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  Books: %d selected", len(m.toMove))))
		b.WriteString("\n")
		for _, item := range m.toMove {
			b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("    - %s (%s) [%s]", item.Book.ID, item.Book.Title, item.ShelfName)))
			b.WriteString("\n")
		}
	}

	// Show destination
	if m.moveType == moveToShelf {
		dstShelf := m.cfg.ShelfByName(m.destShelfName)
		if dstShelf != nil {
			owner := dstShelf.EffectiveOwner(m.cfg.GitHub.Owner)
			release := dstShelf.EffectiveRelease(m.cfg.Defaults.Release)
			b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  To:    %s/%s@%s", owner, dstShelf.Repo, release)))
		}
	} else {
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  To:    %s/%s@%s", m.toMove[0].Owner, m.toMove[0].Repo, m.destRelease)))
	}
	b.WriteString("\n\n")

	b.WriteString(tui.RenderFooterBar([]tui.ShortcutEntry{
		{Key: "enter", Label: "Enter/y Confirm"},
		{Key: "", Label: "Esc/n Back"},
	}, m.activeCmd))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

package unified

import (
	"fmt"
	"strings"

	"github.com/blackwell-systems/bubbletea-carousel"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
)

// bulkEditOps are the operations available in the bulk-edit overlay.
var bulkEditOps = []string{
	"Add tag",
	"Remove tag",
	"Set author",
	"Set year",
}

// carouselBookItem wraps a BookItem together with its current saved state so
// that the delegate can access both without needing the item's index.
type carouselBookItem struct {
	book  tui.BookItem
	saved bool
}

// bookCarouselDelegate implements carousel.ItemDelegate for BookItems.
type bookCarouselDelegate struct{}

// Render returns the card interior content for a carouselBookItem.
// innerW is the number of visible columns available for text.
func (d bookCarouselDelegate) Render(item any, innerW int) string {
	b := item.(carouselBookItem)

	titleText := xansi.Truncate(b.book.Book.Title, innerW, "…")
	authorText := xansi.Truncate(b.book.Book.Author, innerW, "…")
	tagsText := xansi.Truncate(strings.Join(b.book.Book.Tags, ", "), innerW, "…")

	var statusText string
	if b.saved {
		statusText = lipgloss.NewStyle().Foreground(lipgloss.Color("28")).Render("✓ saved")
	} else {
		statusText = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("· unsaved")
	}

	return titleText + "\n" + authorText + "\n" + tagsText + "\n" + statusText
}

// IsMarked reports whether the item has been saved (confirmed by the user).
func (d bookCarouselDelegate) IsMarked(item any) bool {
	return item.(carouselBookItem).saved
}

// rebuildCarouselItems synchronises the carousel model's item slice with the
// current toEdit list and formStates. Call this whenever the saved state of
// any book may have changed or when first entering the carousel.
func (m *EditBookModel) rebuildCarouselItems() {
	items := make([]any, len(m.toEdit))
	for i, b := range m.toEdit {
		items[i] = carouselBookItem{
			book:  b,
			saved: m.formStates[i].saved,
		}
	}
	m.carouselModel.SetItems(items)
}

// updateCarouselFromMsg handles key events while the carousel is shown.
// It intercepts ctrl+c, esc, and the bulk-edit trigger key "a" before
// forwarding remaining keys to the carousel component.
func (m EditBookModel) updateCarouselFromMsg(msg tea.KeyMsg) (EditBookModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }

	case "q", "esc":
		m.inCarousel = false
		m.initFormForBook(m.editIndex)
		return m, textinput.Blink

	case "a":
		m.inBulkEdit = true
		m.bulkFieldIdx = 0
		m.bulkFieldSelected = false
		return m, nil
	}

	var cmd tea.Cmd
	m.carouselModel, cmd = m.carouselModel.Update(msg)
	return m, cmd
}

// updateBulkEdit handles key events in the bulk-edit overlay.
func (m EditBookModel) updateBulkEdit(msg tea.KeyMsg) (EditBookModel, tea.Cmd) {
	if m.bulkFieldSelected {
		// Value-entry mode: forward most keys to the text input
		switch msg.String() {
		case "ctrl+c":
			return m, func() tea.Msg { return QuitAppMsg{} }
		case "esc":
			m.bulkFieldSelected = false
			return m, nil
		case "enter":
			m.applyBulkEdit(m.bulkFieldIdx, m.bulkInput.Value())
			m.inBulkEdit = false
			m.bulkFieldSelected = false
			return m, nil
		}
		var cmd tea.Cmd
		m.bulkInput, cmd = m.bulkInput.Update(msg)
		return m, cmd
	}

	// Operation-selection mode
	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }
	case "q", "esc":
		m.inBulkEdit = false
		return m, nil
	case "up", "k":
		if m.bulkFieldIdx > 0 {
			m.bulkFieldIdx--
		}
	case "down", "j":
		if m.bulkFieldIdx < len(bulkEditOps)-1 {
			m.bulkFieldIdx++
		}
	case "enter":
		inp := textinput.New()
		inp.Width = 30
		inp.CharLimit = 200
		switch m.bulkFieldIdx {
		case 0:
			inp.Placeholder = "tag name to add"
		case 1:
			inp.Placeholder = "tag name to remove"
		case 2:
			inp.Placeholder = "author name"
		case 3:
			inp.Placeholder = "year (e.g., 2023)"
			inp.CharLimit = 4
		}
		m.bulkInput = inp
		m.bulkFieldSelected = true
		return m, textinput.Blink
	}
	return m, nil
}

// renderBulkEditOverlay renders the modal panel for bulk-edit operations.
func (m EditBookModel) renderBulkEditOverlay() string {
	orange := tui.ColorOrange
	dimColor := lipgloss.Color("240")

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render(fmt.Sprintf("Apply to all %d books", len(m.toEdit))))
	b.WriteString("\n\n")

	for i, op := range bulkEditOps {
		if i == m.bulkFieldIdx {
			b.WriteString(lipgloss.NewStyle().Foreground(orange).Bold(true).Render("▶ " + op))
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("  " + op))
		}
		b.WriteString("\n")
	}

	if m.bulkFieldSelected {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(orange).Render(bulkEditOps[m.bulkFieldIdx]))
		b.WriteString(":\n")
		b.WriteString(m.bulkInput.View())
		b.WriteString("\n\n")
		b.WriteString(tui.StyleHelp.Render("Enter to apply  •  Esc back"))
	} else {
		b.WriteString("\n")
		b.WriteString(tui.StyleHelp.Render("↑↓ Select  •  Enter choose  •  Esc cancel"))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(orange).
		Padding(1, 2).
		Render(b.String())
}

// newCarouselModel creates and initialises a carousel.Model for the current
// book selection. Called once when multi-book edit mode is entered.
func newCarouselModel() carousel.Model {
	return carousel.New(carousel.Config{
		Delegate:     bookCarouselDelegate{},
		Title:        "Select a book to edit",
		ActiveColor:  tui.ColorOrange,
		MarkedColor:  lipgloss.Color("28"),
		DefaultColor: lipgloss.Color("240"),
		ExtraFooter:  "a Bulk edit",
	})
}

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m BrowserModel) renderDetailsPane() string {
	selectedItem := m.list.SelectedItem()
	if selectedItem == nil {
		return ""
	}

	bookItem, ok := selectedItem.(BookItem)
	if !ok {
		return ""
	}

	// Calculate details pane width (40% of screen, accounting for divider and master border)
	detailsWidth := ((m.width - 2) * 4) / 10
	if detailsWidth < 30 {
		detailsWidth = 30 // Minimum width for readability
	}

	// Calculate max text width: panel width minus padding (2 chars)
	// Account for label widths (e.g., "Repository: " is 12 chars)
	const labelWidth = 12 // longest label
	maxTextWidth := detailsWidth - 2 - labelWidth
	if maxTextWidth < 10 {
		maxTextWidth = 10
	}

	// Style for the details content area
	detailsStyle := lipgloss.NewStyle().
		Width(detailsWidth).
		Padding(0, 1)

	var s strings.Builder

	// Show cover image if available and terminal supports it
	if bookItem.HasCover {
		protocol := DetectImageProtocol()
		if protocol != ProtocolNone {
			if img := RenderInlineImage(bookItem.CoverPath, protocol); img != "" {
				s.WriteString(img)
				s.WriteString("\n\n")
			}
		}
	}

	// Title
	s.WriteString(StyleHeader.Render("Book Details"))
	s.WriteString("\n\n")

	// Title
	s.WriteString(StyleHighlight.Render("Title: "))
	s.WriteString(truncateText(bookItem.Book.Title, maxTextWidth))
	s.WriteString("\n\n")

	// Author
	if bookItem.Book.Author != "" {
		s.WriteString(StyleHighlight.Render("Author: "))
		s.WriteString(truncateText(bookItem.Book.Author, maxTextWidth))
		s.WriteString("\n\n")
	}

	// Year
	if bookItem.Book.Year > 0 {
		s.WriteString(StyleHighlight.Render("Year: "))
		fmt.Fprintf(&s, "%d", bookItem.Book.Year)
		s.WriteString("\n\n")
	}

	// Tags
	if len(bookItem.Book.Tags) > 0 {
		s.WriteString(StyleHighlight.Render("Tags: "))
		s.WriteString("\n")
		for _, t := range bookItem.Book.Tags {
			pill := lipgloss.NewStyle().
				Background(ColorTealDim).Foreground(ColorTealLight).
				Padding(0, 1).Render(t)
			s.WriteString(pill + " ")
		}
		s.WriteString("\n\n")
	}

	// Shelf
	s.WriteString(StyleHighlight.Render("Shelf: "))
	s.WriteString(truncateText(bookItem.ShelfName, maxTextWidth))
	s.WriteString("\n\n")

	// Repository
	s.WriteString(StyleHighlight.Render("Repository: "))
	repoText := fmt.Sprintf("%s/%s", bookItem.Owner, bookItem.Repo)
	s.WriteString(truncateText(repoText, maxTextWidth))
	s.WriteString("\n\n")

	// Cache status
	s.WriteString(StyleHighlight.Render("Cached: "))
	if bookItem.Cached {
		s.WriteString(StyleCached.Render("✓ Yes"))
	} else {
		s.WriteString("No")
	}
	s.WriteString("\n\n")

	// Size
	if bookItem.Book.SizeBytes > 0 {
		s.WriteString(StyleHighlight.Render("Size: "))
		s.WriteString(formatBytes(bookItem.Book.SizeBytes))
		s.WriteString("\n\n")
	}

	// Format
	s.WriteString(StyleHighlight.Render("Format: "))
	s.WriteString(truncateText(bookItem.Book.Format, maxTextWidth))
	s.WriteString("\n")

	// Apply details panel styling
	return detailsStyle.Render(s.String())
}

// renderFooter creates a footer with all available keyboard shortcuts.
// The shortcut matching activeCmd is rendered with StyleHighlight.
func (m BrowserModel) renderFooter() string {
	return RenderFooterBar([]ShortcutEntry{
		{Key: "", Label: "↑/↓ navigate"},
		{Key: "/", Label: "/ filter"},
		{Key: "", Label: "enter action"},
		{Key: "o", Label: "o open"},
		{Key: "g", Label: "g download"},
		{Key: "x", Label: "x uncache"},
		{Key: "s", Label: "s sync"},
		{Key: "e", Label: "e edit"},
		{Key: " ", Label: "space select"},
		{Key: "c", Label: "c clear"},
		{Key: "tab", Label: "tab detail toggle"},
		{Key: "", Label: "q quit"},
	}, m.activeCmd)
}

func (m BrowserModel) View() string {
	if m.quitting {
		return ""
	}

	// Outer container for centering - adds margin around the entire box
	outerStyle := lipgloss.NewStyle().
		Padding(2, 4) // top/bottom: 2 lines, left/right: 4 chars

	// Inner content box with border
	masterStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorTeal).
		Padding(0)

	// Calculate dimensions for inner box
	// Subtract outer padding (2*2 vertical, 4*2 horizontal) and border (2 each side)
	if m.width > 0 && m.height > 0 {
		innerWidth := m.width - (4 * 2) - 2   // outer padding + border
		innerHeight := m.height - (2 * 2) - 2 // outer padding + border

		// Ensure minimum size
		if innerWidth < 60 {
			innerWidth = 60
		}
		if innerHeight < 10 {
			innerHeight = 10
		}

		masterStyle = masterStyle.
			Width(innerWidth).
			Height(innerHeight)
	}

	var mainContent string
	if m.showDetails {
		// Split-panel layout: compose panels then wrap
		// Add border on right side of list to create solid divider
		listStyle := lipgloss.NewStyle().
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorTeal)
		listView := listStyle.Render(m.list.View())
		detailsView := m.renderDetailsPane()

		// Join horizontally: list (with border) + details
		mainContent = lipgloss.JoinHorizontal(
			lipgloss.Top,
			listView,
			detailsView,
		)
	} else {
		// Single panel: list only
		mainContent = m.list.View()
	}

	// Create footer with divider
	// Calculate divider width based on content width
	dividerWidth := m.width - (4 * 2) - 2 // outer padding + border
	if dividerWidth < 40 {
		dividerWidth = 40
	}
	divider := lipgloss.NewStyle().
		Foreground(ColorTeal).
		Width(dividerWidth).
		Render(strings.Repeat("─", dividerWidth))
	footer := m.renderFooter()

	// Compose: main content + divider + footer
	content := lipgloss.JoinVertical(lipgloss.Left, mainContent, divider, footer)

	// Apply inner box border
	boxed := masterStyle.Render(content)

	// Add download progress if active
	if m.downloading && m.currentDownload != nil {
		progressBar := m.progress.ViewAs(m.downloadPct)
		label := fmt.Sprintf("Downloading %s", m.currentDownload.Book.ID)
		if len(m.downloadQueue) > 0 {
			label = fmt.Sprintf("[%d remaining] %s", len(m.downloadQueue)+1, label)
		}

		progressView := lipgloss.NewStyle().
			Foreground(ColorYellow).
			Render(label + "\n" + progressBar)

		boxed = lipgloss.JoinVertical(lipgloss.Left, boxed, "", progressView)
	} else if m.downloadErr != "" {
		errorView := lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Render(m.downloadErr)
		boxed = lipgloss.JoinVertical(lipgloss.Left, boxed, "", errorView)
	}

	// Apply outer container for floating effect
	return outerStyle.Render(boxed)
}

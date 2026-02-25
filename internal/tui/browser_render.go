package tui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// Cached image protocol detection (computed once)
var (
	detectedProtocol   TerminalImageProtocol
	detectProtocolOnce sync.Once
)

func cachedImageProtocol() TerminalImageProtocol {
	detectProtocolOnce.Do(func() {
		detectedProtocol = DetectImageProtocol()
	})
	return detectedProtocol
}

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
		protocol := cachedImageProtocol()
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
			s.WriteString(StyleTagPill.Render(t))
			s.WriteString(" ")
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

// Pre-allocated styles for View() hot path (avoid per-frame allocations)
var (
	viewOuterStyle = lipgloss.NewStyle().Padding(2, 4)
	viewMasterBase = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorTeal).
			Padding(0)
	viewListBorderStyle = lipgloss.NewStyle().
				BorderRight(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(ColorTeal)
)

func (m BrowserModel) View() string {
	if m.quitting {
		return ""
	}

	// Inner content box with border — copy base and set dimensions
	masterStyle := viewMasterBase
	if m.width > 0 && m.height > 0 {
		innerWidth := m.width - (4 * 2) - 2
		innerHeight := m.height - (2 * 2) - 2
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
		listView := viewListBorderStyle.Render(m.list.View())
		detailsView := m.renderDetailsPane()
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, listView, detailsView)
	} else {
		mainContent = m.list.View()
	}

	// Footer with cached divider (rebuilt on WindowSizeMsg)
	footer := m.renderFooter()
	content := lipgloss.JoinVertical(lipgloss.Left, mainContent, m.cachedDivider, footer)
	boxed := masterStyle.Render(content)

	// Add download progress if active
	if m.downloading && m.currentDownload != nil {
		progressBar := m.progress.ViewAs(m.downloadPct)
		label := fmt.Sprintf("Downloading %s", m.currentDownload.Book.ID)
		if len(m.downloadQueue) > 0 {
			label = fmt.Sprintf("[%d remaining] %s", len(m.downloadQueue)+1, label)
		}
		boxed = lipgloss.JoinVertical(lipgloss.Left, boxed, "", StyleProgress.Render(label+"\n"+progressBar))
	} else if m.downloadErr != "" {
		boxed = lipgloss.JoinVertical(lipgloss.Left, boxed, "", StyleError.Render(m.downloadErr))
	}

	return viewOuterStyle.Render(boxed)
}

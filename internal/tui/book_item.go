package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// BookItem represents a book in the list with metadata.
type BookItem struct {
	Book        catalog.Book
	ShelfName   string
	Cached      bool
	HasCover    bool
	CoverPath   string
	Owner       string
	Repo        string
	Release     string // Release tag for this book
	CatalogPath string // Path to catalog.yml in repo
	selected    bool   // For multi-select mode
}

// FilterValue returns a string used for filtering in the list
func (b BookItem) FilterValue() string {
	// Include ID, title, tags, and shelf name in filter
	tags := strings.Join(b.Book.Tags, " ")
	return fmt.Sprintf("%s %s %s %s", b.Book.ID, b.Book.Title, tags, b.ShelfName)
}

// truncateText truncates a string to maxWidth visible chars with ellipsis.
func truncateText(s string, maxWidth int) string {
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return "…"
	}
	return string(runes[:maxWidth-1]) + "…"
}

// formatBytes formats bytes as human-readable size
func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n := n / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// IsSelected implements multiselect.SelectableItem
func (b BookItem) IsSelected() bool {
	return b.selected
}

// SetSelected implements multiselect.SelectableItem
func (b *BookItem) SetSelected(selected bool) {
	b.selected = selected
}

// IsSelectable implements multiselect.SelectableItem
// All books are selectable
func (b BookItem) IsSelectable() bool {
	return true
}

// browserTitle derives a title from the books' shelf names.
// If all books belong to one shelf, returns "Shelf: <name>".
// If books span multiple shelves, returns "All Shelves".
func browserTitle(books []BookItem) string {
	shelves := make(map[string]struct{})
	for _, b := range books {
		shelves[b.ShelfName] = struct{}{}
	}
	if len(shelves) == 1 {
		for name := range shelves {
			return "Shelf: " + name
		}
	}
	return "All Shelves"
}

// Column width constraints
const (
	minTitleWidth  = 12
	maxTitleWidth  = 48
	minAuthorWidth = 8
	maxAuthorWidth = 26
	minTagWidth    = 6
	minShelfWidth  = 5
	maxShelfWidth  = 16
	minCachedWidth = 7
	maxCachedWidth = 10
	columnGap      = 1
)

// computeColumnWidths distributes available width proportionally across columns.
func computeColumnWidths(totalWidth int) (titleW, authorW, tagW, shelfW, cachedW int) {
	// Reserve space for prefix ("› " or "  " or "[✓] ") and gaps between columns
	prefix := 2
	gaps := columnGap * 4 // 4 gaps between 5 columns
	usable := totalWidth - prefix - gaps
	if usable < minTitleWidth+minAuthorWidth+minTagWidth+minShelfWidth+minCachedWidth {
		return minTitleWidth, minAuthorWidth, minTagWidth, minShelfWidth, minCachedWidth
	}
	titleW = usable * 45 / 100
	if titleW > maxTitleWidth {
		titleW = maxTitleWidth
	}
	remaining := usable - titleW
	authorW = remaining * 35 / 100
	if authorW > maxAuthorWidth {
		authorW = maxAuthorWidth
	}
	shelfW = remaining * 20 / 100
	if shelfW > maxShelfWidth {
		shelfW = maxShelfWidth
	}
	tagW = remaining * 25 / 100
	cachedW = remaining - authorW - tagW - shelfW // remainder
	if cachedW > maxCachedWidth {
		cachedW = maxCachedWidth
	}

	// Enforce minimums
	if titleW < minTitleWidth {
		titleW = minTitleWidth
	}
	if authorW < minAuthorWidth {
		authorW = minAuthorWidth
	}
	if tagW < minTagWidth {
		tagW = minTagWidth
	}
	if shelfW < minShelfWidth {
		shelfW = minShelfWidth
	}
	if cachedW < minCachedWidth {
		cachedW = minCachedWidth
	}
	return
}

// padOrTruncate pads s to exactly width visible chars, truncating with "…" if necessary.
// Uses rune count (not byte length) so multi-byte UTF-8 characters align correctly.
func padOrTruncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	n := len(runes)
	if n > width {
		if width <= 1 {
			return "…"
		}
		return string(runes[:width-1]) + "…"
	}
	if n < width {
		return s + strings.Repeat(" ", width-n)
	}
	return s
}

// renderBookItem renders a book in the browser list with fixed-width columns.
func renderBookItem(w io.Writer, m list.Model, index int, item list.Item) {
	bookItem, ok := item.(BookItem)
	if !ok {
		return
	}

	listWidth := m.Width()
	if listWidth <= 0 {
		listWidth = 80
	}
	titleW, authorW, tagW, shelfW, cachedW := computeColumnWidths(listWidth)

	gap := strings.Repeat(" ", columnGap)

	// Cursor / selection prefix
	isCursor := index == m.Index()
	prefix := "  "
	if isCursor {
		prefix = lipgloss.NewStyle().Foreground(ColorOrange).Render("›") + " "
	}
	if bookItem.selected {
		prefix = lipgloss.NewStyle().Foreground(ColorTealLight).Bold(true).Render("✓") + "  "
		titleW--
		if titleW < minTitleWidth {
			titleW = minTitleWidth
		}
	}

	// Build column content
	titleCol := padOrTruncate(bookItem.Book.Title, titleW)
	authorCol := padOrTruncate(bookItem.Book.Author, authorW)

	tagStr := strings.Join(bookItem.Book.Tags, " · ")
	tagCol := padOrTruncate(tagStr, tagW)

	shelfCol := padOrTruncate(bookItem.ShelfName, shelfW)

	cachedStr := ""
	if bookItem.Cached {
		cachedStr = "✓ local"
	}
	cachedCol := padOrTruncate(cachedStr, cachedW)

	isCursorSelected := isCursor

	// Style each column
	var titleStyled, authorStyled, tagStyled, shelfStyled, cachedStyled string
	if isCursorSelected {
		titleStyled = StyleHighlight.Render(titleCol)
		authorStyled = lipgloss.NewStyle().Foreground(ColorOrange).Faint(true).Render(authorCol)
		tagStyled = lipgloss.NewStyle().Foreground(ColorTealLight).Render(tagCol)
		shelfStyled = lipgloss.NewStyle().Foreground(ColorOrange).Faint(true).Render(shelfCol)
		cachedStyled = StyleHighlight.Render(cachedCol)
	} else {
		titleStyled = StyleNormal.Render(titleCol)
		authorStyled = StyleHelp.Render(authorCol)
		tagStyled = StyleTag.Render(tagCol)
		shelfStyled = StyleHelp.Render(shelfCol)
		if bookItem.Cached {
			cachedStyled = StyleCached.Render(cachedCol)
		} else {
			cachedStyled = cachedCol
		}
	}

	line := prefix + titleStyled + gap + authorStyled + gap + tagStyled + gap + shelfStyled + gap + cachedStyled
	_, _ = fmt.Fprint(w, line)
}

package unified

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CacheInfoModel displays cache statistics in the TUI
type CacheInfoModel struct {
	width  int
	height int

	totalBooks    int
	cachedCount   int
	modifiedCount int
	totalSize     int64
	cacheDir      string

	uncachedBooks []cacheInfoBookEntry
	modifiedBooks []cacheInfoBookEntry
	shelfStats    []cacheInfoShelfStat
}

type cacheInfoBookEntry struct {
	id    string
	title string
	shelf string
}

type cacheInfoShelfStat struct {
	name        string
	repo        string
	totalBooks  int
	cachedBooks int
	cacheSize   int64
}

// NewCacheInfoModel creates a new cache-info view
func NewCacheInfoModel(books []tui.BookItem, cacheMgr *cache.Manager) CacheInfoModel {
	m := CacheInfoModel{}

	// Compute per-shelf stats
	shelfMap := map[string]*cacheInfoShelfStat{}
	var shelfOrder []string

	for _, item := range books {
		key := item.ShelfName
		stat, ok := shelfMap[key]
		if !ok {
			stat = &cacheInfoShelfStat{
				name: item.ShelfName,
				repo: item.Repo,
			}
			shelfMap[key] = stat
			shelfOrder = append(shelfOrder, key)
		}
		stat.totalBooks++
		m.totalBooks++

		if item.Cached {
			m.cachedCount++
			stat.cachedBooks++
			path := cacheMgr.Path(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset)
			if info, err := os.Stat(path); err == nil {
				size := info.Size()
				m.totalSize += size
				stat.cacheSize += size
			}

			if cacheMgr.HasBeenModified(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset, item.Book.Checksum.SHA256) {
				m.modifiedCount++
				m.modifiedBooks = append(m.modifiedBooks, cacheInfoBookEntry{
					id: item.Book.ID, title: item.Book.Title, shelf: item.ShelfName,
				})
			}
		} else {
			m.uncachedBooks = append(m.uncachedBooks, cacheInfoBookEntry{
				id: item.Book.ID, title: item.Book.Title, shelf: item.ShelfName,
			})
		}
	}

	for _, key := range shelfOrder {
		m.shelfStats = append(m.shelfStats, *shelfMap[key])
	}

	m.cacheDir = cacheMgr.Path("", "", "", "")
	if m.cacheDir != "" {
		m.cacheDir = filepath.Dir(m.cacheDir)
	}

	return m
}

func (m CacheInfoModel) Init() tea.Cmd { return nil }

func (m CacheInfoModel) Update(msg tea.Msg) (CacheInfoModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc", "q":
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		case "ctrl+c":
			return m, func() tea.Msg { return QuitAppMsg{} }
		}
	}
	return m, nil
}

func (m CacheInfoModel) View() string {
	style := lipgloss.NewStyle().Padding(2, 4)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	warn := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	var b strings.Builder

	b.WriteString(tui.StyleHeader.Render("Cache Statistics"))
	b.WriteString("\n\n")

	// Summary stats
	b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  Total books:  %d", m.totalBooks)))
	b.WriteString("\n")
	b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  Cached:       %d", m.cachedCount)))
	b.WriteString("\n")
	if m.modifiedCount > 0 {
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  Modified:     %d (annotations/highlights)", m.modifiedCount)))
		b.WriteString("\n")
	}
	b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  Cache size:   %s", humanBytes(m.totalSize))))
	b.WriteString("\n")
	b.WriteString(dim.Render(fmt.Sprintf("  Cache dir:    %s", m.cacheDir)))
	b.WriteString("\n")

	// Per-shelf breakdown
	if len(m.shelfStats) > 1 {
		b.WriteString("\n")
		b.WriteString(tui.StyleHeader.Render("Per Shelf"))
		b.WriteString("\n\n")
		for _, s := range m.shelfStats {
			cached := fmt.Sprintf("%d/%d cached", s.cachedBooks, s.totalBooks)
			if s.cacheSize > 0 {
				cached += fmt.Sprintf(" (%s)", humanBytes(s.cacheSize))
			}
			b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  %s: %s", s.name, cached)))
			b.WriteString("\n")
		}
	}

	// Uncached books
	uncachedCount := m.totalBooks - m.cachedCount
	if uncachedCount > 0 {
		b.WriteString("\n")
		b.WriteString(warn.Render(fmt.Sprintf("  %d book(s) not cached:", uncachedCount)))
		b.WriteString("\n")
		limit := len(m.uncachedBooks)
		if limit > 10 {
			limit = 10
		}
		for _, entry := range m.uncachedBooks[:limit] {
			b.WriteString(dim.Render(fmt.Sprintf("    - %s (%s)", entry.id, entry.title)))
			b.WriteString("\n")
		}
		if len(m.uncachedBooks) > 10 {
			b.WriteString(dim.Render(fmt.Sprintf("    ... and %d more", len(m.uncachedBooks)-10)))
			b.WriteString("\n")
		}
	}

	// Modified books
	if m.modifiedCount > 0 {
		b.WriteString("\n")
		info := lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
		b.WriteString(info.Render(fmt.Sprintf("  %d book(s) with local changes:", m.modifiedCount)))
		b.WriteString("\n")
		for _, entry := range m.modifiedBooks {
			b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("    - %s (%s)", entry.id, entry.title)))
			b.WriteString("\n")
		}
		b.WriteString(dim.Render("    Run 'shelfctl sync --all' to upload changes"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(tui.StyleHelp.Render("Press Enter to return to menu"))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

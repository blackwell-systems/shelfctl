package unified

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// IndexModel shows all books across shelves with search and HTML export.
type IndexModel struct {
	width  int
	height int

	books    []cache.IndexBook // all books, unfiltered
	filtered []cache.IndexBook // books matching current search

	cursor    int
	offset    int
	searching bool
	query     string

	cfg      *config.Config
	cacheMgr *cache.Manager
	gh       *github.Client

	statusMsg string // shown in bottom bar after generate
}

// NewIndexModel creates a new index view from the pre-collected books.
func NewIndexModel(books []cache.IndexBook, gh *github.Client, cfg *config.Config, cacheMgr *cache.Manager) IndexModel {
	m := IndexModel{
		books:    books,
		filtered: books,
		cfg:      cfg,
		cacheMgr: cacheMgr,
		gh:       gh,
	}
	return m
}

func (m IndexModel) Init() tea.Cmd { return nil }

func (m IndexModel) Update(msg tea.Msg) (IndexModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.searching {
			return m.updateSearchMode(msg)
		}
		return m.updateNormalMode(msg)
	}
	return m, nil
}

func (m IndexModel) updateNormalMode(msg tea.KeyMsg) (IndexModel, tea.Cmd) {
	listHeight := m.listHeight()

	switch msg.String() {
	case "esc", "q":
		return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			if m.cursor >= m.offset+listHeight {
				m.offset = m.cursor - listHeight + 1
			}
		}

	case "/":
		m.searching = true
		m.statusMsg = ""

	case "g":
		path, err := m.generateHTML()
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", err)
		} else {
			m.statusMsg = fmt.Sprintf("Generated: file://%s", path)
		}

	case "o":
		path, err := m.generateHTML()
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", err)
		} else {
			m.openInBrowser(path)
			m.statusMsg = fmt.Sprintf("Opened: file://%s", path)
		}
	}
	return m, nil
}

func (m IndexModel) updateSearchMode(msg tea.KeyMsg) (IndexModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		m.query = ""
		m.filtered = m.books
		m.cursor = 0
		m.offset = 0
	case "enter":
		m.searching = false
	case "backspace":
		if len(m.query) > 0 {
			m.query = m.query[:len(m.query)-1]
			m.applyFilter()
		}
	default:
		if len(msg.String()) == 1 {
			m.query += msg.String()
			m.applyFilter()
		}
	}
	return m, nil
}

func (m *IndexModel) applyFilter() {
	if m.query == "" {
		m.filtered = m.books
	} else {
		q := strings.ToLower(m.query)
		m.filtered = m.filtered[:0]
		for _, b := range m.books {
			if strings.Contains(strings.ToLower(b.Book.Title), q) ||
				strings.Contains(strings.ToLower(b.Book.Author), q) ||
				strings.Contains(strings.ToLower(b.ShelfName), q) {
				m.filtered = append(m.filtered, b)
			}
		}
	}
	m.cursor = 0
	m.offset = 0
}

func (m IndexModel) generateHTML() (string, error) {
	if err := m.cacheMgr.GenerateHTMLIndex(m.books); err != nil {
		return "", err
	}
	return filepath.Join(m.cfg.Defaults.CacheDir, "index.html"), nil
}

func (m IndexModel) openInBrowser(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	_ = cmd.Start()
}

func (m IndexModel) listHeight() int {
	// Reserve rows: 2 padding top, 3 header, 1 search, 1 spacing, 2 footer
	reserved := 9
	return max(m.height-reserved, 4)
}

func (m IndexModel) View() string {
	outerPad := lipgloss.NewStyle().Padding(1, 3)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cachedStyle := tui.StyleCached
	uncachedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selectedBg := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("255"))

	var b strings.Builder

	// Header
	total := len(m.books)
	cached := 0
	for _, bk := range m.books {
		if bk.IsCached {
			cached++
		}
	}
	headerText := fmt.Sprintf("Library Index  %s", dim.Render(fmt.Sprintf("(%d books, %d cached)", total, cached)))
	b.WriteString(tui.StyleHeader.Render(headerText))
	b.WriteString("\n\n")

	// Search bar
	if m.searching {
		b.WriteString(tui.StyleHighlight.Render("/") + " " + m.query + "█")
	} else if m.query != "" {
		b.WriteString(dim.Render(fmt.Sprintf("filter: %q — press / to change, esc to clear", m.query)))
	} else {
		b.WriteString(dim.Render("Press / to search"))
	}
	b.WriteString("\n\n")

	// Book list
	listH := m.listHeight()
	end := min(m.offset+listH, len(m.filtered))

	if len(m.filtered) == 0 {
		b.WriteString(dim.Render("  No books match."))
		b.WriteString("\n")
	}

	for i := m.offset; i < end; i++ {
		bk := m.filtered[i]

		var indicator string
		var titleStyle lipgloss.Style
		if bk.IsCached {
			indicator = cachedStyle.Render("●")
			titleStyle = tui.StyleNormal
		} else {
			indicator = uncachedStyle.Render("○")
			titleStyle = uncachedStyle
		}

		tags := ""
		if len(bk.Book.Tags) > 0 {
			tags = " " + dim.Render("["+strings.Join(bk.Book.Tags, ", ")+"]")
		}

		shelf := ""
		if len(m.cfg.Shelves) > 1 {
			shelf = " " + dim.Render("("+bk.ShelfName+")")
		}

		row := fmt.Sprintf("  %s  %s  %s%s%s",
			indicator,
			titleStyle.Render(bk.Book.Title),
			dim.Render("by "+bk.Book.Author),
			tags,
			shelf,
		)

		if i == m.cursor {
			row = selectedBg.Render(fmt.Sprintf("  %s  %-30s  %s%s%s",
				indicator,
				bk.Book.Title,
				"by "+bk.Book.Author,
				tags,
				shelf,
			))
		}

		b.WriteString(row)
		b.WriteString("\n")
	}

	// Scroll hint
	if len(m.filtered) > listH {
		shown := fmt.Sprintf("%d–%d of %d", m.offset+1, end, len(m.filtered))
		b.WriteString("\n")
		b.WriteString(dim.Render("  " + shown))
	}

	b.WriteString("\n")

	// Status / help bar
	if m.statusMsg != "" {
		b.WriteString(tui.StyleCached.Render("  "+m.statusMsg) + "\n")
	} else {
		help := "↑/↓ navigate · / search · g generate HTML · o generate & open · esc back"
		b.WriteString(tui.StyleHelp.Render("  "+help) + "\n")
	}

	innerPad := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return outerPad.Render(tui.StyleBorder.Render(innerPad.Render(b.String())))
}

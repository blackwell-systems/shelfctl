package unified

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type syncAllPhase int

const (
	syncAllDetecting  syncAllPhase = iota // scanning catalogs
	syncAllConfirming                     // show list, wait for confirmation
	syncAllProcessing                     // uploading one-by-one
	syncAllDone                           // summary
)

type syncEntry struct {
	shelf       config.ShelfConfig
	book        catalog.Book
	cachedPath  string
	size        int64
	newSHA      string
	releaseTag  string
	catalogPath string
	owner       string
}

type syncDetectedMsg struct {
	books []syncEntry
}

type syncProgressMsg struct {
	idx    int
	bookID string
	err    error
}

// SyncAllModel is the unified view for syncing all modified cached books back to GitHub
type SyncAllModel struct {
	phase    syncAllPhase
	gh       *github.Client
	cfg      *config.Config
	cacheMgr *cache.Manager

	books   []syncEntry
	current int
	synced  int
	errors  []string

	width, height int
	activeCmd     string
}

// NewSyncAllModel creates a new sync-all view
func NewSyncAllModel(gh *github.Client, cfg *config.Config, cacheMgr *cache.Manager) SyncAllModel {
	return SyncAllModel{
		phase:    syncAllDetecting,
		gh:       gh,
		cfg:      cfg,
		cacheMgr: cacheMgr,
	}
}

func (m SyncAllModel) Init() tea.Cmd {
	return m.detectAsync()
}

func (m SyncAllModel) Update(msg tea.Msg) (SyncAllModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tui.ClearActiveCmdMsg:
		m.activeCmd = ""
		return m, nil

	case tea.KeyMsg:
		switch m.phase {
		case syncAllDetecting:
			if msg.String() == "ctrl+c" {
				return m, func() tea.Msg { return QuitAppMsg{} }
			}
		case syncAllConfirming:
			switch msg.String() {
			case "ctrl+c":
				return m, func() tea.Msg { return QuitAppMsg{} }
			case "q", "esc":
				return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
			case "enter", "y":
				m.phase = syncAllProcessing
				m.current = 0
				m.activeCmd = "enter"
				return m, tea.Batch(m.syncBookCmd(0), tui.HighlightCmd())
			}
		case syncAllProcessing:
			if msg.String() == "ctrl+c" {
				return m, func() tea.Msg { return QuitAppMsg{} }
			}
		case syncAllDone:
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		}

	case syncDetectedMsg:
		if len(msg.books) == 0 {
			m.phase = syncAllDone
			return m, nil
		}
		m.books = msg.books
		m.phase = syncAllConfirming
		return m, nil

	case syncProgressMsg:
		if msg.err != nil {
			m.errors = append(m.errors, fmt.Sprintf("%s: %v", msg.bookID, msg.err))
		} else {
			m.synced++
		}
		m.current++
		if m.current >= len(m.books) {
			m.phase = syncAllDone
			return m, nil
		}
		return m, m.syncBookCmd(m.current)
	}

	return m, nil
}

func (m SyncAllModel) detectAsync() tea.Cmd {
	gh := m.gh
	cfg := m.cfg
	cacheMgr := m.cacheMgr

	return func() tea.Msg {
		var books []syncEntry

		for _, shelf := range cfg.Shelves {
			owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
			catalogPath := shelf.EffectiveCatalogPath()
			releaseTag := shelf.EffectiveRelease(cfg.Defaults.Release)

			data, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
			if err != nil {
				continue
			}
			parsed, err := catalog.Parse(data)
			if err != nil {
				continue
			}

			for _, b := range parsed {
				if !cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
					continue
				}
				cachedPath := cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)
				newSHA, size, err := syncComputeHash(cachedPath)
				if err != nil {
					continue
				}
				if newSHA == b.Checksum.SHA256 {
					continue // not modified
				}
				shelfCopy := shelf
				books = append(books, syncEntry{
					shelf:       shelfCopy,
					book:        b,
					cachedPath:  cachedPath,
					size:        size,
					newSHA:      newSHA,
					releaseTag:  releaseTag,
					catalogPath: catalogPath,
					owner:       owner,
				})
			}
		}

		return syncDetectedMsg{books: books}
	}
}

func (m SyncAllModel) syncBookCmd(idx int) tea.Cmd {
	if idx >= len(m.books) {
		return nil
	}
	entry := m.books[idx]
	gh := m.gh

	return func() tea.Msg {
		owner := entry.owner
		repo := entry.shelf.Repo
		b := &entry.book

		// Get or create release
		rel, err := gh.EnsureRelease(owner, repo, entry.releaseTag)
		if err != nil {
			return syncProgressMsg{idx: idx, bookID: b.ID, err: fmt.Errorf("get release: %w", err)}
		}

		// Find and delete old asset
		oldAsset, err := gh.FindAsset(owner, repo, rel.ID, b.Source.Asset)
		if err != nil {
			return syncProgressMsg{idx: idx, bookID: b.ID, err: fmt.Errorf("find asset: %w", err)}
		}
		if oldAsset != nil {
			if err := gh.DeleteAsset(owner, repo, oldAsset.ID); err != nil {
				return syncProgressMsg{idx: idx, bookID: b.ID, err: fmt.Errorf("delete asset: %w", err)}
			}
		}

		// Upload modified file
		f, err := os.Open(entry.cachedPath)
		if err != nil {
			return syncProgressMsg{idx: idx, bookID: b.ID, err: fmt.Errorf("open file: %w", err)}
		}
		defer func() { _ = f.Close() }()

		_, err = gh.UploadAsset(owner, repo, rel.ID, b.Source.Asset, f, entry.size, "application/octet-stream")
		if err != nil {
			return syncProgressMsg{idx: idx, bookID: b.ID, err: fmt.Errorf("upload: %w", err)}
		}

		// Reload catalog and update checksum
		data, _, err := gh.GetFileContent(owner, repo, entry.catalogPath, "")
		if err != nil {
			return syncProgressMsg{idx: idx, bookID: b.ID, err: fmt.Errorf("reload catalog: %w", err)}
		}
		books, err := catalog.Parse(data)
		if err != nil {
			return syncProgressMsg{idx: idx, bookID: b.ID, err: fmt.Errorf("parse catalog: %w", err)}
		}
		bookPtr := catalog.ByID(books, b.ID)
		if bookPtr == nil {
			return syncProgressMsg{idx: idx, bookID: b.ID, err: fmt.Errorf("book not found in catalog after upload")}
		}
		bookPtr.Checksum.SHA256 = entry.newSHA
		bookPtr.SizeBytes = entry.size

		mgr := catalog.NewManager(gh, owner, repo, entry.catalogPath)
		commitMsg := fmt.Sprintf("sync: update %s with local changes", b.ID)
		if err := mgr.Save(books, commitMsg); err != nil {
			return syncProgressMsg{idx: idx, bookID: b.ID, err: fmt.Errorf("save catalog: %w", err)}
		}

		return syncProgressMsg{idx: idx, bookID: b.ID, err: nil}
	}
}

func (m SyncAllModel) View() string {
	outerStyle := lipgloss.NewStyle().Padding(2, 4)
	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)

	var b strings.Builder

	switch m.phase {
	case syncAllDetecting:
		b.WriteString(tui.StyleHeader.Render("Sync Modified Books"))
		b.WriteString("\n\n")
		b.WriteString(tui.StyleHighlight.Render("⏳ Detecting modified books..."))
		b.WriteString("\n")
		b.WriteString(tui.StyleHelp.Render("Checking local cache against catalog checksums"))

	case syncAllConfirming:
		b.WriteString(tui.StyleHeader.Render("Sync Modified Books"))
		b.WriteString("\n\n")
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("%d book(s) with local changes:", len(m.books))))
		b.WriteString("\n\n")
		for _, entry := range m.books {
			fmt.Fprintf(&b, "  • %s  %s\n",
				tui.StyleHighlight.Render(entry.book.ID),
				tui.StyleHelp.Render(entry.book.Title),
			)
		}
		b.WriteString("\n")
		b.WriteString(tui.RenderFooterBar([]tui.ShortcutEntry{
			{Key: "enter", Label: "Enter/y Sync All"},
			{Key: "q", Label: "Esc/q Cancel"},
		}, m.activeCmd))

	case syncAllProcessing:
		b.WriteString(tui.StyleHeader.Render("Sync Modified Books"))
		b.WriteString("\n\n")
		for i, entry := range m.books {
			var line string
			if i < m.current {
				// Completed — check if it errored
				errored := false
				prefix := entry.book.ID + ":"
				for _, e := range m.errors {
					if strings.HasPrefix(e, prefix) {
						errored = true
						break
					}
				}
				if errored {
					line = tui.StyleNormal.Render("✗ " + entry.book.ID)
				} else {
					line = tui.StyleCached.Render("✓ " + entry.book.ID)
				}
			} else if i == m.current {
				line = tui.StyleHighlight.Render("⏳ " + entry.book.ID)
			} else {
				line = tui.StyleHelp.Render("  " + entry.book.ID)
			}
			b.WriteString("  " + line + "\n")
		}

	case syncAllDone:
		b.WriteString(tui.StyleHeader.Render("Sync Modified Books"))
		b.WriteString("\n\n")
		if m.synced == 0 && len(m.errors) == 0 {
			b.WriteString(tui.StyleHelp.Render("No modified books found."))
		} else {
			if m.synced > 0 {
				b.WriteString(tui.StyleCached.Render(fmt.Sprintf("✓ Synced %d book(s)", m.synced)))
				b.WriteString("\n")
			}
			if len(m.errors) > 0 {
				fmt.Fprintf(&b, "%s\n", tui.StyleNormal.Render(fmt.Sprintf("✗ %d error(s):", len(m.errors))))
				for _, e := range m.errors {
					fmt.Fprintf(&b, "%s\n", tui.StyleHelp.Render("  "+e))
				}
			}
		}
		b.WriteString("\n")
		b.WriteString(tui.StyleHelp.Render("Press any key to return to menu"))
	}

	return outerStyle.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

// syncComputeHash computes the SHA256 and byte size of a local file.
func syncComputeHash(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	size, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), size, nil
}

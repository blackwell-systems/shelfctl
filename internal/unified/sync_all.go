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
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
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

// syncDetectedMsg carries the list of modified books found during detection.
type syncDetectedMsg struct {
	books []syncEntry
}

// syncUploadReadyMsg signals that the upload goroutine has started and progress
// can be read from ch.
type syncUploadReadyMsg struct {
	idx   int
	total int64
	ch    <-chan int64
	errCh <-chan error
	entry syncEntry
}

// syncUploadTickMsg carries a progress update from the upload goroutine,
// along with the channels needed to read the next tick.
type syncUploadTickMsg struct {
	idx   int
	bytes int64
	total int64
	ch    <-chan int64
	errCh <-chan error
	entry syncEntry
}

// syncUploadDoneMsg signals the upload goroutine has finished.
type syncUploadDoneMsg struct {
	idx   int
	err   error
	entry syncEntry
}

// syncProgressMsg signals that a full book sync (upload + catalog commit) is done.
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

	// Upload progress for current book
	uploadProgress progress.Model
	uploadCurrent  int64
	uploadTotal    int64

	spinner       spinner.Model
	width, height int
	activeCmd     string
}

// NewSyncAllModel creates a new sync-all view
func NewSyncAllModel(gh *github.Client, cfg *config.Config, cacheMgr *cache.Manager) SyncAllModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = tui.StyleHighlight

	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 40

	return SyncAllModel{
		phase:          syncAllDetecting,
		gh:             gh,
		cfg:            cfg,
		cacheMgr:       cacheMgr,
		spinner:        s,
		uploadProgress: prog,
	}
}

func (m SyncAllModel) Init() tea.Cmd {
	return tea.Batch(m.detectAsync(), m.spinner.Tick)
}

func (m SyncAllModel) Update(msg tea.Msg) (SyncAllModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.uploadProgress.Width = min(max(msg.Width-24, 20), 60)
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
				return m, tea.Batch(m.syncBookSetupCmd(0), tui.HighlightCmd())
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

	case syncUploadReadyMsg:
		m.uploadCurrent = 0
		m.uploadTotal = msg.total
		return m, waitForSyncTick(msg.idx, msg.total, msg.entry, msg.ch, msg.errCh)

	case syncUploadTickMsg:
		m.uploadCurrent = msg.bytes
		return m, waitForSyncTick(msg.idx, msg.total, msg.entry, msg.ch, msg.errCh)

	case syncUploadDoneMsg:
		m.uploadCurrent = m.uploadTotal // show 100%
		if msg.err != nil {
			m.errors = append(m.errors, fmt.Sprintf("%s: %v", msg.entry.book.ID, msg.err))
			m.current++
			if m.current >= len(m.books) {
				m.phase = syncAllDone
				return m, nil
			}
			return m, m.syncBookSetupCmd(m.current)
		}
		return m, m.syncCatalogCmd(msg.idx, msg.entry)

	case syncProgressMsg:
		if msg.err != nil {
			m.errors = append(m.errors, fmt.Sprintf("%s: %v", msg.bookID, msg.err))
		} else {
			m.synced++
		}
		m.current++
		m.uploadCurrent = 0
		m.uploadTotal = 0
		if m.current >= len(m.books) {
			m.phase = syncAllDone
			return m, nil
		}
		return m, m.syncBookSetupCmd(m.current)

	case progress.FrameMsg:
		progressModel, cmd := m.uploadProgress.Update(msg)
		m.uploadProgress = progressModel.(progress.Model)
		return m, cmd

	case spinner.TickMsg:
		if m.phase == syncAllDetecting || m.phase == syncAllProcessing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
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

// syncBookSetupCmd does pre-upload work (EnsureRelease, FindAsset, DeleteAsset),
// opens the cached file, starts the upload goroutine, and returns
// syncUploadReadyMsg so the progress reading chain can begin.
func (m SyncAllModel) syncBookSetupCmd(idx int) tea.Cmd {
	if idx >= len(m.books) {
		return nil
	}
	entry := m.books[idx]
	gh := m.gh

	return func() tea.Msg {
		owner := entry.owner
		repo := entry.shelf.Repo
		b := &entry.book

		rel, err := gh.EnsureRelease(owner, repo, entry.releaseTag)
		if err != nil {
			return syncProgressMsg{idx: idx, bookID: b.ID, err: fmt.Errorf("get release: %w", err)}
		}

		oldAsset, err := gh.FindAsset(owner, repo, rel.ID, b.Source.Asset)
		if err != nil {
			return syncProgressMsg{idx: idx, bookID: b.ID, err: fmt.Errorf("find asset: %w", err)}
		}
		if oldAsset != nil {
			if err := gh.DeleteAsset(owner, repo, oldAsset.ID); err != nil {
				return syncProgressMsg{idx: idx, bookID: b.ID, err: fmt.Errorf("delete asset: %w", err)}
			}
		}

		f, err := os.Open(entry.cachedPath)
		if err != nil {
			return syncProgressMsg{idx: idx, bookID: b.ID, err: fmt.Errorf("open file: %w", err)}
		}

		progressCh := make(chan int64, 100)
		errCh := make(chan error, 1)

		go func() {
			defer func() { _ = f.Close() }()
			pr := tui.NewProgressReader(f, entry.size, progressCh)
			_, uploadErr := gh.UploadAsset(owner, repo, rel.ID, b.Source.Asset, pr, entry.size, "application/octet-stream")
			close(progressCh)
			errCh <- uploadErr
		}()

		return syncUploadReadyMsg{idx: idx, total: entry.size, ch: progressCh, errCh: errCh, entry: entry}
	}
}

// waitForSyncTick reads one value from the upload progress channel.
// Returns syncUploadTickMsg if still in progress, syncUploadDoneMsg when complete.
func waitForSyncTick(idx int, total int64, entry syncEntry, ch <-chan int64, errCh <-chan error) tea.Cmd {
	return func() tea.Msg {
		n, ok := <-ch
		if !ok {
			err := <-errCh
			return syncUploadDoneMsg{idx: idx, err: err, entry: entry}
		}
		return syncUploadTickMsg{idx: idx, bytes: n, total: total, ch: ch, errCh: errCh, entry: entry}
	}
}

// syncCatalogCmd reloads the catalog, updates the book's checksum and size, and commits.
func (m SyncAllModel) syncCatalogCmd(idx int, entry syncEntry) tea.Cmd {
	gh := m.gh
	return func() tea.Msg {
		owner := entry.owner
		repo := entry.shelf.Repo
		b := &entry.book

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
		fmt.Fprintf(&b, "%s Detecting modified books...\n", m.spinner.View())
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
				line = m.spinner.View() + " " + tui.StyleHighlight.Render(entry.book.ID)
			} else {
				line = tui.StyleHelp.Render("  " + entry.book.ID)
			}
			b.WriteString("  " + line + "\n")
		}

		// Upload progress bar for current book
		if m.uploadTotal > 0 {
			b.WriteString("\n")
			percent := float64(m.uploadCurrent) / float64(m.uploadTotal)
			currentMB := float64(m.uploadCurrent) / 1024 / 1024
			totalMB := float64(m.uploadTotal) / 1024 / 1024
			b.WriteString("  " + m.uploadProgress.ViewAs(percent) + "\n")
			fmt.Fprintf(&b, "  %s\n",
				tui.StyleHelp.Render(fmt.Sprintf("%.1f / %.1f MB  (%.0f%%)", currentMB, totalMB, percent*100)),
			)
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

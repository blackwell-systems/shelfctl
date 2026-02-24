package unified

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/blackwell-systems/bubbletea-components/multiselect"
	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/operations"
	"github.com/blackwell-systems/shelfctl/internal/readme"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// movePhase tracks the current phase of the move workflow
type movePhase int

const (
	moveBookPicking movePhase = iota // Multi-select book picker
	moveTypePicking                  // Choose "Different shelf" or "Different release"
	moveDestPicking                  // Shelf picker or release tag input
	moveConfirming                   // Show summary, y/n
	moveProcessing                   // Async batch move
)

// Move type constants
const (
	moveToShelf   = 0
	moveToRelease = 1
)

// MoveBookCompleteMsg is emitted when move finishes
type MoveBookCompleteMsg struct {
	SuccessCount int
	FailCount    int
}

// moveCompleteMsg is the internal async completion message
type moveCompleteMsg struct {
	successCount int
	failCount    int
}

// MoveBookModel is the unified view for moving books
type MoveBookModel struct {
	phase    movePhase
	ms       multiselect.Model
	gh       *github.Client
	cfg      *config.Config
	cacheMgr *cache.Manager
	width    int
	height   int
	err      error
	empty    bool

	// Selected books
	toMove []tui.BookItem

	// Move type selection
	moveType     int // moveToShelf or moveToRelease
	typeSelected int // cursor position: 0 = shelf, 1 = release

	// Destination
	destShelfList list.Model
	destShelfName string
	destRelease   string
	releaseInput  textinput.Model

	// For release move: validation
	allSameShelf bool
	sourceShelf  string

	// Results
	successCount int
	failCount    int
}

// NewMoveBookModel creates a new move-book view
func NewMoveBookModel(books []tui.BookItem, gh *github.Client, cfg *config.Config, cacheMgr *cache.Manager) MoveBookModel {
	if len(books) == 0 {
		return MoveBookModel{
			gh:       gh,
			cfg:      cfg,
			cacheMgr: cacheMgr,
			empty:    true,
		}
	}

	ms, err := tui.NewBookPickerMultiModel(books, "Select books to move")
	if err != nil {
		return MoveBookModel{
			gh:       gh,
			cfg:      cfg,
			cacheMgr: cacheMgr,
			err:      err,
		}
	}

	return MoveBookModel{
		phase:    moveBookPicking,
		ms:       ms,
		gh:       gh,
		cfg:      cfg,
		cacheMgr: cacheMgr,
	}
}

// Init initializes the move-book view
func (m MoveBookModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the move-book view
func (m MoveBookModel) Update(msg tea.Msg) (MoveBookModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		switch m.phase {
		case moveBookPicking:
			h, v := tui.StyleBorder.GetFrameSize()
			m.ms.List.SetSize(msg.Width-h, msg.Height-v)
			m.ms.List.Title = tui.StyleHeader.Render("Select books to move") + "\n" + tui.RenderColumnHeader(m.ms.List.Width())
			m.ms.List.Styles.Title = lipgloss.NewStyle()
		case moveDestPicking:
			if m.moveType == moveToShelf {
				h, v := tui.StyleBorder.GetFrameSize()
				m.destShelfList.SetSize(msg.Width-h, msg.Height-v)
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.empty || (m.err != nil && m.phase == moveBookPicking) {
			switch msg.String() {
			case "enter", "esc", "q":
				return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
			}
			return m, nil
		}

		switch m.phase {
		case moveBookPicking:
			return m.updateBookPicking(msg)
		case moveTypePicking:
			return m.updateTypePicking(msg)
		case moveDestPicking:
			return m.updateDestPicking(msg)
		case moveConfirming:
			return m.updateConfirming(msg)
		case moveProcessing:
			if msg.String() == "ctrl+c" {
				return m, func() tea.Msg { return QuitAppMsg{} }
			}
			return m, nil
		}

	case moveCompleteMsg:
		m.successCount = msg.successCount
		m.failCount = msg.failCount
		return m, func() tea.Msg {
			return NavigateMsg{Target: "hub"}
		}
	}

	// Forward non-key messages to sub-models
	switch m.phase {
	case moveBookPicking:
		var cmd tea.Cmd
		m.ms, cmd = m.ms.Update(msg)
		return m, cmd
	case moveDestPicking:
		if m.moveType == moveToShelf {
			var cmd tea.Cmd
			m.destShelfList, cmd = m.destShelfList.Update(msg)
			return m, cmd
		}
		// Release text input
		var cmd tea.Cmd
		m.releaseInput, cmd = m.releaseInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// --- Phase: Book Picking ---

func (m MoveBookModel) updateBookPicking(msg tea.KeyMsg) (MoveBookModel, tea.Cmd) {
	if m.ms.List.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.ms, cmd = m.ms.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }
	case "q", "esc":
		return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
	case " ":
		m.ms.Toggle()
		return m, nil
	case "enter":
		selected := tui.CollectSelectedBooks(&m.ms)
		if len(selected) == 0 {
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		}
		m.toMove = selected

		// Check if all selected books are from the same shelf
		m.sourceShelf = selected[0].ShelfName
		m.allSameShelf = true
		for _, item := range selected {
			if item.ShelfName != m.sourceShelf {
				m.allSameShelf = false
				break
			}
		}

		// Transition to type picking
		m.phase = moveTypePicking
		m.typeSelected = 0
		return m, nil
	}

	var cmd tea.Cmd
	m.ms, cmd = m.ms.Update(msg)
	return m, cmd
}

// --- Phase: Type Picking ---

func (m MoveBookModel) updateTypePicking(msg tea.KeyMsg) (MoveBookModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }
	case "esc":
		// Go back to book picking
		m.phase = moveBookPicking
		return m, nil
	case "up", "k":
		if m.typeSelected > 0 {
			m.typeSelected--
		}
		return m, nil
	case "down", "j":
		if m.typeSelected < 1 {
			m.typeSelected++
		}
		return m, nil
	case "1":
		m.typeSelected = moveToShelf
		return m.selectMoveType()
	case "2":
		m.typeSelected = moveToRelease
		return m.selectMoveType()
	case "enter":
		return m.selectMoveType()
	}
	return m, nil
}

func (m MoveBookModel) selectMoveType() (MoveBookModel, tea.Cmd) {
	m.moveType = m.typeSelected

	if m.moveType == moveToShelf {
		// Build shelf options (exclude source shelves)
		excludedShelves := make(map[string]bool)
		for _, item := range m.toMove {
			excludedShelves[item.ShelfName] = true
		}

		var options []tui.ShelfOption
		for _, s := range m.cfg.Shelves {
			if !excludedShelves[s.Name] {
				options = append(options, tui.ShelfOption{
					Name: s.Name,
					Repo: s.Repo,
				})
			}
		}

		if len(options) == 0 {
			m.err = fmt.Errorf("no other shelves available — create another shelf first")
			return m, nil
		}

		// If only one option, auto-select
		if len(options) == 1 {
			m.destShelfName = options[0].Name
			m.phase = moveConfirming
			return m, nil
		}

		// Build shelf list
		items := make([]list.Item, len(options))
		for i, s := range options {
			items[i] = s
		}
		d := tui.NewShelfDelegate()
		l := list.New(items, d, 0, 0)
		l.Title = "Select Destination Shelf"
		l.SetShowStatusBar(false)
		l.SetFilteringEnabled(true)
		l.Styles.Title = tui.StyleHeader
		l.Styles.HelpStyle = tui.StyleHelp

		m.destShelfList = l
		m.phase = moveDestPicking
		return m, func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		}

	} else {
		// Moving to different release
		if !m.allSameShelf {
			m.err = fmt.Errorf("cannot move books from different shelves to a different release — use different shelf option instead")
			// Go back to type picking
			m.phase = moveTypePicking
			return m, nil
		}

		// Show text input for release tag
		ti := textinput.New()
		ti.Placeholder = "release-tag"
		ti.Focus()
		ti.CharLimit = 100
		ti.Width = 40
		ti.Prompt = ""
		m.releaseInput = ti
		m.phase = moveDestPicking
		return m, textinput.Blink
	}
}

// --- Phase: Destination Picking ---

func (m MoveBookModel) updateDestPicking(msg tea.KeyMsg) (MoveBookModel, tea.Cmd) {
	if m.moveType == moveToShelf {
		return m.updateDestShelfPicking(msg)
	}
	return m.updateDestReleasePicking(msg)
}

func (m MoveBookModel) updateDestShelfPicking(msg tea.KeyMsg) (MoveBookModel, tea.Cmd) {
	if m.destShelfList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.destShelfList, cmd = m.destShelfList.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }
	case "esc":
		// Go back to type picking
		m.phase = moveTypePicking
		m.err = nil
		return m, nil
	case "enter":
		if item, ok := m.destShelfList.SelectedItem().(tui.ShelfOption); ok {
			m.destShelfName = item.Name
			m.phase = moveConfirming
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.destShelfList, cmd = m.destShelfList.Update(msg)
	return m, cmd
}

func (m MoveBookModel) updateDestReleasePicking(msg tea.KeyMsg) (MoveBookModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }
	case "esc":
		m.phase = moveTypePicking
		m.err = nil
		return m, nil
	case "enter":
		tag := strings.TrimSpace(m.releaseInput.Value())
		if tag == "" {
			m.err = fmt.Errorf("release tag is required")
			return m, nil
		}
		m.destRelease = tag
		m.phase = moveConfirming
		return m, nil
	}

	var cmd tea.Cmd
	m.releaseInput, cmd = m.releaseInput.Update(msg)
	return m, cmd
}

// --- Phase: Confirming ---

func (m MoveBookModel) updateConfirming(msg tea.KeyMsg) (MoveBookModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }
	case "esc", "n":
		// Go back to destination picking
		if m.moveType == moveToShelf {
			m.phase = moveDestPicking
		} else {
			m.phase = moveDestPicking
		}
		return m, nil
	case "enter", "y":
		m.phase = moveProcessing
		return m, m.moveAsync()
	}

	return m, nil
}

// --- Phase: Processing (async) ---

func (m MoveBookModel) moveAsync() tea.Cmd {
	toMove := m.toMove
	moveType := m.moveType
	destShelfName := m.destShelfName
	destRelease := m.destRelease
	gh := m.gh
	cfg := m.cfg
	cacheMgr := m.cacheMgr

	return func() tea.Msg {
		successCount := 0
		failCount := 0

		for _, item := range toMove {
			var err error
			if moveType == moveToShelf {
				err = moveSingleBookToShelf(item, destShelfName, gh, cfg, cacheMgr)
			} else {
				err = moveSingleBookToRelease(item, destRelease, gh, cfg)
			}

			if err != nil {
				failCount++
			} else {
				successCount++
			}
		}

		return moveCompleteMsg{
			successCount: successCount,
			failCount:    failCount,
		}
	}
}

// moveSingleBookToShelf moves a book to a different shelf (cross-repo)
func moveSingleBookToShelf(item tui.BookItem, destShelfName string, gh *github.Client, cfg *config.Config, cacheMgr *cache.Manager) error {
	srcShelf := cfg.ShelfByName(item.ShelfName)
	if srcShelf == nil {
		return fmt.Errorf("source shelf %q not found", item.ShelfName)
	}
	dstShelf := cfg.ShelfByName(destShelfName)
	if dstShelf == nil {
		return fmt.Errorf("destination shelf %q not found", destShelfName)
	}

	srcOwner := srcShelf.EffectiveOwner(cfg.GitHub.Owner)
	dstOwner := dstShelf.EffectiveOwner(cfg.GitHub.Owner)
	dstRelease := dstShelf.EffectiveRelease(cfg.Defaults.Release)

	b := &item.Book

	// Check if already at destination
	if srcOwner == dstOwner && srcShelf.Repo == dstShelf.Repo {
		return fmt.Errorf("book is already in shelf %q", destShelfName)
	}

	// 1. Ensure destination release
	dstRel, err := gh.EnsureRelease(dstOwner, dstShelf.Repo, dstRelease)
	if err != nil {
		return fmt.Errorf("ensuring destination release: %w", err)
	}

	// 2. Get source asset
	srcRel, err := gh.GetReleaseByTag(srcOwner, srcShelf.Repo, b.Source.Release)
	if err != nil {
		return fmt.Errorf("getting source release: %w", err)
	}
	srcAsset, err := gh.FindAsset(srcOwner, srcShelf.Repo, srcRel.ID, b.Source.Asset)
	if err != nil {
		return fmt.Errorf("finding source asset: %w", err)
	}
	if srcAsset == nil {
		return fmt.Errorf("source asset %q not found", b.Source.Asset)
	}

	// 3. Download and buffer
	tmpPath, size, err := downloadAndBufferAsset(gh, srcOwner, srcShelf.Repo, srcAsset.ID)
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	defer func() { _ = os.Remove(tmpPath) }()

	// 4. Upload to destination
	uploadFile, err := os.Open(tmpPath)
	if err != nil {
		return err
	}
	defer func() { _ = uploadFile.Close() }()

	_, err = gh.UploadAsset(dstOwner, dstShelf.Repo, dstRel.ID, b.Source.Asset,
		uploadFile, size, "application/octet-stream")
	if err != nil {
		return fmt.Errorf("uploading to destination: %w", err)
	}

	// 5. Delete old asset
	if err := gh.DeleteAsset(srcOwner, srcShelf.Repo, srcAsset.ID); err != nil {
		// Warn but continue — asset was already copied
		_ = err
	}

	// 6. Update source catalog (remove book)
	srcCatalogPath := srcShelf.EffectiveCatalogPath()
	srcData, _, err := gh.GetFileContent(srcOwner, srcShelf.Repo, srcCatalogPath, "")
	if err != nil {
		return fmt.Errorf("loading source catalog: %w", err)
	}
	srcBooks, err := catalog.Parse(srcData)
	if err != nil {
		return fmt.Errorf("parsing source catalog: %w", err)
	}
	srcBooks, _ = catalog.Remove(srcBooks, b.ID)
	srcMarshal, err := catalog.Marshal(srcBooks)
	if err != nil {
		return fmt.Errorf("marshaling source catalog: %w", err)
	}
	if err := gh.CommitFile(srcOwner, srcShelf.Repo, srcCatalogPath, srcMarshal,
		fmt.Sprintf("move: remove %s (moved to %s)", b.ID, destShelfName)); err != nil {
		return fmt.Errorf("committing source catalog: %w", err)
	}

	// 7. Update destination catalog (add book)
	dstCatalogPath := dstShelf.EffectiveCatalogPath()
	dstData, _, _ := gh.GetFileContent(dstOwner, dstShelf.Repo, dstCatalogPath, "")
	dstBooks, _ := catalog.Parse(dstData)

	// Update book metadata for destination
	movedBook := *b
	movedBook.Source.Release = dstRelease
	movedBook.Source.Owner = dstOwner
	movedBook.Source.Repo = dstShelf.Repo

	dstBooks = catalog.Append(dstBooks, movedBook)
	dstMarshal, err := catalog.Marshal(dstBooks)
	if err != nil {
		return fmt.Errorf("marshaling destination catalog: %w", err)
	}
	if err := gh.CommitFile(dstOwner, dstShelf.Repo, dstCatalogPath, dstMarshal,
		fmt.Sprintf("move: add %s (from %s)", b.ID, item.ShelfName)); err != nil {
		return fmt.Errorf("committing destination catalog: %w", err)
	}

	// 8. Clear local cache (path changes after move)
	if cacheMgr.Exists(srcOwner, srcShelf.Repo, b.ID, b.Source.Asset) {
		_ = cacheMgr.Remove(srcOwner, srcShelf.Repo, b.ID, b.Source.Asset)
	}

	// 9. Update README files
	// Source: remove book
	srcReadmeData, _, err := gh.GetFileContent(srcOwner, srcShelf.Repo, "README.md", "")
	if err == nil {
		originalContent := string(srcReadmeData)
		readmeContent := operations.UpdateShelfREADMEStats(originalContent, len(srcBooks))
		readmeContent = operations.RemoveFromShelfREADME(readmeContent, b.ID)
		if readmeContent != originalContent {
			_ = gh.CommitFile(srcOwner, srcShelf.Repo, "README.md", []byte(readmeContent),
				fmt.Sprintf("Update README: remove %s", b.ID))
		}
	}

	// Destination: add book
	dstReadmeMgr := readme.NewUpdater(gh, dstOwner, dstShelf.Repo)
	_ = dstReadmeMgr.UpdateWithStats(len(dstBooks), []catalog.Book{movedBook})

	// 10. Handle catalog cover if it exists
	if b.Cover != "" {
		coverData, _, err := gh.GetFileContent(srcOwner, srcShelf.Repo, b.Cover, "")
		if err == nil {
			_ = gh.CommitFile(dstOwner, dstShelf.Repo, b.Cover, coverData,
				fmt.Sprintf("move: copy cover for %s", b.ID))
		}
	}

	return nil
}

// moveSingleBookToRelease moves a book to a different release within the same shelf
func moveSingleBookToRelease(item tui.BookItem, destRelease string, gh *github.Client, cfg *config.Config) error {
	shelf := cfg.ShelfByName(item.ShelfName)
	if shelf == nil {
		return fmt.Errorf("shelf %q not found", item.ShelfName)
	}

	owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
	b := &item.Book

	// Check if already at destination
	if b.Source.Release == destRelease {
		return fmt.Errorf("book is already at release %q", destRelease)
	}

	// 1. Ensure destination release
	dstRel, err := gh.EnsureRelease(owner, shelf.Repo, destRelease)
	if err != nil {
		return fmt.Errorf("ensuring destination release: %w", err)
	}

	// 2. Get source asset
	srcRel, err := gh.GetReleaseByTag(owner, shelf.Repo, b.Source.Release)
	if err != nil {
		return fmt.Errorf("getting source release: %w", err)
	}
	srcAsset, err := gh.FindAsset(owner, shelf.Repo, srcRel.ID, b.Source.Asset)
	if err != nil {
		return fmt.Errorf("finding source asset: %w", err)
	}
	if srcAsset == nil {
		return fmt.Errorf("source asset %q not found", b.Source.Asset)
	}

	// 3. Download and buffer
	tmpPath, size, err := downloadAndBufferAsset(gh, owner, shelf.Repo, srcAsset.ID)
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	defer func() { _ = os.Remove(tmpPath) }()

	// 4. Upload to destination release
	uploadFile, err := os.Open(tmpPath)
	if err != nil {
		return err
	}
	defer func() { _ = uploadFile.Close() }()

	_, err = gh.UploadAsset(owner, shelf.Repo, dstRel.ID, b.Source.Asset,
		uploadFile, size, "application/octet-stream")
	if err != nil {
		return fmt.Errorf("uploading: %w", err)
	}

	// 5. Delete old asset
	if err := gh.DeleteAsset(owner, shelf.Repo, srcAsset.ID); err != nil {
		_ = err // Warn but continue
	}

	// 6. Update catalog (change release field)
	catalogPath := shelf.EffectiveCatalogPath()
	data, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}
	books, err := catalog.Parse(data)
	if err != nil {
		return fmt.Errorf("parsing catalog: %w", err)
	}

	for i := range books {
		if books[i].ID == b.ID {
			books[i].Source.Release = destRelease
			break
		}
	}

	newData, err := catalog.Marshal(books)
	if err != nil {
		return fmt.Errorf("marshaling catalog: %w", err)
	}
	if err := gh.CommitFile(owner, shelf.Repo, catalogPath, newData,
		fmt.Sprintf("move: %s → release/%s", b.ID, destRelease)); err != nil {
		return fmt.Errorf("committing catalog: %w", err)
	}

	return nil
}

// downloadAndBufferAsset downloads a release asset to a temp file
func downloadAndBufferAsset(gh *github.Client, owner, repo string, assetID int64) (string, int64, error) {
	rc, err := gh.DownloadAsset(owner, repo, assetID)
	if err != nil {
		return "", 0, fmt.Errorf("downloading: %w", err)
	}

	tmp, err := os.CreateTemp("", "shelfctl-move-*")
	if err != nil {
		_ = rc.Close()
		return "", 0, err
	}
	tmpPath := tmp.Name()

	if _, err := io.Copy(tmp, rc); err != nil {
		_ = tmp.Close()
		_ = rc.Close()
		_ = os.Remove(tmpPath)
		return "", 0, err
	}
	_ = tmp.Close()
	_ = rc.Close()

	fi, err := os.Stat(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, err
	}

	return tmpPath, fi.Size(), nil
}

// --- View ---

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
	b.WriteString(tui.StyleHelp.Render("↑↓/1-2: Select  Enter: Confirm  Esc: Back"))
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

	b.WriteString(tui.StyleHelp.Render("Enter: Confirm  Esc: Back"))
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

	b.WriteString(tui.StyleHelp.Render("Enter/y: Confirm   Esc/n: Back"))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

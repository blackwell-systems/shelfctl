package unified

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/blackwell-systems/bubbletea-multiselect"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/migrate"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/blackwell-systems/shelfctl/internal/tui/delegate"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// importRepoPhase tracks the current phase of the import-from-repo workflow
type importRepoPhase int

const (
	importRepoSourceInput  importRepoPhase = iota // textinput for "owner/repo"
	importRepoShelfPicking                        // shelf list picker
	importRepoScanning                            // async: scan repo for files
	importRepoFilePicking                         // multiselect from discovered files
	importRepoProcessing                          // async: per-file download+ingest+upload
	importRepoCommitting                          // async: batch catalog commit + README + ledger
	importRepoDone                                // summary
)

// Internal messages
type importRepoScanCompleteMsg struct {
	files   []migrate.FileEntry
	release *github.Release
	err     error
}

type importRepoProgressMsg struct {
	kind    string // "status", "done"
	path    string
	book    *catalog.Book
	current int
	total   int
	err     error
}

type importRepoCommitCompleteMsg struct{ err error }

// importRepoFileItem wraps migrate.FileEntry for multiselect
type importRepoFileItem struct {
	file migrate.FileEntry
	sel  bool
}

func (f *importRepoFileItem) FilterValue() string {
	return f.file.Path
}
func (f *importRepoFileItem) IsSelected() bool   { return f.sel }
func (f *importRepoFileItem) SetSelected(v bool) { f.sel = v }
func (f *importRepoFileItem) IsSelectable() bool { return true }

// ImportRepoModel is the unified view for importing files from any GitHub repo
type ImportRepoModel struct {
	phase importRepoPhase

	gh  *github.Client
	cfg *config.Config

	width, height int
	err           error
	empty         bool

	// Source input
	sourceInput textinput.Model
	srcOwner    string
	srcRepo     string

	// Shelf picker
	shelfList    list.Model
	shelfOptions []tui.ShelfOption
	destShelf    *config.ShelfConfig
	destOwner    string
	destRelTag   string
	destCatPath  string

	// Scan results
	files   []migrate.FileEntry
	release *github.Release

	// File picking
	ms multiselect.Model

	// Processing
	toImport   []migrate.FileEntry
	progressCh <-chan importRepoProgressMsg
	statusMsg  string

	// Results
	importedBooks []catalog.Book
	successCount  int
	failCount     int
}

// NewImportRepoModel creates a new import-from-repo view
func NewImportRepoModel(gh *github.Client, cfg *config.Config) ImportRepoModel {
	if len(cfg.Shelves) == 0 {
		return ImportRepoModel{gh: gh, cfg: cfg, empty: true}
	}

	ti := textinput.New()
	ti.Placeholder = "owner/repo"
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40

	var options []tui.ShelfOption
	for _, s := range cfg.Shelves {
		options = append(options, tui.ShelfOption{Name: s.Name, Repo: s.Repo})
	}

	m := ImportRepoModel{
		phase:        importRepoSourceInput,
		gh:           gh,
		cfg:          cfg,
		sourceInput:  ti,
		shelfOptions: options,
	}

	// Auto-select if single shelf
	if len(options) == 1 {
		m.resolveShelf(options[0].Name)
	}

	return m
}

func (m *ImportRepoModel) resolveShelf(name string) {
	shelf := m.cfg.ShelfByName(name)
	if shelf == nil {
		return
	}
	m.destShelf = shelf
	m.destOwner = shelf.EffectiveOwner(m.cfg.GitHub.Owner)
	m.destRelTag = shelf.EffectiveRelease(m.cfg.Defaults.Release)
	m.destCatPath = shelf.EffectiveCatalogPath()
}

// Init initializes the model
func (m ImportRepoModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m ImportRepoModel) Update(msg tea.Msg) (ImportRepoModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		switch m.phase {
		case importRepoFilePicking:
			h, v := tui.StyleBorder.GetFrameSize()
			m.ms.List.SetSize(msg.Width-h, msg.Height-v)
		case importRepoShelfPicking:
			h, v := tui.StyleBorder.GetFrameSize()
			m.shelfList.SetSize(msg.Width-h, msg.Height-v)
		}
		return m, nil

	case tea.KeyMsg:
		if m.empty {
			if msg.String() == "enter" || msg.String() == "esc" {
				return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
			}
			return m, nil
		}
		if m.phase == importRepoDone {
			if msg.String() == "enter" || msg.String() == "esc" {
				return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
			}
			return m, nil
		}

		switch m.phase {
		case importRepoSourceInput:
			return m.updateSourceInput(msg)
		case importRepoShelfPicking:
			return m.updateShelfPicking(msg)
		case importRepoFilePicking:
			return m.updateFilePicking(msg)
		case importRepoProcessing, importRepoCommitting:
			if msg.String() == "ctrl+c" {
				return m, func() tea.Msg { return QuitAppMsg{} }
			}
			return m, nil
		}

	case importRepoScanCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
			m.phase = importRepoDone
			return m, nil
		}
		m.files = msg.files
		m.release = msg.release

		if len(m.files) == 0 {
			m.phase = importRepoDone
			return m, nil
		}

		// Build multiselect
		m.ms = buildImportRepoMultiselect(m.files)
		m.phase = importRepoFilePicking
		return m, func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		}

	case importRepoProgressMsg:
		return m.handleProgress(msg)

	case importRepoCommitCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		m.phase = importRepoDone
		return m, nil
	}

	// Forward non-key messages to sub-models
	switch m.phase {
	case importRepoSourceInput:
		var cmd tea.Cmd
		m.sourceInput, cmd = m.sourceInput.Update(msg)
		return m, cmd
	case importRepoFilePicking:
		var cmd tea.Cmd
		m.ms, cmd = m.ms.Update(msg)
		return m, cmd
	case importRepoShelfPicking:
		var cmd tea.Cmd
		m.shelfList, cmd = m.shelfList.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m ImportRepoModel) updateSourceInput(msg tea.KeyMsg) (ImportRepoModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
	case "enter":
		val := strings.TrimSpace(m.sourceInput.Value())
		parts := strings.SplitN(val, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			m.err = fmt.Errorf("enter source as owner/repo")
			return m, nil
		}
		m.srcOwner = parts[0]
		m.srcRepo = parts[1]
		m.err = nil

		// If shelf already resolved (single shelf), skip picker
		if m.destShelf != nil {
			m.phase = importRepoScanning
			return m, m.scanAsync()
		}

		// Show shelf picker
		m.shelfList = m.createShelfList()
		m.phase = importRepoShelfPicking
		return m, func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		}
	}

	var cmd tea.Cmd
	m.sourceInput, cmd = m.sourceInput.Update(msg)
	return m, cmd
}

func (m ImportRepoModel) updateShelfPicking(msg tea.KeyMsg) (ImportRepoModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.phase = importRepoSourceInput
		m.sourceInput.Focus()
		return m, textinput.Blink
	case "enter":
		if item, ok := m.shelfList.SelectedItem().(tui.ShelfOption); ok {
			m.resolveShelf(item.Name)
			m.phase = importRepoScanning
			return m, m.scanAsync()
		}
	}

	var cmd tea.Cmd
	m.shelfList, cmd = m.shelfList.Update(msg)
	return m, cmd
}

func (m ImportRepoModel) updateFilePicking(msg tea.KeyMsg) (ImportRepoModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
	case "enter":
		selected := collectSelectedRepoFiles(&m.ms)
		if len(selected) == 0 {
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		}
		m.toImport = selected
		ch := make(chan importRepoProgressMsg, 10)
		m.progressCh = ch
		m.phase = importRepoProcessing
		return m, tea.Batch(m.processAsync(ch), waitForImportRepoProgress(ch))
	case " ":
		m.ms.Toggle()
		return m, nil
	}

	var cmd tea.Cmd
	m.ms, cmd = m.ms.Update(msg)
	return m, cmd
}

func (m ImportRepoModel) handleProgress(msg importRepoProgressMsg) (ImportRepoModel, tea.Cmd) {
	switch msg.kind {
	case "status":
		m.statusMsg = fmt.Sprintf("Migrating %d/%d: %s", msg.current, msg.total, msg.path)
		return m, waitForImportRepoProgress(m.progressCh)
	case "done":
		if msg.err != nil {
			m.failCount++
		} else if msg.book != nil {
			m.importedBooks = append(m.importedBooks, *msg.book)
			m.successCount++
		}
		if msg.current >= msg.total {
			if m.successCount > 0 {
				m.phase = importRepoCommitting
				m.statusMsg = fmt.Sprintf("Committing %d file(s)...", m.successCount)
				return m, m.commitAsync()
			}
			m.phase = importRepoDone
			return m, nil
		}
		return m, waitForImportRepoProgress(m.progressCh)
	}
	return m, nil
}

func (m ImportRepoModel) createShelfList() list.Model {
	items := make([]list.Item, len(m.shelfOptions))
	for i := range m.shelfOptions {
		items[i] = m.shelfOptions[i]
	}
	d := delegate.New(func(w io.Writer, ml list.Model, index int, item list.Item) {
		opt, ok := item.(tui.ShelfOption)
		if !ok {
			return
		}
		cursor := "  "
		if index == ml.Index() {
			cursor = "> "
		}
		_, _ = fmt.Fprintf(w, "%s%s (%s)", cursor, opt.Name, opt.Repo)
	})
	l := list.New(items, d, 0, 0)
	l.Title = "Select destination shelf"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = tui.StyleHeader
	return l
}

func buildImportRepoMultiselect(files []migrate.FileEntry) multiselect.Model {
	items := make([]list.Item, len(files))
	for i := range files {
		items[i] = &importRepoFileItem{file: files[i]}
	}

	d := delegate.New(func(w io.Writer, ml list.Model, index int, item list.Item) {})
	l := list.New(items, d, 0, 0)
	l.Title = "Select files to import"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = tui.StyleHeader
	l.Styles.HelpStyle = tui.StyleHelp

	ms := multiselect.New(l)
	ms.SetTitle("Select files to import")

	// Set delegate with multiselect rendering
	rd := delegate.New(func(w io.Writer, ml list.Model, index int, item list.Item) {
		fi, ok := item.(*importRepoFileItem)
		if !ok {
			return
		}

		isCurrent := index == ml.Index()
		checkbox := ms.CheckboxPrefix(fi)

		sizeStr := humanBytes(int64(fi.file.Size))
		line := fmt.Sprintf("%s %s (%s)", checkbox, fi.file.Path, sizeStr)

		if isCurrent {
			sel := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
			_, _ = fmt.Fprint(w, sel.Render(line))
		} else {
			_, _ = fmt.Fprint(w, line)
		}
	})
	ms.List.SetDelegate(rd)

	toggleKey := key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle"))
	selectKey := key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "import"))
	ms.List.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{toggleKey, selectKey}
	}

	ms.RestoreSelectionState()
	return ms
}

func collectSelectedRepoFiles(ms *multiselect.Model) []migrate.FileEntry {
	var selected []migrate.FileEntry
	for _, item := range ms.List.Items() {
		if fi, ok := item.(*importRepoFileItem); ok && fi.sel {
			selected = append(selected, fi.file)
		}
	}
	if len(selected) == 0 {
		if fi, ok := ms.List.SelectedItem().(*importRepoFileItem); ok {
			selected = []migrate.FileEntry{fi.file}
		}
	}
	return selected
}

// fileBaseName returns the filename without extension
func fileBaseName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

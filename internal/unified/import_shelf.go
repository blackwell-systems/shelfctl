package unified

import (
	"fmt"
	"io"
	"strings"

	"github.com/blackwell-systems/bubbletea-multiselect"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/blackwell-systems/shelfctl/internal/tui/delegate"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// importShelfPhase tracks the current phase of the import-from-shelf workflow
type importShelfPhase int

const (
	importShelfSourceInput  importShelfPhase = iota // textinput for "owner/repo"
	importShelfShelfPicking                         // shelf list picker
	importShelfScanning                             // async: load catalogs, build dedup
	importShelfBookPicking                          // multiselect from source books
	importShelfProcessing                           // async: per-book download+upload
	importShelfCommitting                           // async: batch catalog commit + README
	importShelfDone                                 // summary
)

// Internal messages
type importShelfScanCompleteMsg struct {
	srcBooks     []catalog.Book
	dstBooks     []catalog.Book
	existingSHAs map[string]bool
	release      *github.Release
	err          error
}

type importShelfProgressMsg struct {
	kind    string // "status", "done"
	bookID  string
	book    *catalog.Book
	current int
	total   int
	err     error
}

type importShelfCommitCompleteMsg struct{ err error }

// importShelfBookItem wraps catalog.Book for multiselect
type importShelfBookItem struct {
	book   catalog.Book
	isDupe bool
	sel    bool
}

func (b *importShelfBookItem) FilterValue() string {
	return b.book.Title + " " + b.book.Author + " " + b.book.ID
}
func (b *importShelfBookItem) IsSelected() bool   { return b.sel }
func (b *importShelfBookItem) SetSelected(v bool) { b.sel = v }
func (b *importShelfBookItem) IsSelectable() bool { return !b.isDupe }

// ImportShelfModel is the unified view for importing from another shelfctl shelf
type ImportShelfModel struct {
	phase importShelfPhase

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
	srcBooks     []catalog.Book
	dstBooks     []catalog.Book
	existingSHAs map[string]bool
	release      *github.Release

	// Book picking
	ms multiselect.Model

	// Processing
	toImport   []catalog.Book
	progressCh <-chan importShelfProgressMsg
	statusMsg  string

	// Results
	importedBooks []catalog.Book
	successCount  int
	failCount     int
	dupeCount     int
}

// NewImportShelfModel creates a new import-from-shelf view
func NewImportShelfModel(gh *github.Client, cfg *config.Config) ImportShelfModel {
	if len(cfg.Shelves) == 0 {
		return ImportShelfModel{gh: gh, cfg: cfg, empty: true}
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

	m := ImportShelfModel{
		phase:        importShelfSourceInput,
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

func (m *ImportShelfModel) resolveShelf(name string) {
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
func (m ImportShelfModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m ImportShelfModel) Update(msg tea.Msg) (ImportShelfModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		switch m.phase {
		case importShelfBookPicking:
			h, v := tui.StyleBorder.GetFrameSize()
			m.ms.List.SetSize(msg.Width-h, msg.Height-v)
		case importShelfShelfPicking:
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
		if m.phase == importShelfDone {
			if msg.String() == "enter" || msg.String() == "esc" {
				return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
			}
			return m, nil
		}

		switch m.phase {
		case importShelfSourceInput:
			return m.updateSourceInput(msg)
		case importShelfShelfPicking:
			return m.updateShelfPicking(msg)
		case importShelfBookPicking:
			return m.updateBookPicking(msg)
		case importShelfProcessing, importShelfCommitting:
			if msg.String() == "ctrl+c" {
				return m, func() tea.Msg { return QuitAppMsg{} }
			}
			return m, nil
		}

	case importShelfScanCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
			m.phase = importShelfDone
			return m, nil
		}
		m.srcBooks = msg.srcBooks
		m.dstBooks = msg.dstBooks
		m.existingSHAs = msg.existingSHAs
		m.release = msg.release

		// Count importable books
		importable := 0
		for _, b := range m.srcBooks {
			if !m.existingSHAs[b.Checksum.SHA256] {
				importable++
			}
		}
		m.dupeCount = len(m.srcBooks) - importable

		if importable == 0 {
			m.phase = importShelfDone
			return m, nil
		}

		// Build multiselect
		m.ms = buildImportShelfMultiselect(m.srcBooks, m.existingSHAs)
		m.phase = importShelfBookPicking
		return m, func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		}

	case importShelfProgressMsg:
		return m.handleProgress(msg)

	case importShelfCommitCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		m.phase = importShelfDone
		return m, nil
	}

	// Forward non-key messages to sub-models
	switch m.phase {
	case importShelfSourceInput:
		var cmd tea.Cmd
		m.sourceInput, cmd = m.sourceInput.Update(msg)
		return m, cmd
	case importShelfBookPicking:
		var cmd tea.Cmd
		m.ms, cmd = m.ms.Update(msg)
		return m, cmd
	case importShelfShelfPicking:
		var cmd tea.Cmd
		m.shelfList, cmd = m.shelfList.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m ImportShelfModel) updateSourceInput(msg tea.KeyMsg) (ImportShelfModel, tea.Cmd) {
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
			m.phase = importShelfScanning
			return m, m.scanAsync()
		}

		// Show shelf picker
		m.shelfList = m.createShelfList()
		m.phase = importShelfShelfPicking
		return m, func() tea.Msg {
			return tea.WindowSizeMsg{Width: m.width, Height: m.height}
		}
	}

	var cmd tea.Cmd
	m.sourceInput, cmd = m.sourceInput.Update(msg)
	return m, cmd
}

func (m ImportShelfModel) updateShelfPicking(msg tea.KeyMsg) (ImportShelfModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.phase = importShelfSourceInput
		m.sourceInput.Focus()
		return m, textinput.Blink
	case "enter":
		if item, ok := m.shelfList.SelectedItem().(tui.ShelfOption); ok {
			m.resolveShelf(item.Name)
			m.phase = importShelfScanning
			return m, m.scanAsync()
		}
	}

	var cmd tea.Cmd
	m.shelfList, cmd = m.shelfList.Update(msg)
	return m, cmd
}

func (m ImportShelfModel) updateBookPicking(msg tea.KeyMsg) (ImportShelfModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
	case "enter":
		selected := collectSelectedShelfBooks(&m.ms)
		if len(selected) == 0 {
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		}
		m.toImport = selected
		ch := make(chan importShelfProgressMsg, 10)
		m.progressCh = ch
		m.phase = importShelfProcessing
		return m, tea.Batch(m.processAsync(ch), waitForImportShelfProgress(ch))
	case " ":
		m.ms.Toggle()
		return m, nil
	}

	var cmd tea.Cmd
	m.ms, cmd = m.ms.Update(msg)
	return m, cmd
}

func (m ImportShelfModel) handleProgress(msg importShelfProgressMsg) (ImportShelfModel, tea.Cmd) {
	switch msg.kind {
	case "status":
		m.statusMsg = fmt.Sprintf("Importing %d/%d: %s", msg.current, msg.total, msg.bookID)
		return m, waitForImportShelfProgress(m.progressCh)
	case "done":
		if msg.err != nil {
			m.failCount++
		} else if msg.book != nil {
			m.importedBooks = append(m.importedBooks, *msg.book)
			m.successCount++
		}
		if msg.current >= msg.total {
			if m.successCount > 0 {
				m.phase = importShelfCommitting
				m.statusMsg = fmt.Sprintf("Committing %d book(s)...", m.successCount)
				return m, m.commitAsync()
			}
			m.phase = importShelfDone
			return m, nil
		}
		return m, waitForImportShelfProgress(m.progressCh)
	}
	return m, nil
}

func (m ImportShelfModel) createShelfList() list.Model {
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
		fmt.Fprintf(w, "%s%s (%s)", cursor, opt.Name, opt.Repo)
	})
	l := list.New(items, d, 0, 0)
	l.Title = "Select destination shelf"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = tui.StyleHeader
	return l
}

func buildImportShelfMultiselect(srcBooks []catalog.Book, existingSHAs map[string]bool) multiselect.Model {
	items := make([]list.Item, len(srcBooks))
	for i := range srcBooks {
		items[i] = &importShelfBookItem{
			book:   srcBooks[i],
			isDupe: existingSHAs[srcBooks[i].Checksum.SHA256],
		}
	}

	d := delegate.New(func(w io.Writer, ml list.Model, index int, item list.Item) {})
	l := list.New(items, d, 0, 0)
	l.Title = "Select books to import"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = tui.StyleHeader
	l.Styles.HelpStyle = tui.StyleHelp

	ms := multiselect.New(l)
	ms.SetTitle("Select books to import")

	// Set delegate with multiselect rendering
	rd := delegate.New(func(w io.Writer, ml list.Model, index int, item list.Item) {
		bi, ok := item.(*importShelfBookItem)
		if !ok {
			return
		}

		isCurrent := index == ml.Index()
		checkbox := ms.CheckboxPrefix(bi)

		title := bi.book.Title
		author := bi.book.Author
		id := bi.book.ID

		line := fmt.Sprintf("%s %s by %s (%s)", checkbox, title, author, id)
		if bi.isDupe {
			line = fmt.Sprintf("  %s by %s (%s) [already imported]", title, author, id)
			dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			fmt.Fprint(w, dim.Render(line))
			return
		}

		if isCurrent {
			sel := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
			fmt.Fprint(w, sel.Render(line))
		} else {
			fmt.Fprint(w, line)
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

func collectSelectedShelfBooks(ms *multiselect.Model) []catalog.Book {
	var selected []catalog.Book
	for _, item := range ms.List.Items() {
		if bi, ok := item.(*importShelfBookItem); ok && bi.sel {
			selected = append(selected, bi.book)
		}
	}
	if len(selected) == 0 {
		if bi, ok := ms.List.SelectedItem().(*importShelfBookItem); ok && !bi.isDupe {
			selected = []catalog.Book{bi.book}
		}
	}
	return selected
}

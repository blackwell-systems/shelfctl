package unified

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/ingest"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// shelvePhase tracks the current phase of the shelve workflow
type shelvePhase int

const (
	shelveShelfPicking shelvePhase = iota // Shelf selection (skipped if single shelf)
	shelveFilePicking                     // Miller columns file browser
	shelveSetup                           // Loading catalog + ensuring release
	shelveIngesting                       // Ingesting current file (async)
	shelveForm                            // Metadata form for current file
	shelveUploading                       // Upload + dup check + collision + cache (async with progress)
	shelveCommitting                      // Batch commit catalog + README (async)
)

// Shelve form field indices
const (
	shelveFieldTitle  = 0
	shelveFieldAuthor = 1
	shelveFieldTags   = 2
	shelveFieldID     = 3
	shelveFieldCache  = 4 // Not a text input, handled separately
)

// ShelveCompleteMsg is emitted when the shelve workflow finishes
type ShelveCompleteMsg struct {
	SuccessCount int
	FailCount    int
}

// Internal messages
type shelveSetupCompleteMsg struct {
	existingBooks []catalog.Book
	release       *github.Release
	err           error
}

type shelveIngestCompleteMsg struct {
	tmpPath     string
	sha256      string
	size        int64
	format      string
	srcName     string
	pdfMetadata *ingest.PDFMetadata
	err         error
}

type shelveProcessingMsg struct {
	kind   string // "status", "progress", "done"
	status string
	bytes  int64
	total  int64
	book   *catalog.Book
	err    error
}

type shelveCommitCompleteMsg struct {
	err error
}

// ingestedFile holds result of file ingestion
type shelveIngestedFile struct {
	tmpPath     string
	sha256      string
	size        int64
	format      string
	srcName     string
	pdfMetadata *ingest.PDFMetadata
}

// ShelveModel is the unified view for adding books to the library
type ShelveModel struct {
	phase shelvePhase

	// Dependencies
	gh       *github.Client
	cfg      *config.Config
	cacheMgr *cache.Manager

	// Shelf selection
	shelfList    list.Model
	shelfOptions []tui.ShelfOption
	shelfName    string
	shelf        *config.ShelfConfig
	owner        string
	releaseTag   string
	catalogPath  string
	autoSelected bool // true if single shelf, skip picker

	// File selection
	filePicker    tui.FilePickerModel
	selectedFiles []string

	// Per-file processing state
	fileIndex int // Which file we're processing (0..len(selectedFiles)-1)
	ingested  *shelveIngestedFile

	// Metadata form (embedded text inputs)
	inputs       []textinput.Model
	focused      int
	cacheLocally bool
	formDefaults shelveFormDefaults

	// Upload progress
	uploadProgress float64
	uploadBytes    int64
	uploadTotal    int64
	uploadStatus   string
	statusCh       <-chan shelveProcessingMsg

	// Accumulated results
	existingBooks []catalog.Book
	newBooks      []catalog.Book
	release       *github.Release
	successCount  int
	failCount     int

	// General
	width     int
	height    int
	err       error
	empty     bool // No shelves configured
	activeCmd string
}

type shelveFormDefaults struct {
	filename string
	title    string
	author   string
	id       string
}

// NewShelveModel creates a new shelve (add book) view
func NewShelveModel(gh *github.Client, cfg *config.Config, cacheMgr *cache.Manager) ShelveModel {
	if len(cfg.Shelves) == 0 {
		return ShelveModel{
			gh:       gh,
			cfg:      cfg,
			cacheMgr: cacheMgr,
			empty:    true,
		}
	}

	// Build shelf options
	var options []tui.ShelfOption
	for _, s := range cfg.Shelves {
		options = append(options, tui.ShelfOption{
			Name: s.Name,
			Repo: s.Repo,
		})
	}

	m := ShelveModel{
		gh:           gh,
		cfg:          cfg,
		cacheMgr:     cacheMgr,
		shelfOptions: options,
		cacheLocally: true, // Default to caching
	}

	// Auto-select if only one shelf
	if len(options) == 1 {
		m.autoSelected = true
		m.shelfName = options[0].Name
		m.resolveShelf()

		// Skip to file picking
		m.phase = shelveFilePicking
		fp, err := m.createFilePicker()
		if err != nil {
			m.err = err
			return m
		}
		m.filePicker = fp
	} else {
		// Multiple shelves - show picker
		m.phase = shelveShelfPicking
		m.shelfList = m.createShelfList()
	}

	return m
}

func (m *ShelveModel) resolveShelf() {
	m.shelf = m.cfg.ShelfByName(m.shelfName)
	if m.shelf != nil {
		m.owner = m.shelf.EffectiveOwner(m.cfg.GitHub.Owner)
		m.releaseTag = m.shelf.EffectiveRelease(m.cfg.Defaults.Release)
		m.catalogPath = m.shelf.EffectiveCatalogPath()
	}
}

func (m ShelveModel) createShelfList() list.Model {
	items := make([]list.Item, len(m.shelfOptions))
	for i, s := range m.shelfOptions {
		items[i] = s
	}

	d := tui.NewShelfDelegate()
	l := list.New(items, d, 0, 0)
	l.Title = "Select Shelf"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = tui.StyleHeader
	l.Styles.HelpStyle = tui.StyleHelp

	return l
}

func (m ShelveModel) createFilePicker() (tui.FilePickerModel, error) {
	// Determine start path (same logic as app/shelve.go selectFile)
	startPath, err := os.Getwd()
	if err != nil {
		home := os.Getenv("HOME")
		startPath = filepath.Join(home, "Downloads")
		if _, err := os.Stat(startPath); err != nil {
			startPath = home
		}
	}

	return tui.NewFilePickerModel(startPath)
}

// Init initializes the shelve view
func (m ShelveModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the shelve view
func (m ShelveModel) Update(msg tea.Msg) (ShelveModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tui.ClearActiveCmdMsg:
		m.activeCmd = ""
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		switch m.phase {
		case shelveShelfPicking:
			h, v := tui.StyleBorder.GetFrameSize()
			m.shelfList.SetSize(msg.Width-h, msg.Height-v)
		case shelveFilePicking:
			// Forward to file picker
			var fpModel tea.Model
			fpModel, _ = m.filePicker.Update(msg)
			m.filePicker = fpModel.(tui.FilePickerModel)
		}
		return m, nil

	case tea.KeyMsg:
		if m.empty || (m.err != nil && m.phase != shelveForm) {
			switch msg.String() {
			case "enter", "esc", "q":
				return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
			}
			return m, nil
		}

		switch m.phase {
		case shelveShelfPicking:
			return m.updateShelfPicking(msg)
		case shelveFilePicking:
			return m.updateFilePicking(msg)
		case shelveForm:
			return m.updateForm(msg)
		case shelveSetup, shelveIngesting, shelveUploading, shelveCommitting:
			// Allow Ctrl+C during async operations
			if msg.String() == "ctrl+c" {
				return m, func() tea.Msg { return QuitAppMsg{} }
			}
			return m, nil
		}

	case shelveSetupCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		}
		m.existingBooks = msg.existingBooks
		m.release = msg.release
		m.fileIndex = 0
		m.phase = shelveIngesting
		return m, m.ingestCurrentFile()

	case shelveIngestCompleteMsg:
		if msg.err != nil {
			m.failCount++
			return m.advanceToNextFileOrCommit()
		}
		m.ingested = &shelveIngestedFile{
			tmpPath:     msg.tmpPath,
			sha256:      msg.sha256,
			size:        msg.size,
			format:      msg.format,
			srcName:     msg.srcName,
			pdfMetadata: msg.pdfMetadata,
		}
		// Initialize form with ingested metadata
		m.initFormForCurrentFile()
		m.phase = shelveForm
		return m, textinput.Blink

	case shelveProcessingMsg:
		return m.handleProcessingMsg(msg)

	case shelveCommitCompleteMsg:
		return m, func() tea.Msg {
			return NavigateMsg{Target: "hub"}
		}

	case ShelveCompleteMsg:
		return m, func() tea.Msg {
			return NavigateMsg{Target: "hub"}
		}
	}

	// Forward non-key messages to sub-models
	switch m.phase {
	case shelveShelfPicking:
		var cmd tea.Cmd
		m.shelfList, cmd = m.shelfList.Update(msg)
		return m, cmd
	case shelveFilePicking:
		var fpModel tea.Model
		var cmd tea.Cmd
		fpModel, cmd = m.filePicker.Update(msg)
		m.filePicker = fpModel.(tui.FilePickerModel)

		// Check if file picker completed
		if m.filePicker.IsQuitting() {
			if m.filePicker.Error() != nil {
				return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
			}
			m.selectedFiles = m.filePicker.SelectedFiles()
			if len(m.selectedFiles) == 0 {
				return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
			}
			m.phase = shelveSetup
			return m, m.setupAsync()
		}
		return m, cmd
	case shelveForm:
		// Forward to text inputs
		cmd := m.updateFormInputs(msg)
		return m, cmd
	}

	return m, nil
}

// --- Phase: Shelf Picking ---

func (m ShelveModel) updateShelfPicking(msg tea.KeyMsg) (ShelveModel, tea.Cmd) {
	if m.shelfList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.shelfList, cmd = m.shelfList.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }
	case "q", "esc":
		return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
	case "enter":
		if item, ok := m.shelfList.SelectedItem().(tui.ShelfOption); ok {
			m.shelfName = item.Name
			m.resolveShelf()

			// Transition to file picking
			m.phase = shelveFilePicking
			fp, err := m.createFilePicker()
			if err != nil {
				m.err = err
				return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
			}
			m.filePicker = fp
			return m, func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			}
		}
	}

	var cmd tea.Cmd
	m.shelfList, cmd = m.shelfList.Update(msg)
	return m, cmd
}

// --- Phase: File Picking ---

func (m ShelveModel) updateFilePicking(msg tea.KeyMsg) (ShelveModel, tea.Cmd) {
	var fpModel tea.Model
	var cmd tea.Cmd
	fpModel, cmd = m.filePicker.Update(msg)
	m.filePicker = fpModel.(tui.FilePickerModel)

	// Check if file picker completed
	if m.filePicker.IsQuitting() {
		if m.filePicker.Error() != nil {
			// User cancelled
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		}
		m.selectedFiles = m.filePicker.SelectedFiles()
		if len(m.selectedFiles) == 0 {
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		}

		// Transition to setup
		m.phase = shelveSetup
		return m, m.setupAsync()
	}

	return m, cmd
}

// --- Phase: Form ---
// (async ops in shelve_ops.go; rendering in shelve_render.go)

func (m *ShelveModel) initFormForCurrentFile() {
	ing := m.ingested

	// Compute defaults (same logic as app/shelve.go collectMetadata)
	defaultTitle := strings.TrimSuffix(ing.srcName, filepath.Ext(ing.srcName))
	defaultAuthor := ""

	if ing.pdfMetadata != nil {
		if ing.pdfMetadata.Title != "" {
			defaultTitle = ing.pdfMetadata.Title
		}
		if ing.pdfMetadata.Author != "" {
			defaultAuthor = ing.pdfMetadata.Author
		}
	}

	defaultID := slugify(defaultTitle)

	m.formDefaults = shelveFormDefaults{
		filename: ing.srcName,
		title:    defaultTitle,
		author:   defaultAuthor,
		id:       defaultID,
	}

	const inputWidth = 50
	const maxPlaceholderWidth = 45

	truncate := func(s string, max int) string {
		if len(s) > max {
			return s[:max-1] + "…"
		}
		return s
	}

	m.inputs = make([]textinput.Model, 4)
	m.focused = 0
	m.cacheLocally = true
	m.err = nil

	// Title
	m.inputs[shelveFieldTitle] = textinput.New()
	m.inputs[shelveFieldTitle].Placeholder = truncate(defaultTitle, maxPlaceholderWidth)
	m.inputs[shelveFieldTitle].Focus()
	m.inputs[shelveFieldTitle].CharLimit = 200
	m.inputs[shelveFieldTitle].Width = inputWidth
	m.inputs[shelveFieldTitle].Prompt = ""

	// Author
	m.inputs[shelveFieldAuthor] = textinput.New()
	if defaultAuthor != "" {
		m.inputs[shelveFieldAuthor].Placeholder = truncate(defaultAuthor, maxPlaceholderWidth)
	} else {
		m.inputs[shelveFieldAuthor].Placeholder = "Author name"
	}
	m.inputs[shelveFieldAuthor].CharLimit = 100
	m.inputs[shelveFieldAuthor].Width = inputWidth
	m.inputs[shelveFieldAuthor].Prompt = ""

	// Tags
	m.inputs[shelveFieldTags] = textinput.New()
	m.inputs[shelveFieldTags].Placeholder = "comma,separated,tags"
	m.inputs[shelveFieldTags].CharLimit = 200
	m.inputs[shelveFieldTags].Width = inputWidth
	m.inputs[shelveFieldTags].Prompt = ""

	// ID
	m.inputs[shelveFieldID] = textinput.New()
	m.inputs[shelveFieldID].Placeholder = truncate(defaultID, maxPlaceholderWidth)
	m.inputs[shelveFieldID].CharLimit = 63
	m.inputs[shelveFieldID].Width = inputWidth
	m.inputs[shelveFieldID].Prompt = ""
}

// getFormValue returns input value or default if empty (same as shelve_form.go getValue)
func (m ShelveModel) getFormValue(field int) string {
	val := m.inputs[field].Value()
	if val != "" {
		return val
	}
	switch field {
	case shelveFieldTitle:
		return m.formDefaults.title
	case shelveFieldAuthor:
		return m.formDefaults.author
	case shelveFieldID:
		return m.formDefaults.id
	default:
		return ""
	}
}

func (m ShelveModel) updateForm(msg tea.KeyMsg) (ShelveModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }

	case "esc":
		// Cancel form - clean up temp file and return to hub
		if m.ingested != nil {
			_ = os.Remove(m.ingested.tmpPath)
		}
		return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }

	case "enter":
		highlightCmd := tui.SetActiveCmd(&m.activeCmd, "enter")
		m, cmd := m.submitForm()
		return m, tea.Batch(cmd, highlightCmd)

	case " ":
		// Toggle cache checkbox if focused on it
		if m.focused == shelveFieldCache {
			m.cacheLocally = !m.cacheLocally
			return m, tui.SetActiveCmd(&m.activeCmd, "space")
		}

	case "tab", "down":
		// Move to next field (including cache checkbox)
		highlightCmd := tui.SetActiveCmd(&m.activeCmd, "tab")
		if m.focused < len(m.inputs) {
			m.inputs[m.focused].Blur()
		}
		m.focused = (m.focused + 1) % (shelveFieldCache + 1)
		if m.focused < len(m.inputs) {
			return m, tea.Batch(m.inputs[m.focused].Focus(), highlightCmd)
		}
		return m, highlightCmd

	case "shift+tab", "up":
		// Move to previous field
		highlightCmd := tui.SetActiveCmd(&m.activeCmd, "tab")
		if m.focused < len(m.inputs) {
			m.inputs[m.focused].Blur()
		}
		m.focused--
		if m.focused < 0 {
			m.focused = shelveFieldCache
		}
		if m.focused < len(m.inputs) {
			return m, tea.Batch(m.inputs[m.focused].Focus(), highlightCmd)
		}
		return m, highlightCmd
	}

	// Update focused text input
	if m.focused < len(m.inputs) {
		cmd := m.updateFormInputs(msg)
		return m, cmd
	}

	return m, nil
}

func (m *ShelveModel) updateFormInputs(msg tea.Msg) tea.Cmd {
	if m.focused >= len(m.inputs) {
		return nil
	}
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m ShelveModel) submitForm() (ShelveModel, tea.Cmd) {
	// Collect form values
	title := m.getFormValue(shelveFieldTitle)
	author := m.getFormValue(shelveFieldAuthor)
	tagsCSV := m.inputs[shelveFieldTags].Value()
	bookID := m.getFormValue(shelveFieldID)

	// Validate ID
	if !idRegexp.MatchString(bookID) {
		m.err = fmt.Errorf("invalid ID %q — must match ^[a-z0-9][a-z0-9-]{1,62}$", bookID)
		return m, nil
	}

	// Determine asset name
	assetName := ""
	if m.cfg.Defaults.AssetNaming == "original" {
		assetName = m.ingested.srcName
	}
	if assetName == "" {
		assetName = bookID + "." + m.ingested.format
	}

	// Parse tags
	var tags []string
	if tagsCSV != "" {
		for _, t := range strings.Split(tagsCSV, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	// Store form results in model for the processing phase
	m.uploadStatus = "Preparing..."
	m.uploadBytes = 0
	m.uploadTotal = m.ingested.size
	m.uploadProgress = 0

	// Create processing channel
	ch := make(chan shelveProcessingMsg, 50)
	m.statusCh = ch
	m.phase = shelveUploading

	// Start async processing
	return m, m.processFileAsync(ch, bookID, title, author, tags, assetName)
}

// --- Helpers ---

// slugify converts a title to a lowercase, hyphenated ID candidate.
// Apostrophes and quotes are removed. Other non-alphanumeric characters
// collapse into a single hyphen.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevWasSep := false
	for _, r := range s {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		isQuote := r == '\'' || r == '"' ||
			r == '\u2018' || r == '\u2019' ||
			r == '\u201C' || r == '\u201D'

		if isAlnum {
			b.WriteRune(r)
			prevWasSep = false
		} else if isQuote {
			continue
		} else {
			if !prevWasSep && b.Len() > 0 {
				b.WriteRune('-')
			}
			prevWasSep = true
		}
	}
	result := strings.TrimRight(b.String(), "-")
	if len(result) > 63 {
		result = result[:63]
		result = strings.TrimRight(result, "-")
	}
	if result == "" {
		return "book"
	}
	return result
}

// idRegexp validates book IDs
var idRegexp = mustCompileRegexp(`^[a-z0-9][a-z0-9-]{1,62}$`)

func mustCompileRegexp(pattern string) interface{ MatchString(string) bool } {
	return regexpWrapper{pattern}
}

type regexpWrapper struct {
	pattern string
}

func (r regexpWrapper) MatchString(s string) bool {
	if len(s) < 2 || len(s) > 63 {
		return false
	}
	// First char must be [a-z0-9]
	first := s[0]
	if (first < 'a' || first > 'z') && (first < '0' || first > '9') {
		return false
	}
	// Rest must be [a-z0-9-]
	for i := 1; i < len(s); i++ {
		c := s[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}
	return true
}

package unified

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/ingest"
	"github.com/blackwell-systems/shelfctl/internal/readme"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// --- Phase: Setup (async) ---

func (m ShelveModel) setupAsync() tea.Cmd {
	owner := m.owner
	repo := m.shelf.Repo
	catalogPath := m.catalogPath
	releaseTag := m.releaseTag
	gh := m.gh

	return func() tea.Msg {
		// Load existing catalog
		catalogMgr := catalog.NewManager(gh, owner, repo, catalogPath)
		existingBooks, err := catalogMgr.Load()
		if err != nil {
			return shelveSetupCompleteMsg{err: fmt.Errorf("loading catalog: %w", err)}
		}

		// Ensure release exists
		rel, err := gh.EnsureRelease(owner, repo, releaseTag)
		if err != nil {
			return shelveSetupCompleteMsg{err: fmt.Errorf("ensuring release: %w", err)}
		}

		return shelveSetupCompleteMsg{
			existingBooks: existingBooks,
			release:       rel,
		}
	}
}

// --- Phase: Ingesting (async) ---

func (m ShelveModel) ingestCurrentFile() tea.Cmd {
	filePath := m.selectedFiles[m.fileIndex]
	token := m.cfg.GitHub.Token
	apiBase := m.cfg.GitHub.APIBase

	return func() tea.Msg {
		// Resolve source
		src, err := ingest.Resolve(filePath, token, apiBase)
		if err != nil {
			return shelveIngestCompleteMsg{err: fmt.Errorf("resolve %s: %w", filePath, err)}
		}

		// Open and stream to temp file
		rc, err := src.Open()
		if err != nil {
			return shelveIngestCompleteMsg{err: fmt.Errorf("open: %w", err)}
		}

		tmp, err := os.CreateTemp("", "shelfctl-add-*")
		if err != nil {
			_ = rc.Close()
			return shelveIngestCompleteMsg{err: err}
		}
		tmpPath := tmp.Name()

		hr := ingest.NewReader(rc)
		if _, err := io.Copy(tmp, hr); err != nil {
			_ = tmp.Close()
			_ = rc.Close()
			_ = os.Remove(tmpPath)
			return shelveIngestCompleteMsg{err: fmt.Errorf("buffering: %w", err)}
		}
		_ = tmp.Close()
		_ = rc.Close()

		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(src.Name), "."))

		result := shelveIngestCompleteMsg{
			tmpPath: tmpPath,
			sha256:  hr.SHA256(),
			size:    hr.Size(),
			format:  ext,
			srcName: src.Name,
		}

		// Extract PDF metadata if applicable
		if ext == "pdf" {
			if pdfMeta, err := ingest.ExtractPDFMetadata(tmpPath); err == nil {
				result.pdfMetadata = pdfMeta
			}
		}

		return result
	}
}

// --- Phase: Form ---

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

// --- Phase: Uploading / Processing (async with progress) ---

func (m ShelveModel) processFileAsync(statusCh chan shelveProcessingMsg, bookID, title, author string, tags []string, assetName string) tea.Cmd {
	existingBooks := m.existingBooks
	sha256 := m.ingested.sha256
	tmpPath := m.ingested.tmpPath
	size := m.ingested.size
	format := m.ingested.format
	doCache := m.cacheLocally
	owner := m.owner
	repo := m.shelf.Repo
	releaseID := m.release.ID
	releaseTag := m.releaseTag
	gh := m.gh
	cacheMgr := m.cacheMgr

	go func() {
		defer close(statusCh)

		// 1. Check duplicates
		statusCh <- shelveProcessingMsg{kind: "status", status: "Checking for duplicates..."}
		for _, b := range existingBooks {
			if b.Checksum.SHA256 == sha256 {
				statusCh <- shelveProcessingMsg{
					kind: "done",
					err:  fmt.Errorf("duplicate: file already exists as %q (%s)", b.ID, b.Title),
				}
				return
			}
		}

		// 2. Check asset collision
		statusCh <- shelveProcessingMsg{kind: "status", status: "Checking for conflicts..."}
		existingAsset, err := gh.FindAsset(owner, repo, releaseID, assetName)
		if err != nil {
			statusCh <- shelveProcessingMsg{kind: "done", err: fmt.Errorf("checking assets: %w", err)}
			return
		}
		if existingAsset != nil {
			statusCh <- shelveProcessingMsg{
				kind: "done",
				err:  fmt.Errorf("asset %q already exists in release %s", assetName, releaseTag),
			}
			return
		}

		// 3. Upload with progress
		statusCh <- shelveProcessingMsg{kind: "status", status: "Uploading..."}

		uploadFile, err := os.Open(tmpPath)
		if err != nil {
			statusCh <- shelveProcessingMsg{kind: "done", err: err}
			return
		}

		progressCh := make(chan int64, 50)
		uploadErrCh := make(chan error, 1)

		go func() {
			defer uploadFile.Close() //nolint:errcheck
			progressCh <- 0
			pr := tui.NewProgressReader(uploadFile, size, progressCh)
			_, uploadErr := gh.UploadAsset(owner, repo, releaseID, assetName, pr, size, "application/octet-stream")
			close(progressCh)
			uploadErrCh <- uploadErr
		}()

		// Forward progress to status channel
		for bytes := range progressCh {
			statusCh <- shelveProcessingMsg{kind: "progress", bytes: bytes, total: size}
		}

		if err := <-uploadErrCh; err != nil {
			statusCh <- shelveProcessingMsg{kind: "done", err: fmt.Errorf("upload: %w", err)}
			return
		}

		// 4. Build catalog entry
		book := catalog.Book{
			ID:        bookID,
			Title:     title,
			Author:    author,
			Tags:      tags,
			Format:    format,
			SizeBytes: size,
			Checksum:  catalog.Checksum{SHA256: sha256},
			Source: catalog.Source{
				Type:    "github_release",
				Owner:   owner,
				Repo:    repo,
				Release: releaseTag,
				Asset:   assetName,
			},
			Meta: catalog.Meta{
				AddedAt: time.Now().UTC().Format(time.RFC3339),
			},
		}

		// 5. Cache locally if requested
		if doCache {
			statusCh <- shelveProcessingMsg{kind: "status", status: "Caching locally..."}
			f, err := os.Open(tmpPath)
			if err == nil {
				_, _ = cacheMgr.Store(owner, repo, bookID, assetName, f, sha256)
				_ = f.Close()
			}
		}

		statusCh <- shelveProcessingMsg{kind: "done", book: &book}
	}()

	return waitForProcessingMsg(statusCh)
}

func waitForProcessingMsg(ch <-chan shelveProcessingMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return shelveProcessingMsg{kind: "done", err: fmt.Errorf("processing channel closed unexpectedly")}
		}
		return msg
	}
}

func (m ShelveModel) handleProcessingMsg(msg shelveProcessingMsg) (ShelveModel, tea.Cmd) {
	switch msg.kind {
	case "status":
		m.uploadStatus = msg.status
		return m, waitForProcessingMsg(m.statusCh)

	case "progress":
		m.uploadBytes = msg.bytes
		m.uploadTotal = msg.total
		if msg.total > 0 {
			m.uploadProgress = float64(msg.bytes) / float64(msg.total)
		}
		return m, waitForProcessingMsg(m.statusCh)

	case "done":
		// Clean up temp file
		if m.ingested != nil {
			_ = os.Remove(m.ingested.tmpPath)
			m.ingested = nil
		}

		if msg.err != nil {
			m.failCount++
		} else if msg.book != nil {
			m.newBooks = append(m.newBooks, *msg.book)
			m.existingBooks = catalog.Append(m.existingBooks, *msg.book)
			m.successCount++
		}

		return m.advanceToNextFileOrCommit()
	}

	return m, nil
}

func (m ShelveModel) advanceToNextFileOrCommit() (ShelveModel, tea.Cmd) {
	m.fileIndex++

	if m.fileIndex < len(m.selectedFiles) {
		// More files to process
		m.phase = shelveIngesting
		m.uploadStatus = ""
		m.uploadProgress = 0
		m.uploadBytes = 0
		return m, m.ingestCurrentFile()
	}

	// All files processed
	if m.successCount == 0 {
		// Nothing to commit
		return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
	}

	// Commit
	m.phase = shelveCommitting
	return m, m.commitAsync()
}

// --- Phase: Committing (async) ---

func (m ShelveModel) commitAsync() tea.Cmd {
	existingBooks := m.existingBooks
	newBooks := m.newBooks
	owner := m.owner
	repo := m.shelf.Repo
	catalogPath := m.catalogPath
	gh := m.gh

	return func() tea.Msg {
		// Create commit message
		var msg string
		if len(newBooks) == 1 {
			msg = fmt.Sprintf("add: %s — %s", newBooks[0].ID, newBooks[0].Title)
		} else {
			msg = fmt.Sprintf("add: %d books", len(newBooks))
		}

		// Save catalog
		catalogMgr := catalog.NewManager(gh, owner, repo, catalogPath)
		if err := catalogMgr.Save(existingBooks, msg); err != nil {
			return shelveCommitCompleteMsg{err: err}
		}

		// Update README
		readmeMgr := readme.NewUpdater(gh, owner, repo)
		_ = readmeMgr.UpdateWithStats(len(existingBooks), newBooks)

		return shelveCommitCompleteMsg{}
	}
}

// --- View ---

func (m ShelveModel) View() string {
	if m.empty {
		return m.renderMessage("No shelves configured", "Run 'shelfctl init' first\n\nPress Enter to return to menu")
	}

	if m.err != nil && m.phase != shelveForm {
		return m.renderMessage(fmt.Sprintf("Error: %v", m.err), "Press Enter to return to menu")
	}

	switch m.phase {
	case shelveShelfPicking:
		return m.renderShelfPicker()

	case shelveFilePicking:
		return m.filePicker.View()

	case shelveSetup:
		return m.renderMessage("Preparing...", fmt.Sprintf("Loading catalog and release for shelf %q", m.shelfName))

	case shelveIngesting:
		filename := filepath.Base(m.selectedFiles[m.fileIndex])
		if len(m.selectedFiles) > 1 {
			return m.renderMessage(
				fmt.Sprintf("Ingesting file [%d/%d]", m.fileIndex+1, len(m.selectedFiles)),
				fmt.Sprintf("Processing %s...", filename),
			)
		}
		return m.renderMessage("Ingesting file", fmt.Sprintf("Processing %s...", filename))

	case shelveForm:
		return m.renderForm()

	case shelveUploading:
		return m.renderUploadProgress()

	case shelveCommitting:
		return m.renderMessage("Committing changes...", fmt.Sprintf("Added %d book(s) to shelf %q", m.successCount, m.shelfName))

	default:
		return ""
	}
}

func (m ShelveModel) renderMessage(title, help string) string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render(title))
	b.WriteString("\n\n")
	b.WriteString(tui.StyleHelp.Render(help))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

func (m ShelveModel) renderShelfPicker() string {
	return tui.StyleBorder.Render(m.shelfList.View())
}

func (m ShelveModel) renderForm() string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder

	// Header
	b.WriteString(tui.StyleHeader.Render("Add Book to Library"))
	b.WriteString("\n\n")

	// File info with progress for multi-file
	filename := m.formDefaults.filename
	if len(m.selectedFiles) > 1 {
		filename = fmt.Sprintf("[%d/%d] %s", m.fileIndex+1, len(m.selectedFiles), filename)
	}
	const maxFilenameWidth = 60
	if len(filename) > maxFilenameWidth {
		filename = filename[:maxFilenameWidth-1] + "…"
	}
	b.WriteString(tui.StyleHelp.Render(fmt.Sprintf("File: %s", filename)))
	b.WriteString("\n")
	b.WriteString(tui.StyleHelp.Render(fmt.Sprintf("Shelf: %s", m.shelfName)))
	b.WriteString("\n\n")

	// Error display
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	// Form fields
	fields := []string{"Title", "Author", "Tags", "ID"}
	for i, label := range fields {
		if i == m.focused {
			b.WriteString(tui.StyleHighlight.Render("> " + label + ":"))
		} else {
			b.WriteString(tui.StyleNormal.Render("  " + label + ":"))
		}
		b.WriteString("\n  ")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}

	// Cache checkbox
	checkboxLabel := "Cache locally"
	checkbox := "[ ]"
	if m.cacheLocally {
		checkbox = "[✓]"
	}
	if m.focused == shelveFieldCache {
		b.WriteString(tui.StyleHighlight.Render("> " + checkboxLabel + ": " + checkbox))
	} else {
		b.WriteString(tui.StyleNormal.Render("  " + checkboxLabel + ": " + checkbox))
	}
	b.WriteString("\n")
	b.WriteString(tui.StyleHelp.Render("  (Space to toggle)"))
	b.WriteString("\n")

	// Help text
	b.WriteString("\n")
	b.WriteString(tui.RenderFooterBar([]tui.ShortcutEntry{
		{Key: "tab", Label: "Tab/↑↓ Navigate"},
		{Key: "space", Label: "Space Toggle"},
		{Key: "enter", Label: "Enter Submit"},
		{Key: "", Label: "Esc Cancel"},
	}, m.activeCmd))
	b.WriteString("\n")

	content := b.String()

	innerPadding := lipgloss.NewStyle().
		Padding(0, 2, 0, 1)

	return style.Render(tui.StyleBorder.Render(innerPadding.Render(content)))
}

func (m ShelveModel) renderUploadProgress() string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder

	// Header
	filename := filepath.Base(m.selectedFiles[m.fileIndex])
	if len(m.selectedFiles) > 1 {
		b.WriteString(tui.StyleHeader.Render(fmt.Sprintf("Processing [%d/%d]: %s", m.fileIndex+1, len(m.selectedFiles), filename)))
	} else {
		b.WriteString(tui.StyleHeader.Render(fmt.Sprintf("Processing: %s", filename)))
	}
	b.WriteString("\n\n")

	// Status
	b.WriteString(tui.StyleNormal.Render(m.uploadStatus))
	b.WriteString("\n\n")

	// Progress bar (only during actual upload)
	if m.uploadTotal > 0 && m.uploadStatus == "Uploading..." {
		// Simple text-based progress bar
		barWidth := 40
		filled := int(m.uploadProgress * float64(barWidth))
		if filled > barWidth {
			filled = barWidth
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		progressStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
		b.WriteString(progressStyle.Render(bar))
		b.WriteString("\n")

		// Bytes display
		currentMB := float64(m.uploadBytes) / 1024 / 1024
		totalMB := float64(m.uploadTotal) / 1024 / 1024
		b.WriteString(tui.StyleHelp.Render(fmt.Sprintf("%.2f MB / %.2f MB (%.0f%%)", currentMB, totalMB, m.uploadProgress*100)))
		b.WriteString("\n")
	}

	// Summary so far (for multi-file)
	if len(m.selectedFiles) > 1 && (m.successCount > 0 || m.failCount > 0) {
		b.WriteString("\n")
		if m.successCount > 0 {
			b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("Completed: %d", m.successCount)))
			b.WriteString("\n")
		}
		if m.failCount > 0 {
			errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			b.WriteString(errStyle.Render(fmt.Sprintf("Failed: %d", m.failCount)))
			b.WriteString("\n")
		}
	}

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
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

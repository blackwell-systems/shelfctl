package unified

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/blackwell-systems/bubbletea-components/multiselect"
	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/operations"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// editBookPhase tracks the current phase of the edit workflow
type editBookPhase int

const (
	editBookPicking    editBookPhase = iota // Showing book picker
	editBookEditing                         // Showing edit form for current book
	editBookProcessing                      // Committing changes
)

const (
	editFieldTitle  = 0
	editFieldAuthor = 1
	editFieldYear   = 2
	editFieldTags   = 3
)

// EditBookCompleteMsg is emitted when editing finishes
type EditBookCompleteMsg struct {
	SuccessCount int
	FailCount    int
}

// editedBook holds the result of editing a single book
type editedBook struct {
	item    tui.BookItem
	updated catalog.Book
}

// editFormState persists form input values per card in multi-edit sessions
type editFormState struct {
	title  string
	author string
	year   string
	tags   string
	saved  bool // true once user has confirmed this card via Enter
}

// EditBookModel is the unified view for editing book metadata
type EditBookModel struct {
	phase    editBookPhase
	ms       multiselect.Model
	gh       *github.Client
	cfg      *config.Config
	cacheMgr *cache.Manager
	width    int
	height   int
	err      error
	empty    bool

	// Selected books to edit
	toEdit []tui.BookItem

	// Editing phase state
	editIndex  int // Which book we're currently editing (0..len(toEdit)-1)
	inputs     []textinput.Model
	focused    int
	confirming bool
	formErr    error
	edits      []editedBook // Accumulated edits

	// Per-card form state (multi-book edit)
	formStates []editFormState

	// Full-screen carousel sub-view
	inCarousel     bool
	carouselCursor int // highlighted card index while browsing carousel

	// Results
	successCount int
	failCount    int

	// Navigation: where to go when done (default "hub")
	returnTo string

	// Footer highlight
	activeCmd string
}

// NewEditBookModel creates a new edit-book view
func NewEditBookModel(books []tui.BookItem, gh *github.Client, cfg *config.Config, cacheMgr *cache.Manager) EditBookModel {
	if len(books) == 0 {
		return EditBookModel{
			gh:       gh,
			cfg:      cfg,
			cacheMgr: cacheMgr,
			empty:    true,
			returnTo: "hub",
		}
	}

	ms, err := tui.NewBookPickerMultiModel(books, "Select books to edit")
	if err != nil {
		return EditBookModel{
			gh:       gh,
			cfg:      cfg,
			cacheMgr: cacheMgr,
			err:      err,
			returnTo: "hub",
		}
	}

	return EditBookModel{
		phase:    editBookPicking,
		ms:       ms,
		gh:       gh,
		cfg:      cfg,
		cacheMgr: cacheMgr,
		returnTo: "hub",
	}
}

// NewEditBookModelSingle creates an edit-book view that skips the picker
// and goes straight to editing a single book. Returns to returnTo when done.
func NewEditBookModelSingle(item *tui.BookItem, gh *github.Client, cfg *config.Config, cacheMgr *cache.Manager, returnTo string) EditBookModel {
	if item == nil {
		return EditBookModel{
			gh:       gh,
			cfg:      cfg,
			cacheMgr: cacheMgr,
			empty:    true,
			returnTo: returnTo,
		}
	}

	m := EditBookModel{
		phase:    editBookEditing,
		gh:       gh,
		cfg:      cfg,
		cacheMgr: cacheMgr,
		toEdit:   []tui.BookItem{*item},
		returnTo: returnTo,
	}
	m.formStates = initFormStates(m.toEdit)
	m.initFormForBook(0)
	return m
}

// initFormStates builds per-card form state initialised from book data
func initFormStates(books []tui.BookItem) []editFormState {
	states := make([]editFormState, len(books))
	for i, book := range books {
		yearStr := ""
		if book.Book.Year > 0 {
			yearStr = strconv.Itoa(book.Book.Year)
		}
		states[i] = editFormState{
			title:  book.Book.Title,
			author: book.Book.Author,
			year:   yearStr,
			tags:   strings.Join(book.Book.Tags, ","),
		}
	}
	return states
}

// Init initializes the edit-book view
func (m EditBookModel) Init() tea.Cmd {
	if m.phase == editBookEditing {
		return textinput.Blink
	}
	return nil
}

// Update handles messages for the edit-book view
func (m EditBookModel) Update(msg tea.Msg) (EditBookModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.phase == editBookPicking {
			h, v := tui.StyleBorder.GetFrameSize()
			m.ms.List.SetSize(msg.Width-h, msg.Height-v)
			m.ms.List.Title = tui.StyleHeader.Render("Select books to edit") + "\n" + tui.RenderColumnHeader(m.ms.List.Width())
			m.ms.List.Styles.Title = lipgloss.NewStyle()
		}
		return m, nil

	case tui.ClearActiveCmdMsg:
		m.activeCmd = ""
		return m, nil

	case tea.KeyMsg:
		if m.empty || (m.err != nil && m.phase == editBookPicking) {
			switch msg.String() {
			case "enter", "esc", "q":
				return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
			}
			return m, nil
		}

		switch m.phase {
		case editBookPicking:
			return m.updatePicking(msg)
		case editBookEditing:
			if m.inCarousel {
				return m.updateCarouselView(msg)
			}
			return m.updateEditing(msg)
		case editBookProcessing:
			return m, nil
		}

	case EditBookCompleteMsg:
		m.successCount = msg.SuccessCount
		m.failCount = msg.FailCount
		return m, func() tea.Msg {
			return NavigateMsg{Target: "hub"}
		}
	}

	// Forward to picker in picking phase
	if m.phase == editBookPicking {
		var cmd tea.Cmd
		m.ms, cmd = m.ms.Update(msg)
		return m, cmd
	}

	// Forward to text inputs in editing phase (not when in carousel or confirming)
	if m.phase == editBookEditing && !m.confirming && !m.inCarousel {
		cmd := m.updateInputs(msg)
		return m, cmd
	}

	return m, nil
}

func (m EditBookModel) updatePicking(msg tea.KeyMsg) (EditBookModel, tea.Cmd) {
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
		return m, tui.SetActiveCmd(&m.activeCmd, "space")

	case "enter":
		selected := tui.CollectSelectedBooks(&m.ms)
		if len(selected) == 0 {
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		}

		m.toEdit = selected
		m.editIndex = 0
		m.edits = nil
		m.formStates = initFormStates(selected)
		m.initFormForBook(0)
		m.phase = editBookEditing
		return m, tea.Batch(textinput.Blink, tui.SetActiveCmd(&m.activeCmd, "enter"))
	}

	var cmd tea.Cmd
	m.ms, cmd = m.ms.Update(msg)
	return m, cmd
}

func (m *EditBookModel) initFormForBook(index int) {
	fs := m.formStates[index]

	m.inputs = make([]textinput.Model, 4)
	m.focused = 0
	m.confirming = false
	m.formErr = nil

	m.inputs[editFieldTitle] = textinput.New()
	m.inputs[editFieldTitle].Placeholder = fs.title
	m.inputs[editFieldTitle].SetValue(fs.title)
	m.inputs[editFieldTitle].Focus()
	m.inputs[editFieldTitle].CharLimit = 200
	m.inputs[editFieldTitle].Width = 50

	m.inputs[editFieldAuthor] = textinput.New()
	m.inputs[editFieldAuthor].Placeholder = "Author name"
	m.inputs[editFieldAuthor].SetValue(fs.author)
	m.inputs[editFieldAuthor].CharLimit = 100
	m.inputs[editFieldAuthor].Width = 50

	m.inputs[editFieldYear] = textinput.New()
	m.inputs[editFieldYear].Placeholder = "Publication year (e.g., 2023)"
	m.inputs[editFieldYear].SetValue(fs.year)
	m.inputs[editFieldYear].CharLimit = 4
	m.inputs[editFieldYear].Width = 50

	m.inputs[editFieldTags] = textinput.New()
	m.inputs[editFieldTags].Placeholder = "comma,separated,tags"
	m.inputs[editFieldTags].SetValue(fs.tags)
	m.inputs[editFieldTags].CharLimit = 200
	m.inputs[editFieldTags].Width = 50
}

// saveCurrentFormToState copies current input values back into formStates
func (m *EditBookModel) saveCurrentFormToState() {
	if m.formStates == nil || m.editIndex >= len(m.formStates) {
		return
	}
	m.formStates[m.editIndex].title = m.inputs[editFieldTitle].Value()
	m.formStates[m.editIndex].author = m.inputs[editFieldAuthor].Value()
	m.formStates[m.editIndex].year = m.inputs[editFieldYear].Value()
	m.formStates[m.editIndex].tags = m.inputs[editFieldTags].Value()
}

func (m EditBookModel) updateEditing(msg tea.KeyMsg) (EditBookModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }

	case "esc":
		if m.confirming {
			m.confirming = false
			return m, nil
		}
		return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }

	case "enter":
		highlightCmd := tui.SetActiveCmd(&m.activeCmd, "enter")
		if m.confirming {
			m, cmd := m.submitCurrentBook()
			return m, tea.Batch(cmd, highlightCmd)
		}
		m.confirming = true
		return m, highlightCmd

	case "y", "Y":
		if m.confirming {
			highlightCmd := tui.SetActiveCmd(&m.activeCmd, "y")
			m, cmd := m.submitCurrentBook()
			return m, tea.Batch(cmd, highlightCmd)
		}

	case "n", "N":
		if m.confirming {
			highlightCmd := tui.SetActiveCmd(&m.activeCmd, "n")
			m, cmd := m.advanceToNextBook()
			return m, tea.Batch(cmd, highlightCmd)
		}

	case "tab", "shift+tab", "up", "down":
		if m.confirming {
			return m, nil
		}

		highlightCmd := tui.SetActiveCmd(&m.activeCmd, "tab")

		if msg.String() == "up" || msg.String() == "shift+tab" {
			// Up from the first field in multi-book edit: open carousel
			if msg.String() == "up" && m.focused == 0 && len(m.toEdit) > 1 {
				m.saveCurrentFormToState()
				m.carouselCursor = m.editIndex
				m.inCarousel = true
				return m, highlightCmd
			}
			m.focused--
		} else {
			m.focused++
		}

		if m.focused < 0 {
			m.focused = len(m.inputs) - 1
		} else if m.focused >= len(m.inputs) {
			m.focused = 0
		}

		cmds := make([]tea.Cmd, len(m.inputs))
		for i := range m.inputs {
			if i == m.focused {
				cmds[i] = m.inputs[i].Focus()
			} else {
				m.inputs[i].Blur()
			}
		}
		return m, tea.Batch(append(cmds, highlightCmd)...)
	}

	if !m.confirming {
		cmd := m.updateInputs(msg)
		return m, cmd
	}

	return m, nil
}

// updateCarouselView handles key events in the full-screen carousel
func (m EditBookModel) updateCarouselView(msg tea.KeyMsg) (EditBookModel, tea.Cmd) {
	cols := m.carouselColumns()
	n := len(m.toEdit)

	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }

	case "esc":
		// Return to form without changing selection
		m.inCarousel = false
		return m, textinput.Blink

	case "left", "h":
		if m.carouselCursor > 0 {
			m.carouselCursor--
		}

	case "right", "l":
		if m.carouselCursor < n-1 {
			m.carouselCursor++
		}

	case "up", "k":
		if m.carouselCursor-cols >= 0 {
			m.carouselCursor -= cols
		}

	case "down", "j":
		if m.carouselCursor+cols < n {
			m.carouselCursor += cols
		}

	case "enter", " ":
		// Select this card, drop back into the form
		m.editIndex = m.carouselCursor
		m.inCarousel = false
		m.initFormForBook(m.editIndex)
		return m, textinput.Blink
	}

	return m, nil
}

// carouselColumns returns the number of columns that fit the terminal width
func (m EditBookModel) carouselColumns() int {
	const minCardWidth = 28 // minimum usable card width
	usable := m.width - 4   // outer padding
	cols := usable / minCardWidth
	if cols < 1 {
		cols = 1
	}
	if cols > 4 {
		cols = 4
	}
	if cols > len(m.toEdit) {
		cols = len(m.toEdit)
	}
	return cols
}

func (m EditBookModel) submitCurrentBook() (EditBookModel, tea.Cmd) {
	yearVal := 0
	if yearStr := m.inputs[editFieldYear].Value(); yearStr != "" {
		year, err := strconv.Atoi(yearStr)
		if err != nil || year < 0 || year > 9999 {
			m.formErr = fmt.Errorf("invalid year (must be 0-9999)")
			m.confirming = false
			return m, nil
		}
		yearVal = year
	}

	var tags []string
	if tagStr := m.inputs[editFieldTags].Value(); tagStr != "" {
		for _, t := range strings.Split(tagStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	item := m.toEdit[m.editIndex]
	updated := item.Book
	updated.Title = m.inputs[editFieldTitle].Value()
	updated.Author = m.inputs[editFieldAuthor].Value()
	updated.Year = yearVal
	updated.Tags = tags

	m.edits = append(m.edits, editedBook{
		item:    item,
		updated: updated,
	})

	if m.editIndex < len(m.formStates) {
		m.formStates[m.editIndex].saved = true
	}

	return m.advanceToNextBook()
}

func (m EditBookModel) advanceToNextBook() (EditBookModel, tea.Cmd) {
	// Single-book path: commit immediately
	if len(m.toEdit) == 1 {
		if len(m.edits) == 0 {
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		}
		m.phase = editBookProcessing
		return m, m.commitEditsAsync()
	}

	// Multi-book: find next unsaved card (wrapping forward)
	n := len(m.toEdit)
	next := -1
	for i := 1; i <= n; i++ {
		candidate := (m.editIndex + i) % n
		if candidate < len(m.formStates) && !m.formStates[candidate].saved {
			next = candidate
			break
		}
	}

	// All cards saved — commit
	if next == -1 {
		if len(m.edits) == 0 {
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		}
		m.phase = editBookProcessing
		return m, m.commitEditsAsync()
	}

	// Load the next unsaved card
	m.editIndex = next
	m.initFormForBook(next)
	return m, textinput.Blink
}

func (m *EditBookModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

// View renders the edit-book view
func (m EditBookModel) View() string {
	if m.empty {
		return m.renderMessage("No books in library", "Press Enter to return to menu")
	}

	if m.err != nil && m.phase == editBookPicking {
		return m.renderMessage(fmt.Sprintf("Error: %v", m.err), "Press Enter to return to menu")
	}

	switch m.phase {
	case editBookPicking:
		return tui.StyleBorder.Render(m.ms.View())

	case editBookEditing:
		if m.inCarousel {
			return m.renderCarouselView()
		}
		return m.renderEditForm()

	case editBookProcessing:
		return m.renderMessage("Saving changes...", "Please wait")
	}

	return ""
}

func (m EditBookModel) renderMessage(title, help string) string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render(title))
	b.WriteString("\n\n")
	b.WriteString(tui.StyleHelp.Render(help))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

// renderCarouselView renders the full-screen card grid for navigating between books
func (m EditBookModel) renderCarouselView() string {
	cols := m.carouselColumns()
	usable := m.width - 4
	cardW := usable/cols - 2 // subtract gap between cards
	if cardW < 20 {
		cardW = 20
	}

	saved := 0
	for _, fs := range m.formStates {
		if fs.saved {
			saved++
		}
	}

	// Header
	var header strings.Builder
	header.WriteString(tui.StyleHeader.Render("Select a book to edit"))
	header.WriteString("  ")
	header.WriteString(tui.StyleHelp.Render(fmt.Sprintf("%d/%d saved", saved, len(m.toEdit))))
	header.WriteString("\n\n")

	// Card grid
	var gridRows []string
	for rowStart := 0; rowStart < len(m.toEdit); rowStart += cols {
		var rowCards []string
		for col := 0; col < cols; col++ {
			i := rowStart + col
			if i >= len(m.toEdit) {
				// Empty placeholder to keep grid aligned
				placeholder := lipgloss.NewStyle().
					Width(cardW).
					Height(8).
					Border(lipgloss.HiddenBorder()).
					Render("")
				rowCards = append(rowCards, placeholder)
			} else {
				rowCards = append(rowCards, m.renderCarouselCard(i, cardW))
			}
		}
		gridRows = append(gridRows, lipgloss.JoinHorizontal(lipgloss.Top, rowCards...))
	}

	grid := strings.Join(gridRows, "\n")

	// Footer
	footer := tui.RenderFooterBar([]tui.ShortcutEntry{
		{Key: "tab", Label: "↑↓←→ Navigate"},
		{Key: "enter", Label: "Enter Select"},
		{Key: "", Label: "Esc Back"},
	}, m.activeCmd)

	var b strings.Builder
	b.WriteString(header.String())
	b.WriteString(grid)
	b.WriteString("\n\n")
	b.WriteString(footer)

	outerPad := lipgloss.NewStyle().Padding(1, 2)
	return tui.StyleBorder.Render(outerPad.Render(b.String()))
}

// renderCarouselCard renders a single card in the full-screen carousel
func (m EditBookModel) renderCarouselCard(i, cardW int) string {
	book := m.toEdit[i]
	fs := m.formStates[i]

	titleText := carouselTruncate(book.Book.Title, cardW-2)
	authorText := carouselTruncate(book.Book.Author, cardW-2)
	tagsText := carouselTruncate(strings.Join(book.Book.Tags, ", "), cardW-2)

	var statusText string
	if fs.saved {
		statusText = lipgloss.NewStyle().Foreground(lipgloss.Color("28")).Render("✓ saved")
	} else {
		statusText = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("· unsaved")
	}

	content := titleText + "\n" + authorText + "\n" + tagsText + "\n" + statusText

	isActive := i == m.carouselCursor

	var style lipgloss.Style
	switch {
	case isActive:
		style = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#fb6820")).
			Width(cardW).
			Height(8).
			Padding(1, 1)
	case fs.saved:
		style = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("28")).
			Foreground(lipgloss.Color("242")).
			Width(cardW).
			Height(8).
			Padding(1, 1)
	default:
		style = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Foreground(lipgloss.Color("242")).
			Width(cardW).
			Height(8).
			Padding(1, 1)
	}

	return style.Render(content)
}

// carouselTruncate truncates a string to fit within a carousel card
func carouselTruncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return "…"
	}
	return s[:maxWidth-1] + "…"
}

func (m EditBookModel) renderEditForm() string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder

	// Header
	book := m.toEdit[m.editIndex]
	if len(m.toEdit) > 1 {
		saved := 0
		for _, fs := range m.formStates {
			if fs.saved {
				saved++
			}
		}
		b.WriteString(tui.StyleHeader.Render(fmt.Sprintf("Edit: %s", book.Book.ID)))
		b.WriteString("  ")
		b.WriteString(tui.StyleHelp.Render(fmt.Sprintf("%d/%d saved — ↑ to browse all cards", saved, len(m.toEdit))))
	} else {
		b.WriteString(tui.StyleHeader.Render(fmt.Sprintf("Edit Book: %s", book.Book.ID)))
	}
	b.WriteString("\n\n")

	if m.formErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.formErr)))
		b.WriteString("\n\n")
	}

	if len(book.Book.Tags) > 0 {
		b.WriteString(tui.StyleHelp.Render("Current tags: " + strings.Join(book.Book.Tags, ", ")))
		b.WriteString("\n\n")
	}

	fields := []string{"Title", "Author", "Year", "Tags"}
	for i, label := range fields {
		if i == m.focused {
			b.WriteString(tui.StyleHighlight.Render("› " + label + ":"))
		} else {
			b.WriteString(tui.StyleNormal.Render("  " + label + ":"))
		}
		b.WriteString("\n  ")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}

	b.WriteString("\n")
	if m.confirming {
		b.WriteString(tui.RenderFooterBar([]tui.ShortcutEntry{
			{Key: "y", Label: "Y Confirm"},
			{Key: "n", Label: "N Skip"},
			{Key: "", Label: "Esc Back"},
		}, m.activeCmd))
	} else {
		b.WriteString(tui.RenderFooterBar([]tui.ShortcutEntry{
			{Key: "tab", Label: "Tab/↑↓ Navigate"},
			{Key: "enter", Label: "Enter Submit"},
			{Key: "", Label: "Esc Cancel"},
		}, m.activeCmd))
	}
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

// commitEditsAsync commits all edits in background
func (m EditBookModel) commitEditsAsync() tea.Cmd {
	edits := m.edits
	gh := m.gh
	cfg := m.cfg

	return func() tea.Msg {
		successCount := 0
		failCount := 0

		editsByShelf := make(map[string][]editedBook)
		for _, e := range edits {
			editsByShelf[e.item.ShelfName] = append(editsByShelf[e.item.ShelfName], e)
		}

		for shelfName, shelfEdits := range editsByShelf {
			shelf := cfg.ShelfByName(shelfName)
			if shelf == nil {
				failCount += len(shelfEdits)
				continue
			}

			owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
			catalogPath := shelf.EffectiveCatalogPath()

			catalogData, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
			if err != nil {
				failCount += len(shelfEdits)
				continue
			}
			books, err := catalog.Parse(catalogData)
			if err != nil {
				failCount += len(shelfEdits)
				continue
			}

			var updatedBooks []catalog.Book
			for _, e := range shelfEdits {
				books = catalog.Append(books, e.updated)
				updatedBooks = append(updatedBooks, e.updated)
				successCount++
			}

			updatedData, err := catalog.Marshal(books)
			if err != nil {
				continue
			}

			commitMsg := fmt.Sprintf("edit: update %d books", len(shelfEdits))
			if len(shelfEdits) == 1 {
				commitMsg = fmt.Sprintf("edit: update metadata for %s", shelfEdits[0].item.Book.ID)
			}

			if err := gh.CommitFile(owner, shelf.Repo, catalogPath, updatedData, commitMsg); err != nil {
				continue
			}

			readmeData, _, readmeErr := gh.GetFileContent(owner, shelf.Repo, "README.md", "")
			if readmeErr == nil {
				originalContent := string(readmeData)
				readmeContent := operations.UpdateShelfREADMEStats(originalContent, len(books))
				for _, book := range updatedBooks {
					readmeContent = operations.AppendToShelfREADME(readmeContent, book)
				}
				if readmeContent != originalContent {
					readmeMsg := fmt.Sprintf("Update README: edit %d books", len(updatedBooks))
					if len(updatedBooks) == 1 {
						readmeMsg = fmt.Sprintf("Update README: edit %s", updatedBooks[0].ID)
					}
					_ = gh.CommitFile(owner, shelf.Repo, "README.md", []byte(readmeContent), readmeMsg)
				}
			}
		}

		return EditBookCompleteMsg{
			SuccessCount: successCount,
			FailCount:    failCount,
		}
	}
}

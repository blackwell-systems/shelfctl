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
	m.initFormForBook(0)
	return m
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
		// Handle empty state or error (not form error)
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

	// Forward to text inputs in editing phase
	if m.phase == editBookEditing && !m.confirming {
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
		m.initFormForBook(0)
		m.phase = editBookEditing
		return m, tea.Batch(textinput.Blink, tui.SetActiveCmd(&m.activeCmd, "enter"))
	}

	var cmd tea.Cmd
	m.ms, cmd = m.ms.Update(msg)
	return m, cmd
}

func (m *EditBookModel) initFormForBook(index int) {
	b := &m.toEdit[index].Book

	m.inputs = make([]textinput.Model, 4)
	m.focused = 0
	m.confirming = false
	m.formErr = nil

	// Title
	m.inputs[editFieldTitle] = textinput.New()
	m.inputs[editFieldTitle].Placeholder = b.Title
	m.inputs[editFieldTitle].SetValue(b.Title)
	m.inputs[editFieldTitle].Focus()
	m.inputs[editFieldTitle].CharLimit = 200
	m.inputs[editFieldTitle].Width = 50

	// Author
	m.inputs[editFieldAuthor] = textinput.New()
	m.inputs[editFieldAuthor].Placeholder = "Author name"
	m.inputs[editFieldAuthor].SetValue(b.Author)
	m.inputs[editFieldAuthor].CharLimit = 100
	m.inputs[editFieldAuthor].Width = 50

	// Year
	m.inputs[editFieldYear] = textinput.New()
	m.inputs[editFieldYear].Placeholder = "Publication year (e.g., 2023)"
	if b.Year > 0 {
		m.inputs[editFieldYear].SetValue(strconv.Itoa(b.Year))
	}
	m.inputs[editFieldYear].CharLimit = 4
	m.inputs[editFieldYear].Width = 50

	// Tags
	m.inputs[editFieldTags] = textinput.New()
	m.inputs[editFieldTags].Placeholder = "comma,separated,tags"
	if len(b.Tags) > 0 {
		m.inputs[editFieldTags].SetValue(strings.Join(b.Tags, ","))
	}
	m.inputs[editFieldTags].CharLimit = 200
	m.inputs[editFieldTags].Width = 50
}

func (m EditBookModel) updateEditing(msg tea.KeyMsg) (EditBookModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }

	case "esc":
		if m.confirming {
			// Cancel confirmation, go back to form
			m.confirming = false
			return m, nil
		}
		// Cancel editing entirely, return to hub
		return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }

	case "enter":
		highlightCmd := tui.SetActiveCmd(&m.activeCmd, "enter")
		if m.confirming {
			m, cmd := m.submitCurrentBook()
			return m, tea.Batch(cmd, highlightCmd)
		}
		// Show confirmation
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

func (m EditBookModel) submitCurrentBook() (EditBookModel, tea.Cmd) {
	// Parse year
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

	// Parse tags
	var tags []string
	if tagStr := m.inputs[editFieldTags].Value(); tagStr != "" {
		for _, t := range strings.Split(tagStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	// Build updated book
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

	return m.advanceToNextBook()
}

func (m EditBookModel) advanceToNextBook() (EditBookModel, tea.Cmd) {
	m.editIndex++

	if m.editIndex >= len(m.toEdit) {
		// All books edited, start processing
		if len(m.edits) == 0 {
			// Nothing was actually edited
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		}
		m.phase = editBookProcessing
		return m, m.commitEditsAsync()
	}

	// Initialize form for next book
	m.initFormForBook(m.editIndex)
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

func (m EditBookModel) renderEditForm() string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder

	// Header with progress
	book := m.toEdit[m.editIndex]
	if len(m.toEdit) > 1 {
		b.WriteString(tui.StyleHeader.Render(fmt.Sprintf("Edit Book [%d/%d]: %s", m.editIndex+1, len(m.toEdit), book.Book.ID)))
	} else {
		b.WriteString(tui.StyleHeader.Render(fmt.Sprintf("Edit Book: %s", book.Book.ID)))
	}
	b.WriteString("\n\n")

	if m.formErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.formErr)))
		b.WriteString("\n\n")
	}

	// Show current tags
	if len(book.Book.Tags) > 0 {
		b.WriteString(tui.StyleHelp.Render("Current tags: " + strings.Join(book.Book.Tags, ", ")))
		b.WriteString("\n\n")
	}

	// Form fields
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

	// Help text or confirmation
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

		// Group by shelf for batch commits
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

			// Load catalog once for this shelf
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

			// Apply all edits
			for _, e := range shelfEdits {
				books = catalog.Append(books, e.updated)
				updatedBooks = append(updatedBooks, e.updated)
				successCount++
			}

			// Commit catalog once
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

			// Update README
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

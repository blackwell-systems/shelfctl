package unified

import (
	"fmt"
	"strings"

	"github.com/blackwell-systems/bubbletea-multiselect"
	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/operations"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// deleteBookPhase tracks the current phase of the delete workflow
type deleteBookPhase int

const (
	deleteBookPicking    deleteBookPhase = iota // Showing book picker
	deleteBookConfirming                        // Showing confirmation screen
	deleteBookProcessing                        // Deleting books
)

// DeleteBookCompleteMsg is emitted when deletion finishes
type DeleteBookCompleteMsg struct {
	SuccessCount int
	FailCount    int
}

// DeleteBookModel is the unified view for deleting books
type DeleteBookModel struct {
	phase     deleteBookPhase
	ms        multiselect.Model
	gh        *github.Client
	cfg       *config.Config
	cacheMgr  *cache.Manager
	width     int
	height    int
	err       error
	empty     bool // true if no books
	activeCmd string

	// Confirmation phase
	toDelete []tui.BookItem

	// Results
	successCount int
	failCount    int
}

// NewDeleteBookModel creates a new delete-book view
func NewDeleteBookModel(books []tui.BookItem, gh *github.Client, cfg *config.Config, cacheMgr *cache.Manager) DeleteBookModel {
	if len(books) == 0 {
		return DeleteBookModel{
			gh:       gh,
			cfg:      cfg,
			cacheMgr: cacheMgr,
			empty:    true,
		}
	}

	ms, err := tui.NewBookPickerMultiModel(books, "Select books to delete")
	if err != nil {
		return DeleteBookModel{
			gh:       gh,
			cfg:      cfg,
			cacheMgr: cacheMgr,
			err:      err,
		}
	}

	return DeleteBookModel{
		phase:    deleteBookPicking,
		ms:       ms,
		gh:       gh,
		cfg:      cfg,
		cacheMgr: cacheMgr,
	}
}

// Init initializes the delete-book view
func (m DeleteBookModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the delete-book view
func (m DeleteBookModel) Update(msg tea.Msg) (DeleteBookModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.phase == deleteBookPicking {
			h, v := tui.StyleBorder.GetFrameSize()
			m.ms.List.SetSize(msg.Width-h, msg.Height-v)
			m.ms.List.Title = tui.StyleHeader.Render("Select books to delete") + "\n" + tui.RenderColumnHeader(m.ms.List.Width())
			m.ms.List.Styles.Title = lipgloss.NewStyle()
		}
		return m, nil

	case tui.ClearActiveCmdMsg:
		m.activeCmd = ""
		return m, nil

	case tea.KeyMsg:
		// Handle empty state or error
		if m.empty || m.err != nil {
			switch msg.String() {
			case "enter", "esc", "q":
				return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
			}
			return m, nil
		}

		switch m.phase {
		case deleteBookPicking:
			return m.updatePicking(msg)
		case deleteBookConfirming:
			return m.updateConfirming(msg)
		case deleteBookProcessing:
			// Ignore input during processing
			return m, nil
		}

	case DeleteBookCompleteMsg:
		m.successCount = msg.SuccessCount
		m.failCount = msg.FailCount
		// Return to hub
		return m, func() tea.Msg {
			return NavigateMsg{Target: "hub"}
		}
	}

	// Forward to picker in picking phase
	if m.phase == deleteBookPicking {
		var cmd tea.Cmd
		m.ms, cmd = m.ms.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m DeleteBookModel) updatePicking(msg tea.KeyMsg) (DeleteBookModel, tea.Cmd) {
	// Don't handle keys when filtering
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
		// Toggle checkbox
		m.ms.Toggle()
		return m, tui.SetActiveCmd(&m.activeCmd, "space")

	case "enter":
		// Collect selected books
		selected := tui.CollectSelectedBooks(&m.ms)
		if len(selected) == 0 {
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		}

		m.toDelete = selected

		// Switch to confirmation phase
		m.phase = deleteBookConfirming
		return m, tui.SetActiveCmd(&m.activeCmd, "enter")
	}

	// Forward other keys to multiselect
	var cmd tea.Cmd
	m.ms, cmd = m.ms.Update(msg)
	return m, cmd
}

func (m DeleteBookModel) updateConfirming(msg tea.KeyMsg) (DeleteBookModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }

	case "esc", "n":
		// Go back to picker
		m.phase = deleteBookPicking
		return m, nil

	case "enter", "y":
		// Confirm - start processing
		highlightCmd := tui.SetActiveCmd(&m.activeCmd, msg.String())
		m.phase = deleteBookProcessing
		return m, tea.Batch(m.deleteAsync(), highlightCmd)
	}

	return m, nil
}

// View renders the delete-book view
func (m DeleteBookModel) View() string {
	// Empty state
	if m.empty {
		return m.renderMessage("No books in library", "Press Enter to return to menu")
	}

	// Error state
	if m.err != nil && m.phase == deleteBookPicking {
		return m.renderMessage(fmt.Sprintf("Error: %v", m.err), "Press Enter to return to menu")
	}

	switch m.phase {
	case deleteBookPicking:
		return tui.StyleBorder.Render(m.ms.View())

	case deleteBookConfirming:
		return m.renderConfirmation()

	case deleteBookProcessing:
		return m.renderMessage("Deleting books...", "Please wait")
	}

	return ""
}

func (m DeleteBookModel) renderMessage(title, help string) string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render(title))
	b.WriteString("\n\n")
	b.WriteString(tui.StyleHelp.Render(help))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

func (m DeleteBookModel) renderConfirmation() string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder

	dangerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)

	b.WriteString(dangerStyle.Render("Confirm Delete"))
	b.WriteString("\n\n")

	// Show books to delete
	b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("Delete %d book(s):", len(m.toDelete))))
	b.WriteString("\n")
	for _, item := range m.toDelete {
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  - %s (%s) [%s]", item.Book.ID, item.Book.Title, item.ShelfName)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dangerStyle.Render("This will:"))
	b.WriteString("\n")
	b.WriteString(tui.StyleNormal.Render("  - Remove books from catalog.yml"))
	b.WriteString("\n")
	b.WriteString(dangerStyle.Render("  - DELETE files from GitHub Release assets"))
	b.WriteString("\n\n")
	b.WriteString(dangerStyle.Render("THIS CANNOT BE UNDONE"))
	b.WriteString("\n\n")

	// Help
	b.WriteString(tui.RenderFooterBar([]tui.ShortcutEntry{
		{Key: "enter", Label: "Enter/y Confirm"},
		{Key: "", Label: "Esc/n Cancel"},
	}, m.activeCmd))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

// deleteAsync removes books in background
func (m DeleteBookModel) deleteAsync() tea.Cmd {
	toDelete := m.toDelete
	gh := m.gh
	cfg := m.cfg
	cacheMgr := m.cacheMgr

	return func() tea.Msg {
		successCount := 0
		failCount := 0

		for _, item := range toDelete {
			if err := deleteSingleBookOp(item, gh, cfg, cacheMgr); err != nil {
				failCount++
			} else {
				successCount++
			}
		}

		return DeleteBookCompleteMsg{
			SuccessCount: successCount,
			FailCount:    failCount,
		}
	}
}

// deleteSingleBookOp deletes a single book: removes asset, updates catalog, clears cache, updates README
func deleteSingleBookOp(item tui.BookItem, gh *github.Client, cfg *config.Config, cacheMgr *cache.Manager) error {
	shelf := cfg.ShelfByName(item.ShelfName)
	if shelf == nil {
		return fmt.Errorf("shelf %q not found", item.ShelfName)
	}

	catalogPath := shelf.EffectiveCatalogPath()
	releaseTag := shelf.EffectiveRelease(cfg.Defaults.Release)

	// Get the release
	rel, err := gh.GetReleaseByTag(item.Owner, item.Repo, releaseTag)
	if err != nil {
		return fmt.Errorf("could not get release: %w", err)
	}

	// Find the asset
	asset, err := gh.FindAsset(item.Owner, item.Repo, rel.ID, item.Book.Source.Asset)
	if err != nil {
		return fmt.Errorf("could not find asset: %w", err)
	}
	if asset == nil {
		return fmt.Errorf("asset %q not found in release", item.Book.Source.Asset)
	}

	// Delete the asset from GitHub
	if err := gh.DeleteAsset(item.Owner, item.Repo, asset.ID); err != nil {
		return fmt.Errorf("could not delete asset: %w", err)
	}

	// Load catalog
	data, _, err := gh.GetFileContent(item.Owner, item.Repo, catalogPath, "")
	if err != nil {
		return fmt.Errorf("could not load catalog: %w", err)
	}
	books, err := catalog.Parse(data)
	if err != nil {
		return fmt.Errorf("could not parse catalog: %w", err)
	}

	// Remove from catalog
	books, removed := catalog.Remove(books, item.Book.ID)
	if !removed {
		return fmt.Errorf("book %q not found in catalog", item.Book.ID)
	}

	// Marshal and commit updated catalog
	updatedData, err := catalog.Marshal(books)
	if err != nil {
		return fmt.Errorf("could not marshal catalog: %w", err)
	}
	commitMsg := fmt.Sprintf("delete: %s", item.Book.ID)
	if err := gh.CommitFile(item.Owner, item.Repo, catalogPath, updatedData, commitMsg); err != nil {
		return fmt.Errorf("could not commit catalog: %w", err)
	}

	// Clear from cache
	if cacheMgr.Exists(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset) {
		_ = cacheMgr.Remove(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset)
	}

	// Update README
	readmeData, _, err := gh.GetFileContent(item.Owner, item.Repo, "README.md", "")
	if err == nil {
		originalContent := string(readmeData)
		readmeContent := operations.UpdateShelfREADMEStats(originalContent, len(books))
		readmeContent = operations.RemoveFromShelfREADME(readmeContent, item.Book.ID)

		if readmeContent != originalContent {
			readmeMsg := fmt.Sprintf("Update README: remove %s", item.Book.ID)
			_ = gh.CommitFile(item.Owner, item.Repo, "README.md", []byte(readmeContent), readmeMsg)
		}
	}

	return nil
}

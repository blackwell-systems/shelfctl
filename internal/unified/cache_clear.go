package unified

import (
	"fmt"
	"os"
	"strings"

	"github.com/blackwell-systems/bubbletea-components/multiselect"
	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// cacheClearPhase tracks the current phase of the workflow
type cacheClearPhase int

const (
	cacheClearPicking    cacheClearPhase = iota // Showing book picker
	cacheClearConfirming                        // Showing confirmation screen
	cacheClearProcessing                        // Removing files
)

// CacheClearModel is the unified view for clearing cached books
type CacheClearModel struct {
	phase     cacheClearPhase
	ms        multiselect.Model
	cacheMgr  *cache.Manager
	width     int
	height    int
	err       error
	empty     bool // true if no cached books

	// Confirmation phase
	toRemove  []tui.BookItem
	skipped   []tui.BookItem
	totalSize int64

	// Results
	successCount int
	failCount    int
}

// NewCacheClearModel creates a new cache-clear view
func NewCacheClearModel(books []tui.BookItem, cacheMgr *cache.Manager) CacheClearModel {
	// Filter to only cached books
	var cachedBooks []tui.BookItem
	for _, item := range books {
		if item.Cached {
			cachedBooks = append(cachedBooks, item)
		}
	}

	if len(cachedBooks) == 0 {
		return CacheClearModel{
			cacheMgr: cacheMgr,
			empty:    true,
		}
	}

	ms, err := tui.NewBookPickerMultiModel(cachedBooks, "Select books to remove from cache")
	if err != nil {
		return CacheClearModel{
			cacheMgr: cacheMgr,
			err:      err,
		}
	}

	return CacheClearModel{
		phase:    cacheClearPicking,
		ms:       ms,
		cacheMgr: cacheMgr,
	}
}

// Init initializes the cache-clear view
func (m CacheClearModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the cache-clear view
func (m CacheClearModel) Update(msg tea.Msg) (CacheClearModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.phase == cacheClearPicking {
			h, v := tui.StyleBorder.GetFrameSize()
			m.ms.List.SetSize(msg.Width-h, msg.Height-v)
		}
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
		case cacheClearPicking:
			return m.updatePicking(msg)
		case cacheClearConfirming:
			return m.updateConfirming(msg)
		case cacheClearProcessing:
			// Ignore input during processing
			return m, nil
		}

	case CacheClearCompleteMsg:
		m.successCount = msg.SuccessCount
		m.failCount = msg.FailCount
		// Return to hub
		return m, func() tea.Msg {
			return NavigateMsg{Target: "hub"}
		}
	}

	// Forward to picker in picking phase
	if m.phase == cacheClearPicking {
		var cmd tea.Cmd
		m.ms, cmd = m.ms.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m CacheClearModel) updatePicking(msg tea.KeyMsg) (CacheClearModel, tea.Cmd) {
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
		// Return to hub (instant, no terminal drop)
		return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }

	case " ":
		// Toggle checkbox
		m.ms.Toggle()
		return m, nil

	case "enter":
		// Collect selected books
		selected := tui.CollectSelectedBooks(&m.ms)
		if len(selected) == 0 {
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		}

		// Separate modified from unmodified
		m.toRemove = nil
		m.skipped = nil
		m.totalSize = 0

		for _, item := range selected {
			path := m.cacheMgr.Path(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset)
			if info, err := os.Stat(path); err == nil {
				m.totalSize += info.Size()
			}

			// Check if modified
			if m.cacheMgr.HasBeenModified(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset, item.Book.Checksum.SHA256) {
				m.skipped = append(m.skipped, item)
			} else {
				m.toRemove = append(m.toRemove, item)
			}
		}

		// If nothing to remove (all modified), show error
		if len(m.toRemove) == 0 {
			m.err = fmt.Errorf("all selected books have local changes and cannot be removed (use CLI with --force)")
			return m, nil
		}

		// Switch to confirmation phase
		m.phase = cacheClearConfirming
		return m, nil
	}

	// Forward other keys to multiselect
	var cmd tea.Cmd
	m.ms, cmd = m.ms.Update(msg)
	return m, cmd
}

func (m CacheClearModel) updateConfirming(msg tea.KeyMsg) (CacheClearModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }

	case "esc", "n":
		// Go back to picker
		m.phase = cacheClearPicking
		return m, nil

	case "enter", "y":
		// Confirm - start processing
		m.phase = cacheClearProcessing
		return m, m.clearCacheAsync()
	}

	return m, nil
}

// View renders the cache-clear view
func (m CacheClearModel) View() string {
	// Empty state
	if m.empty {
		return m.renderMessage("No books in cache", "Press Enter to return to menu")
	}

	// Error state (before picker)
	if m.err != nil && m.phase == cacheClearPicking {
		return m.renderMessage(fmt.Sprintf("Error: %v", m.err), "Press Enter to return to menu")
	}

	switch m.phase {
	case cacheClearPicking:
		return tui.StyleBorder.Render(m.ms.View())

	case cacheClearConfirming:
		return m.renderConfirmation()

	case cacheClearProcessing:
		return m.renderMessage("Removing books from cache...", "Please wait")
	}

	return ""
}

func (m CacheClearModel) renderMessage(title, help string) string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render(title))
	b.WriteString("\n\n")
	b.WriteString(tui.StyleHelp.Render(help))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

func (m CacheClearModel) renderConfirmation() string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder

	b.WriteString(tui.StyleHeader.Render("Confirm Cache Clear"))
	b.WriteString("\n\n")

	// Show skipped books (modified)
	if len(m.skipped) > 0 {
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("Skipping %d modified books (have local changes):", len(m.skipped))))
		b.WriteString("\n")
		for _, item := range m.skipped {
			b.WriteString(tui.StyleHelp.Render(fmt.Sprintf("  - %s (%s)", item.Book.ID, item.Book.Title)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Show books to remove
	b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("Remove %d books from cache:", len(m.toRemove))))
	b.WriteString("\n")
	for _, item := range m.toRemove {
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  - %s (%s)", item.Book.ID, item.Book.Title)))
		b.WriteString("\n")
	}

	// Size info
	b.WriteString("\n")
	b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("Space to free: %s", humanBytes(m.totalSize))))
	b.WriteString("\n\n")

	// Help
	b.WriteString(tui.StyleHelp.Render("Enter/y: Confirm   Esc/n: Cancel"))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

// clearCacheAsync removes books from cache in background
func (m CacheClearModel) clearCacheAsync() tea.Cmd {
	toRemove := m.toRemove
	cacheMgr := m.cacheMgr

	return func() tea.Msg {
		successCount := 0
		failCount := 0

		for _, item := range toRemove {
			err := cacheMgr.Remove(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset)
			if err != nil {
				failCount++
			} else {
				successCount++
			}
		}

		return CacheClearCompleteMsg{
			SuccessCount: successCount,
			FailCount:    failCount,
		}
	}
}

// humanBytes is defined in model.go

package unified

import (
	"fmt"
	"io"
	"strings"

	"github.com/blackwell-systems/bubbletea-commandpalette"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/blackwell-systems/shelfctl/internal/tui/delegate"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HubModel is the unified-mode version of the hub menu
type HubModel struct {
	list              list.Model
	context           tui.HubContext
	width             int
	height            int
	shelfData         string
	showDetails       bool
	detailsType       string
	detailsFocused    bool   // true when focus is on details panel
	detailsScroll     int    // scroll offset for details panel (line number)
	cachedDetailsRaw  string // cached raw content for current detailsType
	cachedDetailsType string // detailsType when cachedDetailsRaw was computed
	palette           commandpalette.Model
	paletteOpen       bool
	pendingNavMsg     tea.Msg // set by palette action, consumed by orchestrator
}

type hubKeys struct {
	quit        key.Binding
	selectItem  key.Binding
	switchFocus key.Binding
	scrollUp    key.Binding
	scrollDown  key.Binding
	palette     key.Binding
}

var hubKeyMap = hubKeys{
	quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	selectItem: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	switchFocus: key.NewBinding(
		key.WithKeys("tab", "right"),
		key.WithHelp("tab/→", "focus details"),
	),
	scrollUp: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "scroll up"),
	),
	scrollDown: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "scroll down"),
	),
	palette: key.NewBinding(
		key.WithKeys("ctrl+p"),
		key.WithHelp("ctrl+p", "command palette"),
	),
}

// NewHubModel creates a new hub model for unified mode
func NewHubModel(ctx tui.HubContext) HubModel {
	items := tui.BuildFilteredMenuItems(ctx)

	// Create list with custom delegate
	d := delegate.New(renderHubMenuItem)
	l := list.New(items, d, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.HelpStyle = tui.StyleHelp

	// Set help
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{hubKeyMap.selectItem, hubKeyMap.palette}
	}

	// Start cursor on first real item (index 0 is a separator)
	for i, item := range items {
		if _, isSep := item.(tui.MenuSeparator); !isSep {
			l.Select(i)
			break
		}
	}

	return HubModel{
		list:    l,
		context: ctx,
		palette: buildHubPalette(ctx),
	}
}

func (m HubModel) Init() tea.Cmd {
	return nil
}

func (m HubModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Track navigation direction so we can skip separators after list update
	navDir := 0
	if keyMsg, ok := msg.(tea.KeyMsg); ok && m.list.FilterState() != list.Filtering && !m.detailsFocused {
		switch keyMsg.String() {
		case "up", "k":
			navDir = -1
		case "down", "j":
			navDir = 1
		}
	}

	switch msg := msg.(type) {
	case commandpalette.ActionSelectedMsg:
		m.paletteOpen = false
		// Store the navigation result for the orchestrator to pick up
		// in the same Update cycle (avoids one-frame flash of hub
		// without palette before navigation occurs).
		m.pendingNavMsg = msg.Action.Run()
		return m, nil

	case tea.KeyMsg:
		// Command palette routing — intercepts all keys while open
		if m.paletteOpen {
			switch msg.String() {
			case "esc", "ctrl+c":
				m.paletteOpen = false
				return m, nil
			default:
				var cmd tea.Cmd
				m.palette, cmd = m.palette.Update(msg)
				return m, cmd
			}
		}

		// Open palette with ctrl+p (before filtering check so it always works)
		if key.Matches(msg, hubKeyMap.palette) {
			m.paletteOpen = true
			m.palette.Reset()
			return m, m.palette.Focus()
		}

		// Don't handle keys when filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

		// Handle focus switching and scrolling when details panel is shown
		if m.showDetails {
			switch {
			case key.Matches(msg, hubKeyMap.switchFocus):
				// Tab or right arrow switches focus
				if m.detailsFocused {
					// Return focus to menu
					m.detailsFocused = false
					m.detailsScroll = 0
				} else {
					// Move focus to details panel
					m.detailsFocused = true
					m.detailsScroll = 0
				}
				return m, nil

			case msg.String() == "left":
				// Left arrow returns focus from details to menu
				if m.detailsFocused {
					m.detailsFocused = false
					m.detailsScroll = 0
					return m, nil
				}

			case key.Matches(msg, hubKeyMap.scrollUp):
				if m.detailsFocused {
					// Scroll details up
					if m.detailsScroll > 0 {
						m.detailsScroll--
					}
					return m, nil
				}

			case key.Matches(msg, hubKeyMap.scrollDown):
				if m.detailsFocused {
					// Scroll details down (limit checked in render)
					m.detailsScroll++
					return m, nil
				}

			case key.Matches(msg, hubKeyMap.quit):
				if m.detailsFocused {
					// First escape returns focus to menu
					m.detailsFocused = false
					m.detailsScroll = 0
					return m, nil
				}
				// Second escape quits app
				return m, func() tea.Msg { return QuitAppMsg{} }
			}
		}

		// Normal menu key handling (when not focused on details)
		if !m.detailsFocused {
			switch {
			case key.Matches(msg, hubKeyMap.quit):
				// In unified mode, quitting hub means quitting the app
				return m, func() tea.Msg { return QuitAppMsg{} }

			case key.Matches(msg, hubKeyMap.selectItem):
				if item, ok := m.list.SelectedItem().(tui.MenuItem); ok {
					itemKey := item.GetKey()
					if itemKey == "quit" {
						return m, func() tea.Msg { return QuitAppMsg{} }
					}
					// Emit navigation message to switch views
					return m, func() tea.Msg {
						return NavigateMsg{Target: itemKey}
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateListSize()
		m.palette.SetSize(msg.Width, msg.Height)
	}

	// Only update list if not focused on details
	var cmd tea.Cmd
	if !m.detailsFocused {
		m.list, cmd = m.list.Update(msg)
	}

	// Skip over separators: if navigation landed on one, keep stepping.
	// If no real item exists in that direction, stay put (no wrap past edges).
	if navDir != 0 {
		if _, isSep := m.list.SelectedItem().(tui.MenuSeparator); isSep {
			items := m.list.Items()
			idx := m.list.Index()
			found := false
			for {
				idx += navDir
				if idx < 0 || idx >= len(items) {
					break
				}
				if _, isSep := items[idx].(tui.MenuSeparator); !isSep {
					m.list.Select(idx)
					found = true
					break
				}
			}
			if !found {
				// Restore to nearest real item in opposite direction
				idx = m.list.Index()
				for {
					idx -= navDir
					if idx < 0 || idx >= len(items) {
						break
					}
					if _, isSep := items[idx].(tui.MenuSeparator); !isSep {
						m.list.Select(idx)
						break
					}
				}
			}
		}
	}

	// Check if we should show details panel
	selectedItem := m.list.SelectedItem()
	if _, isSep := selectedItem.(tui.MenuSeparator); isSep {
		// Separator selected — hide details panel
		m.showDetails = false
		m.detailsType = ""
		m.detailsFocused = false
		m.detailsScroll = 0
		m.updateListSize()
	} else if item, ok := selectedItem.(tui.MenuItem); ok {
		itemKey := item.GetKey()
		prevDetailsType := m.detailsType
		if itemKey == "shelves" || itemKey == "cache-info" {
			m.showDetails = true
			m.detailsType = itemKey
			// Reset focus, scroll, and rebuild cache when switching detail types
			if prevDetailsType != itemKey {
				m.detailsFocused = false
				m.detailsScroll = 0
				m.cachedDetailsType = itemKey
				switch itemKey {
				case "shelves":
					m.cachedDetailsRaw = m.renderShelvesDetailsRaw()
				case "cache-info":
					m.cachedDetailsRaw = m.renderCacheDetailsRaw()
				}
			}
		} else {
			m.showDetails = false
			m.detailsType = ""
			m.detailsFocused = false
			m.detailsScroll = 0
		}
		m.updateListSize()
	}

	return m, cmd
}

func (m HubModel) View() string {
	// Outer container for centering
	outerStyle := lipgloss.NewStyle().
		Padding(2, 4)

	// Two-tone wordmark header
	wordmark := lipgloss.NewStyle().Bold(true).Foreground(tui.ColorOrange).Render("shelf") +
		lipgloss.NewStyle().Bold(true).Foreground(tui.ColorTealLight).Render("ctl")
	header := lipgloss.NewStyle().Padding(0, 1).Render(wordmark + "  " +
		lipgloss.NewStyle().Foreground(tui.ColorGray).Render("Personal Library Manager"))

	// Stat pills status bar
	var statusBar string
	if m.context.ShelfCount > 0 {
		num := lipgloss.NewStyle().Foreground(tui.ColorTealLight).Bold(true)
		dim := lipgloss.NewStyle().Foreground(tui.ColorGray)
		stat := "  " + num.Render(fmt.Sprintf("%d", m.context.ShelfCount)) + dim.Render(" shelves")
		if m.context.BookCount > 0 {
			stat += "  " + num.Render(fmt.Sprintf("%d", m.context.BookCount)) + dim.Render(" books")
		}
		if m.context.CachedCount > 0 {
			stat += "  " + num.Render(fmt.Sprintf("%d", m.context.CachedCount)) + dim.Render(" cached")
		}
		statusBar = stat
	}

	// Combine header, status, and list
	parts := []string{header}
	if statusBar != "" {
		parts = append(parts, statusBar)
	}
	parts = append(parts, m.list.View())

	listContent := lipgloss.JoinVertical(lipgloss.Left, parts...)

	var content string
	if m.showDetails {
		// Split-panel layout
		listBorderColor := tui.ColorGray
		if m.detailsFocused {
			// Dim the menu when details panel is focused
			listBorderColor = lipgloss.AdaptiveColor{Light: "#555555", Dark: "#555555"}
		}
		listStyle := lipgloss.NewStyle().
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(listBorderColor)
		listView := listStyle.Render(listContent)
		detailsView := m.renderDetailsPane()

		content = lipgloss.JoinHorizontal(
			lipgloss.Top,
			listView,
			detailsView,
		)
	} else {
		// Single panel: menu only
		content = listContent
	}

	// Add padding inside the border
	innerPadding := lipgloss.NewStyle().
		Padding(0, 2, 0, 1)

	base := outerStyle.Render(tui.StyleBorder.Render(innerPadding.Render(content)))

	if m.paletteOpen {
		overlay := m.palette.View()
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			overlay,
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	return base
}

func (m *HubModel) updateListSize() {
	// Account for outer padding, inner padding, border, and header content
	const outerPaddingH = 4 * 2
	const outerPaddingV = 2 * 2
	const innerPaddingH = 1 + 2
	const headerLines = 4
	const borderWidth = 1
	h, v := tui.StyleBorder.GetFrameSize()

	availableWidth := m.width - outerPaddingH - innerPaddingH - h
	listHeight := m.height - outerPaddingV - v - headerLines

	if m.showDetails {
		const detailsPanelWidth = 45 + 2
		listWidth := availableWidth - detailsPanelWidth - borderWidth
		if listWidth < 30 {
			listWidth = 30
		}
		m.list.SetSize(listWidth, listHeight)
	} else {
		if availableWidth < 40 {
			availableWidth = 40
		}
		m.list.SetSize(availableWidth, listHeight)
	}

	if listHeight < 5 {
		m.list.SetSize(availableWidth, 5)
	}
}

func (m HubModel) renderDetailsPane() string {
	// Use cached raw content if detailsType hasn't changed
	rawContent := m.cachedDetailsRaw
	if m.cachedDetailsType != m.detailsType {
		switch m.detailsType {
		case "shelves":
			rawContent = m.renderShelvesDetailsRaw()
		case "cache-info":
			rawContent = m.renderCacheDetailsRaw()
		default:
			return ""
		}
	}

	// Apply scrolling and viewport
	return m.applyScrollViewport(rawContent)
}

// applyScrollViewport takes raw content, applies scroll offset, and adds indicators
func (m HubModel) applyScrollViewport(rawContent string) string {
	const detailsWidth = 45

	// Calculate available height for details
	// Account for: outer padding (4), inner padding (2), border (2), header (3), status (1), help (1)
	availableHeight := m.height - 13
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Split content into lines
	lines := strings.Split(rawContent, "\n")
	totalLines := len(lines)

	// Clamp scroll position
	maxScroll := totalLines - availableHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.detailsScroll > maxScroll {
		m.detailsScroll = maxScroll
	}
	if m.detailsScroll < 0 {
		m.detailsScroll = 0
	}

	// Extract visible window
	start := m.detailsScroll
	end := start + availableHeight
	if end > totalLines {
		end = totalLines
	}

	visibleLines := lines[start:end]
	visibleContent := strings.Join(visibleLines, "\n")

	// Add scroll indicators
	var header string
	if m.detailsFocused {
		header = tui.StyleHighlight.Render("◆ Details Panel (↑↓ scroll · tab/esc to exit)")
	} else {
		header = tui.StyleHelp.Render("◇ Details Panel (tab/→ to focus & scroll)")
	}

	var scrollInfo string
	if totalLines > availableHeight {
		scrollInfo = tui.StyleHelp.Render(
			fmt.Sprintf("  Lines %d-%d of %d", start+1, end, totalLines),
		)
	}

	// Build final content
	parts := []string{header}
	if scrollInfo != "" {
		parts = append(parts, scrollInfo)
	}
	parts = append(parts, "", visibleContent)

	content := strings.Join(parts, "\n")

	detailsStyle := lipgloss.NewStyle().
		Width(detailsWidth).
		Padding(1, 1)

	// Add left border when focused for visual distinction
	if m.detailsFocused {
		detailsStyle = detailsStyle.
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("86")) // Cyan to match header
	}

	return detailsStyle.Render(content)
}

func (m HubModel) renderShelvesDetailsRaw() string {
	if len(m.context.ShelfDetails) == 0 {
		return "No shelves configured"
	}

	var s strings.Builder
	s.WriteString(tui.StyleHeader.Render("Configured Shelves"))
	s.WriteString("\n\n")

	for _, shelf := range m.context.ShelfDetails {
		s.WriteString(tui.StyleHighlight.Render(shelf.Name))
		s.WriteString("\n")
		fmt.Fprintf(&s, "  Repo: %s/%s\n", shelf.Owner, shelf.Repo)
		fmt.Fprintf(&s, "  Books: %d\n", shelf.BookCount)
		fmt.Fprintf(&s, "  Status: %s\n", shelf.Status)
		s.WriteString("\n")
	}

	return s.String()
}

func (m HubModel) renderCacheDetailsRaw() string {
	var s strings.Builder
	s.WriteString(tui.StyleHeader.Render("Cache Statistics"))
	s.WriteString("\n\n")

	// Total books
	s.WriteString(tui.StyleHighlight.Render("Total Books: "))
	fmt.Fprintf(&s, "%d\n", m.context.BookCount)

	// Cached count
	s.WriteString(tui.StyleHighlight.Render("Cached: "))
	if m.context.CachedCount > 0 {
		fmt.Fprintf(&s, "%s (%d books)\n", tui.StyleCached.Render(fmt.Sprintf("✓ %d", m.context.CachedCount)), m.context.CachedCount)
	} else {
		s.WriteString("0\n")
	}

	// Uncached count
	uncached := m.context.BookCount - m.context.CachedCount
	if uncached > 0 {
		s.WriteString(tui.StyleHighlight.Render("Not Cached: "))
		fmt.Fprintf(&s, "%d\n", uncached)
	}

	// Modified count and list
	if m.context.ModifiedCount > 0 {
		s.WriteString("\n")
		s.WriteString(tui.StyleHighlight.Render("Modified Books:"))
		s.WriteString("\n")
		s.WriteString(tui.StyleHelp.Render(fmt.Sprintf("  %d books with local changes", m.context.ModifiedCount)))
		s.WriteString("\n\n")

		// Show ALL modified books (not limited to 10) so scrolling is useful
		for _, book := range m.context.ModifiedBooks {
			s.WriteString("  • ")
			s.WriteString(tui.StyleHighlight.Render(book.ID))
			s.WriteString("\n")
		}

		s.WriteString("\n")
		s.WriteString(tui.StyleHelp.Render("  Press 's' in browse or run 'sync --all'"))
		s.WriteString("\n")
	}

	// Cache size
	if m.context.CacheSize > 0 {
		s.WriteString("\n")
		s.WriteString(tui.StyleHighlight.Render("Disk Usage: "))
		fmt.Fprintf(&s, "%s\n", formatBytes(m.context.CacheSize))
	}

	// Cache directory
	if m.context.CacheDir != "" {
		s.WriteString("\n")
		s.WriteString(tui.StyleHighlight.Render("Location: "))
		s.WriteString(tui.StyleHelp.Render(m.context.CacheDir))
		s.WriteString("\n")
	}

	return s.String()
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// buildHubPalette creates a command palette pre-populated with actions from the hub menu.
func buildHubPalette(ctx tui.HubContext) commandpalette.Model {
	items := tui.BuildFilteredMenuItems(ctx)
	var actions []commandpalette.Action
	for _, item := range items {
		mi, ok := item.(tui.MenuItem)
		if !ok {
			continue // skip separators
		}
		itemKey := mi.GetKey()
		actions = append(actions, commandpalette.Action{
			Label:    mi.GetLabel() + "  " + mi.GetDescription(),
			Keywords: []string{itemKey, mi.GetDescription()},
			Run: func() tea.Msg {
				if itemKey == "quit" {
					return QuitAppMsg{}
				}
				return NavigateMsg{Target: itemKey}
			},
		})
	}
	return commandpalette.New(commandpalette.Config{
		Actions:     actions,
		ActiveColor: tui.ColorOrange,
	})
}

func renderHubMenuItem(w io.Writer, m list.Model, index int, item list.Item) {
	// Render section separator
	if sep, ok := item.(tui.MenuSeparator); ok {
		line := lipgloss.NewStyle().Foreground(tui.ColorGray).Render("─── " + sep.Title + " ───")
		_, _ = fmt.Fprint(w, "  "+line)
		return
	}

	menuItem, ok := item.(tui.MenuItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	label := menuItem.GetLabel()
	desc := tui.StyleHelp.Render(menuItem.GetDescription())
	display := fmt.Sprintf("%-25s   %s", label, desc)

	if isSelected {
		_, _ = fmt.Fprint(w, tui.StyleHighlight.Render("› "+display))
	} else {
		_, _ = fmt.Fprint(w, "  "+tui.StyleNormal.Render(display))
	}
}

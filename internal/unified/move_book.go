package unified

import (
	"fmt"
	"strings"

	"github.com/blackwell-systems/bubbletea-components/multiselect"
	"github.com/blackwell-systems/shelfctl/internal/cache"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
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

	activeCmd string
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

	case tui.ClearActiveCmdMsg:
		m.activeCmd = ""
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
		return m, tui.SetActiveCmd(&m.activeCmd, "space")
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
		return m, tui.SetActiveCmd(&m.activeCmd, "enter")
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
		highlightCmd := tui.SetActiveCmd(&m.activeCmd, "1")
		m.typeSelected = moveToShelf
		m, cmd := m.selectMoveType()
		return m, tea.Batch(cmd, highlightCmd)
	case "2":
		highlightCmd := tui.SetActiveCmd(&m.activeCmd, "2")
		m.typeSelected = moveToRelease
		m, cmd := m.selectMoveType()
		return m, tea.Batch(cmd, highlightCmd)
	case "enter":
		highlightCmd := tui.SetActiveCmd(&m.activeCmd, "enter")
		m, cmd := m.selectMoveType()
		return m, tea.Batch(cmd, highlightCmd)
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
		highlightCmd := tui.SetActiveCmd(&m.activeCmd, msg.String())
		m.phase = moveProcessing
		return m, tea.Batch(m.moveAsync(), highlightCmd)
	}

	return m, nil
}

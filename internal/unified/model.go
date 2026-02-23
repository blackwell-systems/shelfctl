package unified

import (
	"github.com/blackwell-systems/shelfctl/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// View represents the current active view
type View string

const (
	ViewHub        View = "hub"
	ViewBrowse     View = "browse"
	ViewShelve     View = "shelve"
	ViewEdit       View = "edit"
	ViewMove       View = "move"
	ViewDelete     View = "delete"
	ViewCacheClear View = "cache-clear"
)

// Model is the unified TUI orchestrator that manages view switching
type Model struct {
	currentView View
	width       int
	height      int

	// View models
	hub HubModel

	// Context passed between views
	hubContext tui.HubContext
}

// New creates a new unified model starting at the hub
func New(ctx tui.HubContext) Model {
	return Model{
		currentView: ViewHub,
		hubContext:  ctx,
		hub:         NewHubModel(ctx),
	}
}

func (m Model) Init() tea.Cmd {
	return m.hub.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Forward to current view
		return m.updateCurrentView(msg)

	case NavigateMsg:
		return m.handleNavigation(msg)

	case QuitAppMsg:
		return m, tea.Quit

	default:
		// Forward to current view
		return m.updateCurrentView(msg)
	}
}

func (m Model) View() string {
	switch m.currentView {
	case ViewHub:
		return m.hub.View()
	case ViewBrowse:
		return "Browse view (not yet implemented)"
	case ViewShelve:
		return "Shelve view (not yet implemented)"
	case ViewEdit:
		return "Edit view (not yet implemented)"
	case ViewMove:
		return "Move view (not yet implemented)"
	case ViewDelete:
		return "Delete view (not yet implemented)"
	case ViewCacheClear:
		return "Cache clear view (not yet implemented)"
	default:
		return "Unknown view"
	}
}

func (m Model) updateCurrentView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.currentView {
	case ViewHub:
		var hubModel tea.Model
		hubModel, cmd = m.hub.Update(msg)
		m.hub = hubModel.(HubModel)
	case ViewBrowse:
		// TODO: forward to browse model
	case ViewShelve:
		// TODO: forward to shelve model
	case ViewEdit:
		// TODO: forward to edit model
	case ViewMove:
		// TODO: forward to move model
	case ViewDelete:
		// TODO: forward to delete model
	case ViewCacheClear:
		// TODO: forward to cache clear model
	}

	return m, cmd
}

func (m Model) handleNavigation(msg NavigateMsg) (tea.Model, tea.Cmd) {
	switch msg.Target {
	case "browse":
		m.currentView = ViewBrowse
		// TODO: initialize browse model
		return m, nil

	case "shelve":
		m.currentView = ViewShelve
		// TODO: initialize shelve model
		return m, nil

	case "edit-book":
		m.currentView = ViewEdit
		// TODO: initialize edit model
		return m, nil

	case "move":
		m.currentView = ViewMove
		// TODO: initialize move model
		return m, nil

	case "delete-book":
		m.currentView = ViewDelete
		// TODO: initialize delete model
		return m, nil

	case "cache-clear":
		m.currentView = ViewCacheClear
		// TODO: initialize cache clear model
		return m, nil

	case "hub":
		// Refresh hub context and return to hub
		m.currentView = ViewHub
		// TODO: refresh hub context from app state
		m.hub = NewHubModel(m.hubContext)
		return m, m.hub.Init()

	case "shelves":
		// Non-TUI command - these still need to run outside unified mode
		// For now, just stay on hub
		// TODO: handle non-TUI commands
		return m, nil

	case "index":
		// Non-TUI command
		// TODO: handle non-TUI commands
		return m, nil

	case "cache-info":
		// Non-TUI command
		// TODO: handle non-TUI commands
		return m, nil

	case "shelve-url":
		// Non-TUI command
		// TODO: handle non-TUI commands
		return m, nil

	case "import-repo":
		// Non-TUI command
		// TODO: handle non-TUI commands
		return m, nil

	case "delete-shelf":
		// Non-TUI command
		// TODO: handle non-TUI commands
		return m, nil

	default:
		// Unknown target, stay on current view
		return m, nil
	}
}

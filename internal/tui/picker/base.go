package picker

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SelectHandler is called when an item is selected.
// Return true to quit the picker, false to continue.
type SelectHandler func(selectedItem list.Item) bool

// KeyHandler is called for custom key handling.
// Return true if the key was handled, false to pass through to default handling.
type KeyHandler func(msg tea.KeyMsg) (handled bool, cmd tea.Cmd)

// Config configures a base picker.
type Config struct {
	// List is the underlying bubbles list.Model
	List list.Model

	// Keys are the key bindings (required)
	QuitKeys   key.Binding
	SelectKeys key.Binding

	// Handlers
	OnSelect     SelectHandler // Called when SelectKeys is pressed
	OnKeyPress   KeyHandler    // Optional: custom key handling
	OnWindowSize func(width, height int)

	// Styling
	BorderStyle lipgloss.Style
	ShowBorder  bool
}

// Base provides common picker functionality.
// Embed this in your picker models to reduce boilerplate.
type Base struct {
	config   Config
	list     list.Model
	quitting bool
	err      error
}

// New creates a new base picker.
func New(cfg Config) *Base {
	return &Base{
		config: cfg,
		list:   cfg.List,
	}
}

// List returns the underlying list model for direct access.
func (b *Base) List() *list.Model {
	return &b.list
}

// IsQuitting returns whether the picker is quitting.
func (b *Base) IsQuitting() bool {
	return b.quitting
}

// Error returns any error that occurred.
func (b *Base) Error() error {
	return b.err
}

// SetError sets an error and marks the picker as quitting.
func (b *Base) SetError(err error) {
	b.err = err
	b.quitting = true
}

// Quit marks the picker as quitting without an error.
func (b *Base) Quit() {
	b.quitting = true
}

// Update handles standard picker updates.
// Call this from your picker's Update method to get standard behavior.
func (b *Base) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keys when filtering
		if b.list.FilterState() == list.Filtering {
			break
		}

		// Custom key handler
		if b.config.OnKeyPress != nil {
			if handled, cmd := b.config.OnKeyPress(msg); handled {
				return cmd
			}
		}

		// Standard key handling
		switch {
		case key.Matches(msg, b.config.QuitKeys):
			b.err = fmt.Errorf("canceled by user")
			b.quitting = true
			return tea.Quit

		case key.Matches(msg, b.config.SelectKeys):
			if b.config.OnSelect != nil {
				selectedItem := b.list.SelectedItem()
				if selectedItem != nil {
					shouldQuit := b.config.OnSelect(selectedItem)
					if shouldQuit {
						b.quitting = true
						return tea.Quit
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		if b.config.OnWindowSize != nil {
			b.config.OnWindowSize(msg.Width, msg.Height)
		} else {
			// Default window size handling
			if b.config.ShowBorder {
				h, v := b.config.BorderStyle.GetFrameSize()
				b.list.SetSize(msg.Width-h, msg.Height-v)
			} else {
				b.list.SetSize(msg.Width, msg.Height)
			}
		}
	}

	// Update the list
	var cmd tea.Cmd
	b.list, cmd = b.list.Update(msg)
	return cmd
}

// View renders the picker.
func (b *Base) View() string {
	if b.quitting {
		return ""
	}

	view := b.list.View()

	if b.config.ShowBorder {
		return b.config.BorderStyle.Render(view)
	}

	return view
}

// SelectedItem returns the currently selected item.
func (b *Base) SelectedItem() list.Item {
	return b.list.SelectedItem()
}

// Items returns all list items.
func (b *Base) Items() []list.Item {
	return b.list.Items()
}

// SetItems sets the list items.
func (b *Base) SetItems(items []list.Item) {
	b.list.SetItems(items)
}

// Title returns the list title.
func (b *Base) Title() string {
	return b.list.Title
}

// SetTitle sets the list title.
func (b *Base) SetTitle(title string) {
	b.list.Title = title
}

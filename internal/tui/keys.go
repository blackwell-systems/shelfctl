package tui

import "github.com/charmbracelet/bubbles/key"

// StandardKeys defines common key bindings used across TUI components.
type StandardKeys struct {
	Quit   key.Binding
	Select key.Binding
	Back   key.Binding
	Toggle key.Binding
	Help   key.Binding
}

// NewStandardKeys creates a standard set of key bindings.
func NewStandardKeys() StandardKeys {
	return StandardKeys{
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("backspace", "h"),
			key.WithHelp("backspace", "back"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// PickerKeys are the standard keys for picker components (list selection).
type PickerKeys struct {
	Quit   key.Binding
	Select key.Binding
}

// NewPickerKeys creates key bindings for picker components.
func NewPickerKeys() PickerKeys {
	std := NewStandardKeys()
	return PickerKeys{
		Quit:   std.Quit,
		Select: std.Select,
	}
}

// ShortHelp returns a slice of key bindings for the short help view.
func (k PickerKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Select, k.Quit}
}

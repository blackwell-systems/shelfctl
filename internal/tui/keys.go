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

// NavigablePickerKeys are keys for pickers with navigation (e.g., file browser).
type NavigablePickerKeys struct {
	Quit   key.Binding
	Select key.Binding
	Back   key.Binding
}

// NewNavigablePickerKeys creates key bindings for navigable pickers.
func NewNavigablePickerKeys() NavigablePickerKeys {
	std := NewStandardKeys()
	return NavigablePickerKeys{
		Quit:   std.Quit,
		Select: std.Select,
		Back:   std.Back,
	}
}

// MultiSelectPickerKeys are keys for pickers with multi-selection.
type MultiSelectPickerKeys struct {
	Quit   key.Binding
	Select key.Binding
	Toggle key.Binding
	Back   key.Binding
}

// NewMultiSelectPickerKeys creates key bindings for multi-select pickers.
func NewMultiSelectPickerKeys() MultiSelectPickerKeys {
	std := NewStandardKeys()
	return MultiSelectPickerKeys{
		Quit:   std.Quit,
		Select: std.Select,
		Toggle: std.Toggle,
		Back:   std.Back,
	}
}

// ShortHelp returns a slice of key bindings for the short help view.
func (k PickerKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Select, k.Quit}
}

// ShortHelp returns a slice of key bindings for the short help view.
func (k NavigablePickerKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Select, k.Back, k.Quit}
}

// ShortHelp returns a slice of key bindings for the short help view.
func (k MultiSelectPickerKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Toggle, k.Select, k.Back, k.Quit}
}

// FormKeys are the standard keys for form components.
type FormKeys struct {
	Quit   key.Binding
	Submit key.Binding
	Next   key.Binding
	Prev   key.Binding
}

// NewFormKeys creates key bindings for form components.
func NewFormKeys() FormKeys {
	std := NewStandardKeys()
	return FormKeys{
		Quit:   std.Quit,
		Submit: std.Select, // Enter to submit
		Next: key.NewBinding(
			key.WithKeys("tab", "down"),
			key.WithHelp("tab", "next field"),
		),
		Prev: key.NewBinding(
			key.WithKeys("shift+tab", "up"),
			key.WithHelp("shift+tab", "previous field"),
		),
	}
}

// ShortHelp returns a slice of key bindings for the short help view.
func (k FormKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Submit, k.Next, k.Prev, k.Quit}
}

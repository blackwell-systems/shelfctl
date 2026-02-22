package delegate

import (
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// RenderFunc is a function that renders a list item.
// It receives the writer, list model, item index, and the item itself.
type RenderFunc func(w io.Writer, m list.Model, index int, item list.Item)

// Base provides a reusable delegate implementation.
// Most delegates only need custom rendering logic - height, spacing, and update
// are typically the same across all delegates.
type Base struct {
	height   int
	spacing  int
	renderFn RenderFunc
}

// New creates a new base delegate with the given render function.
// Uses sensible defaults: height=1, spacing=0, no-op update.
func New(renderFn RenderFunc) Base {
	return Base{
		height:   1,
		spacing:  0,
		renderFn: renderFn,
	}
}

// NewWithSpacing creates a base delegate with custom spacing.
func NewWithSpacing(renderFn RenderFunc, spacing int) Base {
	return Base{
		height:   1,
		spacing:  spacing,
		renderFn: renderFn,
	}
}

// Height implements list.ItemDelegate
func (d Base) Height() int {
	return d.height
}

// Spacing implements list.ItemDelegate
func (d Base) Spacing() int {
	return d.spacing
}

// Update implements list.ItemDelegate
// Most delegates don't need custom update logic, so this returns nil by default.
func (d Base) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// Render implements list.ItemDelegate
func (d Base) Render(w io.Writer, m list.Model, index int, item list.Item) {
	if d.renderFn != nil {
		d.renderFn(w, m, index, item)
	}
}

package unified

import "github.com/blackwell-systems/shelfctl/internal/tui"

// NavigateMsg is emitted when a view wants to navigate to another view
type NavigateMsg struct {
	Target string      // The target view ("browse", "shelve", "hub", etc.)
	Data   interface{} // Optional data to pass to the target view
}

// QuitAppMsg is emitted when the entire application should quit
type QuitAppMsg struct{}

// ActionRequestMsg is emitted when a view wants to perform an action that requires suspending the TUI
type ActionRequestMsg struct {
	Action   tui.BrowserAction
	BookItem *tui.BookItem
	ReturnTo string // Which view to return to after action completes
}

// ShelveRequestMsg is emitted when user wants to add books
type ShelveRequestMsg struct {
	ShelfName string // Optional: pre-selected shelf
	ReturnTo  string // Which view to return to after completion
}

// MoveRequestMsg is emitted when user wants to move books
type MoveRequestMsg struct {
	ReturnTo string // Which view to return to after completion
}

// DeleteRequestMsg is emitted when user wants to delete books
type DeleteRequestMsg struct {
	ReturnTo string // Which view to return to after completion
}

// CacheClearRequestMsg is emitted when user wants to clear cache
type CacheClearRequestMsg struct {
	ReturnTo string // Which view to return to after completion
}

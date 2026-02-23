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

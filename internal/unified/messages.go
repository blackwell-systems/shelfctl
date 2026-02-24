package unified

import "github.com/blackwell-systems/shelfctl/internal/tui"

// NavigateMsg is emitted when a view wants to navigate to another view
type NavigateMsg struct {
	Target   string          // The target view ("browse", "shelve", "hub", etc.)
	Data     interface{}     // Optional data to pass to the target view
	BookItem *tui.BookItem   // Optional single book for direct-edit flows
}

// QuitAppMsg is emitted when the entire application should quit
type QuitAppMsg struct{}

// ActionRequestMsg is emitted when a view wants to perform an action that requires suspending the TUI
type ActionRequestMsg struct {
	Action   tui.BrowserAction
	BookItem *tui.BookItem
	ReturnTo string // Which view to return to after action completes
}

// CommandRequestMsg is emitted when user wants to run a non-TUI command
type CommandRequestMsg struct {
	Command  string // Command name: "shelves", "index", "cache-info", "shelve-url", "import-repo", "delete-shelf"
	ReturnTo string // Which view to return to after completion
}

// CacheClearCompleteMsg is emitted when cache clearing finishes
type CacheClearCompleteMsg struct {
	SuccessCount int
	FailCount    int
}

// CreateShelfCompleteMsg is emitted when shelf creation finishes
type CreateShelfCompleteMsg struct {
	ShelfName string
	RepoName  string
	Err       error
}

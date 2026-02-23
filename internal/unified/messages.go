package unified

// NavigateMsg is emitted when a view wants to navigate to another view
type NavigateMsg struct {
	Target string      // The target view ("browse", "shelve", "hub", etc.)
	Data   interface{} // Optional data to pass to the target view
}

// QuitAppMsg is emitted when the entire application should quit
type QuitAppMsg struct{}

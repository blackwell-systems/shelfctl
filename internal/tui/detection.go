package tui

import (
	"github.com/blackwell-systems/shelfctl/internal/util"
	"github.com/spf13/cobra"
)

// ShouldUseTUI returns true if the command should use interactive TUI mode.
// TUI mode is enabled when:
// - stdout is a TTY (not piped or redirected)
// - --no-interactive flag is not set
// - No output format flags are set (indicates scripting intent)
func ShouldUseTUI(cmd *cobra.Command) bool {
	// Must be running in a terminal
	if !util.IsTTY() {
		return false
	}

	// User explicitly disabled interactive mode
	noInteractive, _ := cmd.Flags().GetBool("no-interactive")
	if noInteractive {
		return false
	}

	// Check for flags that indicate scripting intent
	// If format flag is set, user wants specific output format
	if format, _ := cmd.Flags().GetString("format"); format != "" {
		return false
	}

	return true
}

package util

import (
	"os"

	"github.com/fatih/color"
)

// IsTTY returns true if stdout is a terminal.
func IsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// InitColor configures color output based on flags and terminal detection.
func InitColor(noColor bool) {
	if noColor || !IsTTY() {
		color.NoColor = true
	}
}

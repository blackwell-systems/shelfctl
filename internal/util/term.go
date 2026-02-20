package util

import (
	"os"

	"github.com/fatih/color"
)

func IsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func InitColor(noColor bool) {
	if noColor || !IsTTY() {
		color.NoColor = true
	}
}

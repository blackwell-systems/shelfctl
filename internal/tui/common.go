// Package tui provides interactive terminal user interface components using Bubble Tea.
package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	// ColorGreen for cached items and success indicators
	ColorGreen = lipgloss.AdaptiveColor{Light: "#00AF00", Dark: "#00D700"}

	// ColorWhite for primary text
	ColorWhite = lipgloss.AdaptiveColor{Light: "#262626", Dark: "#FFFFFF"}

	// ColorGray for secondary text and help
	ColorGray = lipgloss.AdaptiveColor{Light: "#767676", Dark: "#808080"}

	// ColorYellow retained for any legacy use
	ColorYellow = lipgloss.AdaptiveColor{Light: "#D7AF00", Dark: "#FFD700"}

	// Brand colors
	ColorOrange    = lipgloss.Color("#fb6820") // "shelf" — primary accent
	ColorTeal      = lipgloss.Color("#1b8487") // "ctl" — secondary accent
	ColorTealLight = lipgloss.Color("#2ecfd4") // teal highlight / tag text
	ColorTealDim   = lipgloss.Color("#0d3536") // teal background chips
)

// Reusable styles
var (
	// StyleNormal is the base style for regular text
	StyleNormal = lipgloss.NewStyle().Foreground(ColorWhite)

	// StyleHighlight is for selected / active items — brand orange
	StyleHighlight = lipgloss.NewStyle().
			Foreground(ColorOrange).
			Bold(true)

	// StyleCached is for cached/downloaded indicators
	StyleCached = lipgloss.NewStyle().Foreground(ColorGreen)

	// StyleTag is for book tags — brand teal-light
	StyleTag = lipgloss.NewStyle().Foreground(ColorTealLight)

	// StyleHelp is for help text and hints
	StyleHelp = lipgloss.NewStyle().Foreground(ColorGray)

	// StyleHeader is for section headers
	StyleHeader = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Bold(true)

	// StyleBorder is for the app outer border
	StyleBorder = lipgloss.NewStyle().
			Foreground(ColorGray).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorTeal)
)

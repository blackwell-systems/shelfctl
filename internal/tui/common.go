package tui

import "github.com/charmbracelet/lipgloss"

// Color palette matching existing fatih/color usage
var (
	// ColorGreen for cached items and success indicators
	ColorGreen = lipgloss.AdaptiveColor{Light: "#00AF00", Dark: "#00D700"}

	// ColorCyan for tags and metadata
	ColorCyan = lipgloss.AdaptiveColor{Light: "#00AFAF", Dark: "#00D7D7"}

	// ColorWhite for primary text
	ColorWhite = lipgloss.AdaptiveColor{Light: "#262626", Dark: "#FFFFFF"}

	// ColorGray for secondary text and help
	ColorGray = lipgloss.AdaptiveColor{Light: "#767676", Dark: "#808080"}

	// ColorYellow for warnings and highlights
	ColorYellow = lipgloss.AdaptiveColor{Light: "#D7AF00", Dark: "#FFD700"}
)

// Reusable styles
var (
	// StyleNormal is the base style for regular text
	StyleNormal = lipgloss.NewStyle().Foreground(ColorWhite)

	// StyleHighlight is for selected items
	StyleHighlight = lipgloss.NewStyle().
			Foreground(ColorYellow).
			Bold(true)

	// StyleCached is for cached/downloaded indicators
	StyleCached = lipgloss.NewStyle().Foreground(ColorGreen)

	// StyleTag is for book tags
	StyleTag = lipgloss.NewStyle().Foreground(ColorCyan)

	// StyleHelp is for help text and hints
	StyleHelp = lipgloss.NewStyle().Foreground(ColorGray)

	// StyleHeader is for section headers
	StyleHeader = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Bold(true)

	// StyleBorder is for borders and separators
	StyleBorder = lipgloss.NewStyle().
			Foreground(ColorGray).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorGray)
)

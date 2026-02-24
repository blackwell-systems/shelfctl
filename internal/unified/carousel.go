package unified

import (
	"fmt"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
)

// updateCarouselView handles key events in the peeking carousel
func (m EditBookModel) updateCarouselView(msg tea.KeyMsg) (EditBookModel, tea.Cmd) {
	n := len(m.toEdit)

	selectCurrent := func() (EditBookModel, tea.Cmd) {
		m.editIndex = m.carouselCursor
		m.inCarousel = false
		m.initFormForBook(m.editIndex)
		return m, textinput.Blink
	}

	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }

	case "esc":
		m.inCarousel = false
		m.initFormForBook(m.editIndex)
		return m, textinput.Blink

	case "left", "h":
		if m.carouselCursor > 0 {
			m.carouselCursor--
		}

	case "right", "l":
		if m.carouselCursor < n-1 {
			m.carouselCursor++
		}

	case "down", "j", "enter", " ":
		return selectCurrent()
	}

	return m, nil
}

// renderCarouselView renders the peeking single-row carousel.
// The active card is shown full-width in the center; adjacent cards peek in
// from the sides showing only their near edge.
func (m EditBookModel) renderCarouselView() string {
	const minPeekW = 8    // minimum peek width on each side
	const gap = 2         // space between peek and center card
	const maxCenterW = 44 // cap card width for a realistic aspect ratio

	usable := m.width - 6 // outer padding from StyleBorder + outerPad
	centerW := usable - 2*(minPeekW+gap)
	if centerW > maxCenterW {
		centerW = maxCenterW
	}
	if centerW < 24 {
		centerW = 24
	}
	// Give leftover space to the peek slots, but cap at half the card width
	// so adjacent cards never show more than half of themselves.
	peekW := (usable - centerW - 2*gap) / 2
	if maxPeek := (centerW + 2) / 2; peekW > maxPeek {
		peekW = maxPeek
	}
	if peekW < minPeekW {
		peekW = minPeekW
	}

	// Derive card content height from width to approximate a 3:5 library card.
	// Terminal chars are ~2:1 (height:width in pixels), so:
	//   rows = (centerW + 2) * 3 / 10   (total visible width * ratio / char_aspect)
	totalCardW := centerW + 2 // include border cols
	cardContentH := totalCardW * 3 / 10
	if cardContentH < 8 {
		cardContentH = 8
	}

	saved := 0
	for _, fs := range m.formStates {
		if fs.saved {
			saved++
		}
	}

	cur := m.carouselCursor
	n := len(m.toEdit)

	// Header: card position indicator
	dots := make([]string, n)
	for i := range dots {
		if i == cur {
			dots[i] = lipgloss.NewStyle().Foreground(tui.ColorOrange).Render("●")
		} else {
			dots[i] = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("○")
		}
	}
	indicator := strings.Join(dots, " ")

	var header strings.Builder
	header.WriteString(tui.StyleHeader.Render("Select a book to edit"))
	header.WriteString("  ")
	header.WriteString(tui.StyleHelp.Render(fmt.Sprintf("%d/%d saved", saved, n)))
	header.WriteString("\n")
	header.WriteString(indicator)
	header.WriteString("\n\n")

	// Render the three slots
	centerCard := m.renderCarouselCard(cur, centerW, cardContentH, true)

	gapBlock := strings.Repeat(" ", gap)

	var leftPeek, rightPeek string
	if cur > 0 {
		rendered := m.renderCarouselCard(cur-1, centerW, cardContentH, false)
		leftPeek = peekRight(rendered, peekW)
	} else {
		leftPeek = peekRight(ghostCard(centerW, cardContentH), peekW)
	}
	if cur < n-1 {
		rendered := m.renderCarouselCard(cur+1, centerW, cardContentH, false)
		rightPeek = peekLeft(rendered, peekW)
	} else {
		rightPeek = peekLeft(ghostCard(centerW, cardContentH), peekW)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top,
		leftPeek, gapBlock, centerCard, gapBlock, rightPeek,
	)

	// Footer
	var navHint string
	switch cur {
	case 0:
		navHint = "→ Navigate"
	case n - 1:
		navHint = "← Navigate"
	default:
		navHint = "←→ Navigate"
	}
	footer := tui.RenderFooterBar([]tui.ShortcutEntry{
		{Key: "tab", Label: navHint},
		{Key: "enter", Label: "Enter/↓ Select"},
		{Key: "", Label: "Esc Back"},
	}, m.activeCmd)

	var b strings.Builder
	b.WriteString(header.String())
	b.WriteString(row)
	b.WriteString("\n\n")
	b.WriteString(footer)

	outerPad := lipgloss.NewStyle().Padding(1, 2)
	return tui.StyleBorder.Render(outerPad.Render(b.String()))
}

// renderCarouselCard renders a single card. active=true uses the orange border.
func (m EditBookModel) renderCarouselCard(i, cardW, cardH int, active bool) string {
	book := m.toEdit[i]
	fs := m.formStates[i]

	inner := cardW - 2 // subtract padding
	titleText := xansi.Truncate(book.Book.Title, inner, "…")
	authorText := xansi.Truncate(book.Book.Author, inner, "…")
	tagsText := xansi.Truncate(strings.Join(book.Book.Tags, ", "), inner, "…")

	var statusText string
	if fs.saved {
		statusText = lipgloss.NewStyle().Foreground(lipgloss.Color("28")).Render("✓ saved")
	} else {
		statusText = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("· unsaved")
	}

	content := titleText + "\n" + authorText + "\n" + tagsText + "\n" + statusText

	var style lipgloss.Style
	switch {
	case active:
		style = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(tui.ColorOrange).
			Width(cardW).Height(cardH).Padding(1, 1)
	case fs.saved:
		style = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("28")).
			Foreground(lipgloss.Color("242")).
			Width(cardW).Height(cardH).Padding(1, 1)
	default:
		style = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Foreground(lipgloss.Color("242")).
			Width(cardW).Height(cardH).Padding(1, 1)
	}

	return style.Render(content)
}

// peekLeft clips a rendered multi-line block to the first n visible columns (left edge peek).
func peekLeft(s string, n int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = xansi.Truncate(line, n, "")
	}
	return strings.Join(lines, "\n")
}

// peekRight clips a rendered multi-line block to the last n visible columns (right edge peek).
func peekRight(s string, n int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		w := xansi.StringWidth(line)
		if w > n {
			lines[i] = xansi.TruncateLeft(line, w-n, "")
		}
	}
	return strings.Join(lines, "\n")
}

// ghostCard renders a blank card with a very dim border — used at the edges
// of the carousel to maintain visual rhythm when there's no adjacent card.
func ghostCard(cardW, cardH int) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("235")).
		Width(cardW).Height(cardH).
		Render("")
}

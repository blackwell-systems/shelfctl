package unified

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/charmbracelet/lipgloss"
)

func (m ShelveModel) View() string {
	if m.empty {
		return m.renderMessage("No shelves configured", "Run 'shelfctl init' first\n\nPress Enter to return to menu")
	}

	if m.err != nil && m.phase != shelveForm {
		return m.renderMessage(fmt.Sprintf("Error: %v", m.err), "Press Enter to return to menu")
	}

	switch m.phase {
	case shelveShelfPicking:
		return m.renderShelfPicker()

	case shelveURLInput:
		return m.renderURLInput()

	case shelveFilePicking:
		return m.filePicker.View()

	case shelveSetup:
		return m.renderMessage("Preparing...", fmt.Sprintf("Loading catalog and release for shelf %q", m.shelfName))

	case shelveIngesting:
		filename := filepath.Base(m.selectedFiles[m.fileIndex])
		if len(m.selectedFiles) > 1 {
			return m.renderMessage(
				fmt.Sprintf("Ingesting file [%d/%d]", m.fileIndex+1, len(m.selectedFiles)),
				fmt.Sprintf("Processing %s...", filename),
			)
		}
		return m.renderMessage("Ingesting file", fmt.Sprintf("Processing %s...", filename))

	case shelveForm:
		return m.renderForm()

	case shelveUploading:
		return m.renderUploadProgress()

	case shelveCommitting:
		return m.renderMessage("Committing changes...", fmt.Sprintf("Added %d book(s) to shelf %q", m.successCount, m.shelfName))

	default:
		return ""
	}
}

func (m ShelveModel) renderMessage(title, help string) string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render(title))
	b.WriteString("\n\n")
	b.WriteString(tui.StyleHelp.Render(help))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

func (m ShelveModel) renderShelfPicker() string {
	return tui.StyleBorder.Render(m.shelfList.View())
}

func (m ShelveModel) renderURLInput() string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render("Add Book from URL"))
	b.WriteString("\n\n")

	b.WriteString(tui.StyleNormal.Render("URL:"))
	b.WriteString("\n  ")
	b.WriteString(m.urlInput.View())
	b.WriteString("\n\n")

	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	b.WriteString(tui.RenderFooterBar([]tui.ShortcutEntry{
		{Key: "enter", Label: "Enter Submit"},
		{Key: "", Label: "Esc Back"},
	}, m.activeCmd))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

func (m ShelveModel) renderForm() string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder

	// Header
	b.WriteString(tui.StyleHeader.Render("Add Book to Library"))
	b.WriteString("\n\n")

	// File info with progress for multi-file
	filename := m.formDefaults.filename
	if len(m.selectedFiles) > 1 {
		filename = fmt.Sprintf("[%d/%d] %s", m.fileIndex+1, len(m.selectedFiles), filename)
	}
	const maxFilenameWidth = 60
	if len(filename) > maxFilenameWidth {
		filename = filename[:maxFilenameWidth-1] + "…"
	}
	b.WriteString(tui.StyleHelp.Render(fmt.Sprintf("File: %s", filename)))
	b.WriteString("\n")
	b.WriteString(tui.StyleHelp.Render(fmt.Sprintf("Shelf: %s", m.shelfName)))
	b.WriteString("\n\n")

	// Error display
	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	// Form fields
	fields := []string{"Title", "Author", "Tags", "ID"}
	for i, label := range fields {
		if i == m.focused {
			b.WriteString(tui.StyleHighlight.Render("> " + label + ":"))
		} else {
			b.WriteString(tui.StyleNormal.Render("  " + label + ":"))
		}
		b.WriteString("\n  ")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}

	// Cache checkbox
	checkboxLabel := "Cache locally"
	checkbox := "[ ]"
	if m.cacheLocally {
		checkbox = "[✓]"
	}
	if m.focused == shelveFieldCache {
		b.WriteString(tui.StyleHighlight.Render("> " + checkboxLabel + ": " + checkbox))
	} else {
		b.WriteString(tui.StyleNormal.Render("  " + checkboxLabel + ": " + checkbox))
	}
	b.WriteString("\n")
	b.WriteString(tui.StyleHelp.Render("  (Space to toggle)"))
	b.WriteString("\n")

	// Help text
	b.WriteString("\n")
	b.WriteString(tui.RenderFooterBar([]tui.ShortcutEntry{
		{Key: "tab", Label: "Tab/↑↓ Navigate"},
		{Key: "space", Label: "Space Toggle"},
		{Key: "enter", Label: "Enter Submit"},
		{Key: "", Label: "Esc Cancel"},
	}, m.activeCmd))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

func (m ShelveModel) renderUploadProgress() string {
	style := lipgloss.NewStyle().Padding(2, 4)

	var b strings.Builder

	// Header
	filename := filepath.Base(m.selectedFiles[m.fileIndex])
	if len(m.selectedFiles) > 1 {
		b.WriteString(tui.StyleHeader.Render(fmt.Sprintf("Processing [%d/%d]: %s", m.fileIndex+1, len(m.selectedFiles), filename)))
	} else {
		b.WriteString(tui.StyleHeader.Render(fmt.Sprintf("Processing: %s", filename)))
	}
	b.WriteString("\n\n")

	// Status
	b.WriteString(tui.StyleNormal.Render(m.uploadStatus))
	b.WriteString("\n\n")

	// Progress bar (only during actual upload)
	if m.uploadTotal > 0 && m.uploadStatus == "Uploading..." {
		barWidth := 40
		filled := int(m.uploadProgress * float64(barWidth))
		if filled > barWidth {
			filled = barWidth
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		progressStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
		b.WriteString(progressStyle.Render(bar))
		b.WriteString("\n")

		// Bytes display
		currentMB := float64(m.uploadBytes) / 1024 / 1024
		totalMB := float64(m.uploadTotal) / 1024 / 1024
		b.WriteString(tui.StyleHelp.Render(fmt.Sprintf("%.2f MB / %.2f MB (%.0f%%)", currentMB, totalMB, m.uploadProgress*100)))
		b.WriteString("\n")
	}

	// Summary so far (for multi-file)
	if len(m.selectedFiles) > 1 && (m.successCount > 0 || m.failCount > 0) {
		b.WriteString("\n")
		if m.successCount > 0 {
			b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("Completed: %d", m.successCount)))
			b.WriteString("\n")
		}
		if m.failCount > 0 {
			errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			b.WriteString(errStyle.Render(fmt.Sprintf("Failed: %d", m.failCount)))
			b.WriteString("\n")
		}
	}

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

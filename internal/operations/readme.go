package operations

import (
	"fmt"
	"strings"
	"time"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
)

// UpdateShelfREADMEStats updates the stats section of an existing README
func UpdateShelfREADMEStats(existingREADME string, bookCount int) string {
	now := time.Now().Format("2006-01-02")

	// Find and replace the Quick Stats section
	lines := strings.Split(existingREADME, "\n")
	var result []string
	inStatsSection := false
	statsReplaced := false

	for i, line := range lines {
		if strings.HasPrefix(line, "## Quick Stats") {
			inStatsSection = true
			result = append(result, line)
			result = append(result, "")
			result = append(result, fmt.Sprintf("- **Books**: %d", bookCount))
			result = append(result, fmt.Sprintf("- **Last Updated**: %s", now))
			statsReplaced = true
			continue
		}

		if inStatsSection {
			// Skip old stats lines until next section
			if strings.HasPrefix(line, "##") {
				inStatsSection = false
				result = append(result, "")
				result = append(result, line)
			}
			continue
		}

		// Keep lines outside stats section
		result = append(result, line)

		// If we're at the end and never found stats section, don't add anything
		if i == len(lines)-1 && !statsReplaced {
			return existingREADME // Return unchanged if no stats section found
		}
	}

	return strings.Join(result, "\n")
}

// AppendToShelfREADME adds a new book entry to a "Recently Added" section
func AppendToShelfREADME(existingREADME string, book catalog.Book) string {
	// Find "## Recently Added" section or create it after "## Quick Stats"
	lines := strings.Split(existingREADME, "\n")

	// Check if "Recently Added" section exists
	recentlyAddedIdx := -1
	quickStatsIdx := -1

	for i, line := range lines {
		if strings.HasPrefix(line, "## Recently Added") {
			recentlyAddedIdx = i
			break
		}
		if strings.HasPrefix(line, "## Quick Stats") {
			quickStatsIdx = i
		}
	}

	bookEntry := fmt.Sprintf("- **%s** by %s (`%s`)", book.Title, book.Author, book.ID)
	if len(book.Tags) > 0 {
		bookEntry += fmt.Sprintf(" - Tags: %s", strings.Join(book.Tags, ", "))
	}

	var result []string

	if recentlyAddedIdx >= 0 {
		// Section exists, add to it and keep last 10 entries
		// First, collect existing entries (excluding current book to avoid duplicates)
		var existingEntries []string
		nextSectionIdx := len(lines)

		// Find the end of Recently Added section
		for i := recentlyAddedIdx + 1; i < len(lines); i++ {
			if strings.HasPrefix(lines[i], "##") {
				nextSectionIdx = i
				break
			}
			// Check if this is a book entry line (starts with "- ")
			if strings.HasPrefix(lines[i], "- ") {
				// Extract book ID from entry to check for duplicates
				// Format: "- **Title** by Author (`book-id`)"
				if !strings.Contains(lines[i], fmt.Sprintf("(`%s`)", book.ID)) {
					existingEntries = append(existingEntries, lines[i])
				}
			}
		}

		// Keep only last 9 existing entries (so with new entry we have 10 total)
		const maxEntries = 10
		if len(existingEntries) > maxEntries-1 {
			existingEntries = existingEntries[:maxEntries-1]
		}

		// Rebuild the README
		// Copy lines before Recently Added section
		for i := 0; i <= recentlyAddedIdx; i++ {
			result = append(result, lines[i])
		}

		// Add new entry at the top
		result = append(result, "")
		result = append(result, bookEntry)

		// Add existing entries
		result = append(result, existingEntries...)

		// Add lines after Recently Added section
		if nextSectionIdx < len(lines) {
			result = append(result, "")
			result = append(result, lines[nextSectionIdx:]...)
		}
	} else if quickStatsIdx >= 0 {
		// Create new section after Quick Stats
		for i, line := range lines {
			result = append(result, line)
			if i == quickStatsIdx {
				// Find end of Quick Stats section
				for j := i + 1; j < len(lines); j++ {
					result = append(result, lines[j])
					if strings.HasPrefix(lines[j], "##") {
						// Found next section, insert before it
						result = result[:len(result)-1] // Remove the section header we just added
						result = append(result, "")
						result = append(result, "## Recently Added")
						result = append(result, "")
						result = append(result, bookEntry)
						result = append(result, "")
						result = append(result, lines[j]) // Add back the section header
						// Add rest of lines
						result = append(result, lines[j+1:]...)
						return strings.Join(result, "\n")
					}
				}
			}
		}
	} else {
		// No Quick Stats section found, just append at end
		result = lines
		result = append(result, "")
		result = append(result, "## Recently Added")
		result = append(result, "")
		result = append(result, bookEntry)
	}

	return strings.Join(result, "\n")
}

// RemoveFromShelfREADME removes a book entry from the "Recently Added" section
func RemoveFromShelfREADME(existingREADME string, bookID string) string {
	lines := strings.Split(existingREADME, "\n")

	// Find "## Recently Added" section
	recentlyAddedIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "## Recently Added") {
			recentlyAddedIdx = i
			break
		}
	}

	if recentlyAddedIdx < 0 {
		return existingREADME // No section to remove from
	}

	var result []string
	inRecentlyAdded := false
	nextSectionIdx := len(lines)

	// Find the end of Recently Added section
	for i := recentlyAddedIdx + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "##") {
			nextSectionIdx = i
			break
		}
	}

	// Rebuild README, filtering out the book entry
	for i, line := range lines {
		if i == recentlyAddedIdx {
			inRecentlyAdded = true
			result = append(result, line)
			continue
		}

		if inRecentlyAdded && i >= nextSectionIdx {
			inRecentlyAdded = false
		}

		// Skip the book entry line
		if inRecentlyAdded && strings.Contains(line, fmt.Sprintf("(`%s`)", bookID)) {
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

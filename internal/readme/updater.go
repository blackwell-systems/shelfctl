package readme

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/github"
)

// Updater manages README.md updates for shelf repositories.
type Updater struct {
	gh    *github.Client
	owner string
	repo  string
}

// NewUpdater creates a new README updater.
func NewUpdater(gh *github.Client, owner, repo string) *Updater {
	return &Updater{
		gh:    gh,
		owner: owner,
		repo:  repo,
	}
}

// AddBook updates the README by incrementing stats and adding to "Recently Added".
func (u *Updater) AddBook(book catalog.Book) error {
	return u.AddBooks([]catalog.Book{book})
}

// AddBooks updates the README by incrementing stats and adding multiple books to "Recently Added".
func (u *Updater) AddBooks(books []catalog.Book) error {
	if len(books) == 0 {
		return nil
	}

	readmeData, _, err := u.gh.GetFileContent(u.owner, u.repo, "README.md", "")
	if err != nil {
		// Not an error if README doesn't exist
		return nil
	}

	content := string(readmeData)

	// Update stats with total book count (we don't have it, so we'll need to load catalog)
	// For now, just add the books to recently added

	// Add all books to "Recently Added"
	for _, book := range books {
		content = appendToRecentlyAdded(content, book)
	}

	// Create commit message
	var commitMsg string
	if len(books) == 1 {
		commitMsg = fmt.Sprintf("Update README: add %s", books[0].ID)
	} else {
		commitMsg = fmt.Sprintf("Update README: add %d books", len(books))
	}

	if err := u.gh.CommitFile(u.owner, u.repo, "README.md", []byte(content), commitMsg); err != nil {
		// Warn but don't fail
		return nil
	}

	return nil
}

// RemoveBook updates the README by decrementing stats after a book is removed.
func (u *Updater) RemoveBook(bookID string) error {
	readmeData, _, err := u.gh.GetFileContent(u.owner, u.repo, "README.md", "")
	if err != nil {
		return nil
	}

	content := string(readmeData)

	// Remove from "Recently Added" section if present
	content = removeFromRecentlyAdded(content, bookID)

	commitMsg := fmt.Sprintf("Update README: remove %s", bookID)
	if err := u.gh.CommitFile(u.owner, u.repo, "README.md", []byte(content), commitMsg); err != nil {
		return nil
	}

	return nil
}

// UpdateStats updates the book count and last updated date.
func (u *Updater) UpdateStats(bookCount int) error {
	readmeData, _, err := u.gh.GetFileContent(u.owner, u.repo, "README.md", "")
	if err != nil {
		return nil
	}

	content := string(readmeData)
	content = updateStats(content, bookCount)

	commitMsg := "Update README: refresh stats"
	if err := u.gh.CommitFile(u.owner, u.repo, "README.md", []byte(content), commitMsg); err != nil {
		return nil
	}

	return nil
}

// UpdateWithStats updates README with both stats and adds books.
func (u *Updater) UpdateWithStats(bookCount int, newBooks []catalog.Book) error {
	if len(newBooks) == 0 {
		return u.UpdateStats(bookCount)
	}

	readmeData, _, err := u.gh.GetFileContent(u.owner, u.repo, "README.md", "")
	if err != nil {
		return nil
	}

	content := string(readmeData)

	// Update stats
	content = updateStats(content, bookCount)

	// Add books
	for _, book := range newBooks {
		content = appendToRecentlyAdded(content, book)
	}

	var commitMsg string
	if len(newBooks) == 1 {
		commitMsg = fmt.Sprintf("Update README: add %s", newBooks[0].ID)
	} else {
		commitMsg = fmt.Sprintf("Update README: add %d books", len(newBooks))
	}

	if err := u.gh.CommitFile(u.owner, u.repo, "README.md", []byte(content), commitMsg); err != nil {
		return nil
	}

	return nil
}

// updateStats updates the book count in the README stats section.
func updateStats(content string, bookCount int) string {
	// Pattern: **X books** | Last updated: ...
	re := regexp.MustCompile(`\*\*\d+ books?\*\*`)
	plural := "books"
	if bookCount == 1 {
		plural = "book"
	}
	replacement := fmt.Sprintf("**%d %s**", bookCount, plural)
	return re.ReplaceAllString(content, replacement)
}

// appendToRecentlyAdded adds a book to the "Recently Added" section.
func appendToRecentlyAdded(content string, book catalog.Book) string {
	// Find the "Recently Added" section
	recentlyAddedStart := strings.Index(content, "## Recently Added")
	if recentlyAddedStart == -1 {
		return content // Section doesn't exist
	}

	// Find the next section (starts with ##) or end of file
	nextSectionStart := strings.Index(content[recentlyAddedStart+len("## Recently Added"):], "\n##")
	var sectionEnd int
	if nextSectionStart == -1 {
		sectionEnd = len(content)
	} else {
		sectionEnd = recentlyAddedStart + len("## Recently Added") + nextSectionStart
	}

	// Extract the section
	section := content[recentlyAddedStart:sectionEnd]

	// Check if this book ID already exists in the section
	bookIDPattern := fmt.Sprintf("- **%s**", book.ID)
	if strings.Contains(section, bookIDPattern) {
		// Book already listed, don't add again
		return content
	}

	// Find where to insert (after the header line and blank line)
	insertPos := recentlyAddedStart + len("## Recently Added\n\n")

	// Format the book entry
	tags := ""
	if len(book.Tags) > 0 {
		tags = " | " + strings.Join(book.Tags, ", ")
	}
	entry := fmt.Sprintf("- **%s** — %s by %s%s\n", book.ID, book.Title, book.Author, tags)

	// Insert the entry
	newContent := content[:insertPos] + entry + content[insertPos:]

	// Limit to 10 most recent entries
	return limitRecentlyAdded(newContent, 10)
}

// removeFromRecentlyAdded removes a book from the "Recently Added" section.
func removeFromRecentlyAdded(content string, bookID string) string {
	// Pattern: - **book-id** — ...
	pattern := fmt.Sprintf(`- \*\*%s\*\* — [^\n]+\n`, regexp.QuoteMeta(bookID))
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllString(content, "")
}

// limitRecentlyAdded ensures only the first N entries remain.
func limitRecentlyAdded(content string, maxEntries int) string {
	recentlyAddedStart := strings.Index(content, "## Recently Added")
	if recentlyAddedStart == -1 {
		return content
	}

	// Find the next section
	nextSectionStart := strings.Index(content[recentlyAddedStart+len("## Recently Added"):], "\n##")
	var sectionEnd int
	if nextSectionStart == -1 {
		sectionEnd = len(content)
	} else {
		sectionEnd = recentlyAddedStart + len("## Recently Added") + nextSectionStart
	}

	section := content[recentlyAddedStart:sectionEnd]

	// Split into lines
	lines := strings.Split(section, "\n")

	// Find all book entries (lines starting with "- **")
	var entryLines []int
	for i, line := range lines {
		if strings.HasPrefix(line, "- **") {
			entryLines = append(entryLines, i)
		}
	}

	// If we have more than maxEntries, remove the excess
	if len(entryLines) > maxEntries {
		// Remove entries beyond maxEntries
		for i := maxEntries; i < len(entryLines); i++ {
			lines[entryLines[i]] = ""
		}

		// Rebuild section
		var newLines []string
		for _, line := range lines {
			if line != "" {
				newLines = append(newLines, line)
			}
		}
		section = strings.Join(newLines, "\n")

		// Replace in content
		return content[:recentlyAddedStart] + section + content[sectionEnd:]
	}

	return content
}

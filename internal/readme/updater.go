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

// UpdateWithStats updates README with both stats and adds books.
func (u *Updater) UpdateWithStats(bookCount int, newBooks []catalog.Book) error {
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
	if len(newBooks) == 0 {
		commitMsg = "Update README: refresh stats"
	} else if len(newBooks) == 1 {
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

	return newContent
}

// removeFromRecentlyAdded removes a book from the "Recently Added" section.
func removeFromRecentlyAdded(content string, bookID string) string {
	// Pattern: - **book-id** — ...
	pattern := fmt.Sprintf(`- \*\*%s\*\* — [^\n]+\n`, regexp.QuoteMeta(bookID))
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllString(content, "")
}

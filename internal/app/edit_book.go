package app

import (
	"fmt"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/spf13/cobra"
)

func newEditBookCmd() *cobra.Command {
	var (
		shelfName string
		title     string
		author    string
		year      int
		addTags   string
		rmTags    string
	)

	cmd := &cobra.Command{
		Use:   "edit-book [id]",
		Short: "Edit metadata for a book",
		Long: `Edit metadata for a book in your library.

You can edit: title, author, year, and tags.
You cannot edit: ID, format, checksum, or asset (these are tied to the file).

Examples:
  shelfctl edit-book design-patterns                    # Interactive form
  shelfctl edit-book design-patterns --title "New Title"
  shelfctl edit-book gopl --author "Donovan & Kernighan" --year 2015
  shelfctl edit-book sicp --add-tag favorites --add-tag classics
  shelfctl edit-book sicp --rm-tag draft`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var bookID string

			// Interactive mode: pick a book
			if len(args) == 0 {
				if !tui.ShouldUseTUI(cmd) {
					return fmt.Errorf("book ID required in non-interactive mode")
				}

				// Collect all books
				var allItems []tui.BookItem
				for i := range cfg.Shelves {
					shelf := &cfg.Shelves[i]
					if shelfName != "" && shelf.Name != shelfName {
						continue
					}

					owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
					catalogPath := shelf.EffectiveCatalogPath()

					data, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
					if err != nil {
						continue
					}
					books, err := catalog.Parse(data)
					if err != nil {
						continue
					}

					for j := range books {
						b := &books[j]
						cached := cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset)
						allItems = append(allItems, tui.BookItem{
							Book:      *b,
							ShelfName: shelf.Name,
							Cached:    cached,
							Owner:     owner,
							Repo:      shelf.Repo,
						})
					}
				}

				if len(allItems) == 0 {
					return fmt.Errorf("no books found")
				}

				result, err := tui.RunBookPicker(allItems, "Select a book to edit")
				if err != nil {
					return err
				}

				bookID = result.Book.ID
			} else {
				bookID = args[0]
			}

			// Find the book
			book, shelf, err := findBook(bookID, shelfName)
			if err != nil {
				return err
			}
			owner := shelf.EffectiveOwner(cfg.GitHub.Owner)

			// Determine which mode: interactive form or CLI flags
			useTUI := tui.ShouldUseTUI(cmd) && title == "" && author == "" && year == 0 && addTags == "" && rmTags == ""

			var updatedBook catalog.Book
			if useTUI {
				// Interactive form mode
				defaults := tui.EditFormDefaults{
					BookID: book.ID,
					Title:  book.Title,
					Author: book.Author,
					Year:   book.Year,
					Tags:   book.Tags,
				}

				formData, err := tui.RunEditForm(defaults)
				if err != nil {
					return err
				}

				// Parse tags
				tags := []string{}
				if formData.Tags != "" {
					for _, t := range strings.Split(formData.Tags, ",") {
						t = strings.TrimSpace(t)
						if t != "" {
							tags = append(tags, t)
						}
					}
				}

				// Build updated book
				updatedBook = *book
				updatedBook.Title = formData.Title
				updatedBook.Author = formData.Author
				updatedBook.Year = formData.Year
				updatedBook.Tags = tags

			} else {
				// CLI flag mode
				updatedBook = *book

				// Apply flag changes
				if title != "" {
					updatedBook.Title = title
				}
				if author != "" {
					updatedBook.Author = author
				}
				if year > 0 {
					updatedBook.Year = year
				}

				// Handle tag modifications
				if addTags != "" || rmTags != "" {
					tagSet := make(map[string]bool)
					for _, t := range book.Tags {
						tagSet[t] = true
					}

					// Add new tags
					if addTags != "" {
						for _, t := range strings.Split(addTags, ",") {
							t = strings.TrimSpace(t)
							if t != "" {
								tagSet[t] = true
							}
						}
					}

					// Remove tags
					if rmTags != "" {
						for _, t := range strings.Split(rmTags, ",") {
							t = strings.TrimSpace(t)
							if t != "" {
								delete(tagSet, t)
							}
						}
					}

					// Rebuild tags slice
					tags := []string{}
					for t := range tagSet {
						tags = append(tags, t)
					}
					updatedBook.Tags = tags
				}
			}

			// Load catalog
			catalogPath := shelf.EffectiveCatalogPath()
			data, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			books, err := catalog.Parse(data)
			if err != nil {
				return fmt.Errorf("parsing catalog: %w", err)
			}

			// Update the book (Append replaces if ID exists)
			books = catalog.Append(books, updatedBook)

			// Marshal back to YAML
			updatedData, err := catalog.Marshal(books)
			if err != nil {
				return fmt.Errorf("marshaling catalog: %w", err)
			}

			// Commit changes
			commitMsg := fmt.Sprintf("edit: update metadata for %s", bookID)
			if err := gh.CommitFile(owner, shelf.Repo, catalogPath, updatedData, commitMsg); err != nil {
				return fmt.Errorf("committing catalog: %w", err)
			}

			// Show summary
			fmt.Println()
			header("Book Updated")
			printField("id", updatedBook.ID)
			printField("title", updatedBook.Title)
			if updatedBook.Author != "" {
				printField("author", updatedBook.Author)
			}
			if updatedBook.Year > 0 {
				printField("year", fmt.Sprintf("%d", updatedBook.Year))
			}
			if len(updatedBook.Tags) > 0 {
				printField("tags", strings.Join(updatedBook.Tags, ", "))
			}
			printField("shelf", shelf.Name)
			fmt.Println()

			ok("Metadata updated in catalog")
			return nil
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Specify shelf if ID is ambiguous")
	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&author, "author", "", "New author")
	cmd.Flags().IntVar(&year, "year", 0, "Publication year")
	cmd.Flags().StringVar(&addTags, "add-tag", "", "Add tags (comma-separated)")
	cmd.Flags().StringVar(&rmTags, "rm-tag", "", "Remove tags (comma-separated)")

	return cmd
}

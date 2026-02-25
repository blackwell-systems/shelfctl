package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/operations"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/fatih/color"
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

In TUI mode (no ID provided), you can select multiple books using checkboxes:
  • Spacebar to toggle selection
  • Enter to confirm
  • If no checkboxes selected, edits the current book

Examples:
  shelfctl edit-book                                # Interactive multi-select
  shelfctl edit-book design-patterns                # Interactive form
  shelfctl edit-book design-patterns --title "New Title"
  shelfctl edit-book gopl --author "Donovan & Kernighan" --year 2015
  shelfctl edit-book sicp --add-tag favorites --add-tag classics
  shelfctl edit-book sicp --rm-tag draft`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var booksToEdit []tui.BookItem

			// Interactive mode: pick book(s) with multi-select
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

				// Use multi-select picker
				selected, err := tui.RunBookPickerMulti(allItems, "Select books to edit")
				if err != nil {
					return err
				}

				booksToEdit = selected
			} else {
				// CLI mode: single book ID provided
				bookID := args[0]

				// Find the book
				book, foundShelf, err := findBook(bookID, shelfName)
				if err != nil {
					return err
				}
				owner := foundShelf.EffectiveOwner(cfg.GitHub.Owner)
				cached := cacheMgr.Exists(owner, foundShelf.Repo, book.ID, book.Source.Asset)

				booksToEdit = []tui.BookItem{{
					Book:      *book,
					ShelfName: foundShelf.Name,
					Cached:    cached,
					Owner:     owner,
					Repo:      foundShelf.Repo,
				}}
			}

			// Show confirmation for batch edits
			if len(booksToEdit) > 1 {
				fmt.Println()
				fmt.Printf("You are about to edit %d books:\n", len(booksToEdit))
				for _, item := range booksToEdit {
					fmt.Printf("  • %s - %s [%s]\n",
						color.WhiteString(item.Book.ID),
						item.Book.Title,
						color.CyanString(item.ShelfName))
				}
				fmt.Println()
				fmt.Printf("Type 'UPDATE %d BOOKS' to confirm: ", len(booksToEdit))
				reader := bufio.NewReader(os.Stdin)
				confirmation, _ := reader.ReadString('\n')
				confirmation = strings.TrimSpace(confirmation)
				expected := fmt.Sprintf("UPDATE %d BOOKS", len(booksToEdit))
				if confirmation != expected {
					return fmt.Errorf("cancelled")
				}
			}

			// Determine which mode: interactive form or CLI flags
			useTUI := tui.ShouldUseTUI(cmd) && title == "" && author == "" && year == 0 && addTags == "" && rmTags == ""

			// Group books by shelf for batch commit optimization
			booksByShelf := make(map[string][]tui.BookItem)
			for _, item := range booksToEdit {
				booksByShelf[item.ShelfName] = append(booksByShelf[item.ShelfName], item)
			}

			successCount := 0
			failCount := 0
			allUpdatedBooks := make(map[string]catalog.Book) // Track updates by book ID

			// Process each shelf's books together
			for _, shelfBooks := range booksByShelf {
				shelf := cfg.ShelfByName(shelfBooks[0].ShelfName)
				if shelf == nil {
					warn("Shelf %q not found, skipping %d books", shelfBooks[0].ShelfName, len(shelfBooks))
					failCount += len(shelfBooks)
					continue
				}

				owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
				catalogPath := shelf.EffectiveCatalogPath()

				// Load catalog once for this shelf
				catalogData, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
				if err != nil {
					warn("Could not load catalog for shelf %s: %v", shelf.Name, err)
					failCount += len(shelfBooks)
					continue
				}
				books, err := catalog.Parse(catalogData)
				if err != nil {
					warn("Could not parse catalog for shelf %s: %v", shelf.Name, err)
					failCount += len(shelfBooks)
					continue
				}

				var updatedBooks []catalog.Book
				catalogModified := false

				// Edit each book in this shelf
				for _, item := range shelfBooks {
					if len(booksToEdit) > 1 {
						fmt.Printf("\n[%d/%d] Editing %s\n", successCount+failCount+1, len(booksToEdit), item.Book.ID)
					}

					var updatedBook catalog.Book
					b := &item.Book

					if useTUI {
						// Interactive form mode
						defaults := tui.EditFormDefaults{
							BookID: b.ID,
							Title:  b.Title,
							Author: b.Author,
							Year:   b.Year,
							Tags:   b.Tags,
						}

						formData, err := tui.RunEditForm(defaults)
						if err != nil {
							warn("Failed to edit %s: %v", item.Book.ID, err)
							failCount++
							continue
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
						updatedBook = *b
						updatedBook.Title = formData.Title
						updatedBook.Author = formData.Author
						updatedBook.Year = formData.Year
						updatedBook.Tags = tags

					} else {
						// CLI flag mode
						updatedBook = *b

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
							for _, t := range b.Tags {
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

					// Update book in catalog
					books = catalog.Append(books, updatedBook)
					updatedBooks = append(updatedBooks, updatedBook)
					allUpdatedBooks[updatedBook.ID] = updatedBook
					catalogModified = true
					successCount++
				}

				// Commit catalog once for this shelf if modified
				if catalogModified {
					updatedData, err := catalog.Marshal(books)
					if err != nil {
						warn("Could not marshal catalog for shelf %s: %v", shelf.Name, err)
						continue
					}

					commitMsg := fmt.Sprintf("edit: update %d books", len(shelfBooks))
					if len(shelfBooks) == 1 {
						commitMsg = fmt.Sprintf("edit: update metadata for %s", shelfBooks[0].Book.ID)
					}

					if err := gh.CommitFile(owner, shelf.Repo, catalogPath, updatedData, commitMsg); err != nil {
						warn("Could not commit catalog for shelf %s: %v", shelf.Name, err)
						continue
					}

					// Update README with new metadata
					readmeData, _, readmeErr := gh.GetFileContent(owner, shelf.Repo, "README.md", "")
					if readmeErr == nil {
						originalContent := string(readmeData)
						readmeContent := operations.UpdateShelfREADMEStats(originalContent, len(books))

						// Update all modified books in README
						for _, book := range updatedBooks {
							readmeContent = operations.AppendToShelfREADME(readmeContent, book)
						}

						// Only commit if content actually changed
						if readmeContent != originalContent {
							readmeMsg := fmt.Sprintf("Update README: edit %d books", len(updatedBooks))
							if len(updatedBooks) == 1 {
								readmeMsg = fmt.Sprintf("Update README: edit %s", updatedBooks[0].ID)
							}
							if err := gh.CommitFile(owner, shelf.Repo, "README.md", []byte(readmeContent), readmeMsg); err != nil {
								warn("Could not update README.md: %v", err)
							}
						}
					}
				}
			}

			// Summary
			fmt.Println()
			if len(booksToEdit) == 1 {
				if successCount == 1 {
					// Show detailed info for single book
					originalBook := booksToEdit[0].Book
					updatedBook, found := allUpdatedBooks[originalBook.ID]
					if found {
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
						printField("shelf", booksToEdit[0].ShelfName)
						fmt.Println()
					}
					ok("Metadata updated in catalog")
				}
			} else {
				if successCount > 0 {
					ok("Successfully updated %d books", successCount)
				}
				if failCount > 0 {
					warn("%d books failed to update", failCount)
				}
			}

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

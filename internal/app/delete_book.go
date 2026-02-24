package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/blackwell-systems/shelfctl/internal/util"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newDeleteBookCmd() *cobra.Command {
	var (
		shelfName   string
		skipConfirm bool
	)

	cmd := &cobra.Command{
		Use:   "delete-book [id]",
		Short: "Remove book(s) from your library",
		Long: `Remove one or more books from your library by deleting metadata and files.

This command:
  • Removes book entries from catalog.yml
  • Deletes PDF/EPUB files from GitHub Release assets
  • Pushes the updated catalog

This action is DESTRUCTIVE and cannot be easily undone.

In TUI mode (no ID provided), you can select multiple books using checkboxes:
  • Spacebar to toggle selection
  • Enter to confirm
  • If no checkboxes selected, deletes the current book

Examples:
  # Interactive multi-select picker
  shelfctl delete-book

  # Delete specific book (CLI mode)
  shelfctl delete-book sicp --shelf programming

  # Skip confirmation prompt
  shelfctl delete-book sicp --shelf programming --yes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var booksToDelete []tui.BookItem
			successCount := 0
			failCount := 0

			// If no book ID provided, show interactive multi-select picker
			if len(args) == 0 {
				if !util.IsTTY() {
					return fmt.Errorf("book ID required in non-interactive mode")
				}

				// Collect all books for picker
				var allItems []tui.BookItem
				for i := range cfg.Shelves {
					shelf := &cfg.Shelves[i]
					owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
					catalogPath := shelf.EffectiveCatalogPath()

					data, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
					if err != nil {
						warn("Could not load catalog for shelf %q: %v", shelf.Name, err)
						continue
					}
					books, err := catalog.Parse(data)
					if err != nil {
						warn("Could not parse catalog for shelf %q: %v", shelf.Name, err)
						continue
					}

					for _, b := range books {
						cached := cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset)
						allItems = append(allItems, tui.BookItem{
							Book:      b,
							ShelfName: shelf.Name,
							Cached:    cached,
							Owner:     owner,
							Repo:      shelf.Repo,
						})
					}
				}

				if len(allItems) == 0 {
					return fmt.Errorf("no books found in library")
				}

				// Show multi-select book picker
				selected, err := tui.RunBookPickerMulti(allItems, "Select books to delete")
				if err != nil {
					return err
				}

				booksToDelete = selected
			} else {
				// CLI mode: single book ID provided
				bookID := args[0]

				// Find the book
				var foundShelf *config.ShelfConfig
				var foundBook *catalog.Book

				if shelfName != "" {
					foundShelf = cfg.ShelfByName(shelfName)
					if foundShelf == nil {
						return fmt.Errorf("shelf %q not found in config", shelfName)
					}
				}

				// Search for the book
				for i := range cfg.Shelves {
					s := &cfg.Shelves[i]
					if shelfName != "" && s.Name != shelfName {
						continue
					}

					owner := s.EffectiveOwner(cfg.GitHub.Owner)
					catalogPath := s.EffectiveCatalogPath()

					data, _, err := gh.GetFileContent(owner, s.Repo, catalogPath, "")
					if err != nil {
						continue
					}
					books, err := catalog.Parse(data)
					if err != nil {
						continue
					}

					for j := range books {
						if books[j].ID == bookID {
							foundBook = &books[j]
							foundShelf = s
							break
						}
					}
					if foundBook != nil {
						break
					}
				}

				if foundBook == nil {
					if shelfName != "" {
						return fmt.Errorf("book %q not found in shelf %q", bookID, shelfName)
					}
					return fmt.Errorf("book %q not found in any shelf", bookID)
				}

				// Add to delete list
				owner := foundShelf.EffectiveOwner(cfg.GitHub.Owner)
				cached := cacheMgr.Exists(owner, foundShelf.Repo, foundBook.ID, foundBook.Source.Asset)
				booksToDelete = []tui.BookItem{{
					Book:      *foundBook,
					ShelfName: foundShelf.Name,
					Cached:    cached,
					Owner:     owner,
					Repo:      foundShelf.Repo,
				}}
			}

			// Show warning and get confirmation for batch
			fmt.Println()
			if len(booksToDelete) == 1 {
				fmt.Println(color.YellowString("⚠ Warning: You are about to delete a book"))
			} else {
				fmt.Println(color.YellowString("⚠ Warning: You are about to delete %d books", len(booksToDelete)))
			}
			fmt.Println()

			for _, item := range booksToDelete {
				fmt.Printf("  • %s - %s [%s]\n",
					color.WhiteString(item.Book.ID),
					item.Book.Title,
					color.CyanString(item.ShelfName))
			}

			fmt.Println()
			fmt.Println(color.RedString("This will:"))
			fmt.Println("  • Remove books from catalog.yml")
			fmt.Println("  • " + color.RedString("DELETE") + " the files from GitHub Release assets")
			fmt.Println()
			fmt.Println(color.RedString("THIS CANNOT BE UNDONE"))
			fmt.Println()

			// Confirm
			if !skipConfirm {
				if len(booksToDelete) == 1 {
					fmt.Print("Type the book ID to confirm deletion: ")
					var confirmation string
					_, _ = fmt.Scanln(&confirmation)

					if confirmation != booksToDelete[0].Book.ID {
						return fmt.Errorf("confirmation did not match - aborted")
					}
				} else {
					fmt.Printf("Type 'DELETE %d BOOKS' to confirm: ", len(booksToDelete))
					reader := bufio.NewReader(os.Stdin)
					confirmation, _ := reader.ReadString('\n')
					confirmation = strings.TrimSpace(confirmation)

					expected := fmt.Sprintf("DELETE %d BOOKS", len(booksToDelete))
					if confirmation != expected {
						return fmt.Errorf("confirmation did not match - aborted")
					}
				}
			}

			// Process each book deletion
			for i, item := range booksToDelete {
				if len(booksToDelete) > 1 {
					fmt.Printf("\n[%d/%d] Deleting %s …\n", i+1, len(booksToDelete), item.Book.ID)
				}

				if err := deleteSingleBook(item); err != nil {
					warn("Failed to delete %s: %v", item.Book.ID, err)
					failCount++
					continue
				}

				successCount++
			}

			// Summary
			fmt.Println()
			if len(booksToDelete) == 1 {
				if successCount == 1 {
					ok("Book successfully deleted: %s", booksToDelete[0].Book.ID)
				}
			} else {
				if successCount > 0 {
					ok("Successfully deleted %d books", successCount)
				}
				if failCount > 0 {
					warn("%d books failed to delete", failCount)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Shelf containing the book (if ID is ambiguous)")
	cmd.Flags().BoolVar(&skipConfirm, "yes", false, "Skip confirmation prompt")

	return cmd
}

// deleteSingleBook deletes a single book: removes asset, updates catalog, clears cache
func deleteSingleBook(item tui.BookItem) error {
	shelf := cfg.ShelfByName(item.ShelfName)
	if shelf == nil {
		return fmt.Errorf("shelf %q not found", item.ShelfName)
	}

	catalogPath := shelf.EffectiveCatalogPath()
	releaseTag := shelf.EffectiveRelease(cfg.Defaults.Release)

	// Get the release
	rel, err := gh.GetReleaseByTag(item.Owner, item.Repo, releaseTag)
	if err != nil {
		return fmt.Errorf("could not get release: %w", err)
	}

	// Find the asset
	asset, err := gh.FindAsset(item.Owner, item.Repo, rel.ID, item.Book.Source.Asset)
	if err != nil {
		return fmt.Errorf("could not find asset: %w", err)
	}
	if asset == nil {
		return fmt.Errorf("asset %q not found in release", item.Book.Source.Asset)
	}

	// Delete the asset from GitHub
	if err := gh.DeleteAsset(item.Owner, item.Repo, asset.ID); err != nil {
		return fmt.Errorf("could not delete asset: %w", err)
	}

	// Load catalog
	data, _, err := gh.GetFileContent(item.Owner, item.Repo, catalogPath, "")
	if err != nil {
		return fmt.Errorf("could not load catalog: %w", err)
	}
	books, err := catalog.Parse(data)
	if err != nil {
		return fmt.Errorf("could not parse catalog: %w", err)
	}

	// Remove from catalog
	books, removed := catalog.Remove(books, item.Book.ID)
	if !removed {
		return fmt.Errorf("book %q not found in catalog", item.Book.ID)
	}

	// Marshal and commit updated catalog
	updatedData, err := catalog.Marshal(books)
	if err != nil {
		return fmt.Errorf("could not marshal catalog: %w", err)
	}
	commitMsg := fmt.Sprintf("delete: %s", item.Book.ID)
	if err := gh.CommitFile(item.Owner, item.Repo, catalogPath, updatedData, commitMsg); err != nil {
		return fmt.Errorf("could not commit catalog: %w", err)
	}

	// Clear from cache
	if cacheMgr.Exists(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset) {
		if err := cacheMgr.Remove(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset); err != nil {
			warn("Could not clear cache: %v", err)
		}
	}

	// Update README
	updateREADMEAfterRemove(item.Owner, item.Repo, books, item.Book.ID)

	return nil
}


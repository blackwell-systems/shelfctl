package app

import (
	"fmt"

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
		Use:   "delete-book <id>",
		Short: "Remove a book from your library",
		Long: `Remove a book from your library by deleting its metadata and file.

This command:
  • Removes the book entry from catalog.yml
  • Deletes the PDF/EPUB file from GitHub Release assets
  • Pushes the updated catalog

This action is DESTRUCTIVE and cannot be easily undone.

Examples:
  # Interactive book picker
  shelfctl delete-book

  # Delete specific book
  shelfctl delete-book sicp --shelf programming

  # Skip confirmation prompt
  shelfctl delete-book sicp --shelf programming --yes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var bookID string

			// If no book ID provided, show interactive picker
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

				// Show book picker
				selected, err := tui.RunBookPicker(allItems, "Select book to delete")
				if err != nil {
					return err
				}

				bookID = selected.Book.ID
				shelfName = selected.ShelfName
			} else {
				bookID = args[0]
			}

			// Find the book
			var shelf *config.ShelfConfig
			var book *catalog.Book

			if shelfName != "" {
				shelf = cfg.ShelfByName(shelfName)
				if shelf == nil {
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
						book = &books[j]
						shelf = s
						break
					}
				}
				if book != nil {
					break
				}
			}

			if book == nil {
				if shelfName != "" {
					return fmt.Errorf("book %q not found in shelf %q", bookID, shelfName)
				}
				return fmt.Errorf("book %q not found in any shelf", bookID)
			}

			owner := shelf.EffectiveOwner(cfg.GitHub.Owner)

			// Show warning and get confirmation
			fmt.Println()
			fmt.Println(color.YellowString("⚠ Warning: You are about to delete a book"))
			fmt.Println()
			fmt.Printf("Book ID:    %s\n", color.WhiteString(book.ID))
			fmt.Printf("Title:      %s\n", color.WhiteString(book.Title))
			if book.Author != "" {
				fmt.Printf("Author:     %s\n", book.Author)
			}
			fmt.Printf("Shelf:      %s\n", color.CyanString(shelf.Name))
			fmt.Printf("Format:     %s\n", book.Format)
			fmt.Println()
			fmt.Println(color.RedString("This will:"))
			fmt.Println("  • Remove book from catalog.yml")
			fmt.Println("  • " + color.RedString("DELETE") + " the file from GitHub Release assets")
			fmt.Println()
			fmt.Println(color.RedString("THIS CANNOT BE UNDONE"))
			fmt.Println()

			// Confirm
			if !skipConfirm {
				fmt.Print("Type the book ID to confirm deletion: ")
				var confirmation string
				_, _ = fmt.Scanln(&confirmation)

				if confirmation != bookID {
					return fmt.Errorf("confirmation did not match - aborted")
				}
			}

			// Get release info
			releaseTag := book.Source.Release
			if releaseTag == "" {
				releaseTag = shelf.EffectiveRelease(cfg.Defaults.Release)
			}

			rel, err := gh.GetReleaseByTag(owner, shelf.Repo, releaseTag)
			if err != nil {
				return fmt.Errorf("getting release %q: %w", releaseTag, err)
			}

			// Find and delete the asset
			asset, err := gh.FindAsset(owner, shelf.Repo, rel.ID, book.Source.Asset)
			if err != nil {
				return fmt.Errorf("finding asset: %w", err)
			}

			if asset != nil {
				header("Deleting file from GitHub Release …")
				if err := gh.DeleteAsset(owner, shelf.Repo, asset.ID); err != nil {
					return fmt.Errorf("deleting asset: %w", err)
				}
				ok("Deleted asset: %s", book.Source.Asset)
			} else {
				warn("Asset %q not found in release (may have been manually deleted)", book.Source.Asset)
			}

			// Remove from catalog
			header("Updating catalog …")
			catalogPath := shelf.EffectiveCatalogPath()
			data, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			books, err := catalog.Parse(data)
			if err != nil {
				return fmt.Errorf("parsing catalog: %w", err)
			}

			books, removed := catalog.Remove(books, bookID)
			if !removed {
				return fmt.Errorf("book not found in catalog")
			}

			updatedData, err := catalog.Marshal(books)
			if err != nil {
				return fmt.Errorf("marshaling catalog: %w", err)
			}

			commitMsg := fmt.Sprintf("delete: remove %s", bookID)
			if err := gh.CommitFile(owner, shelf.Repo, catalogPath, updatedData, commitMsg); err != nil {
				return fmt.Errorf("committing catalog: %w", err)
			}

			ok("Updated catalog")

			// Clear from cache if present
			if cacheMgr.Exists(owner, shelf.Repo, bookID, book.Source.Asset) {
				if err := cacheMgr.Remove(owner, shelf.Repo, bookID, book.Source.Asset); err != nil {
					warn("Could not remove from cache: %v", err)
				} else {
					ok("Removed from local cache")
				}
			}

			fmt.Println()
			fmt.Println(color.GreenString("✓ Book deleted: %s", bookID))

			return nil
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Shelf containing the book (if ID is ambiguous)")
	cmd.Flags().BoolVar(&skipConfirm, "yes", false, "Skip confirmation prompt")

	return cmd
}

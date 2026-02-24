package app

import (
	"fmt"
	"io"
	"os"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/spf13/cobra"
)

type moveParams struct {
	shelfName   string
	toRelease   string
	toShelfName string
	dryRun      bool
	keepOld     bool
}

type moveDestination struct {
	owner   string
	repo    string
	release string
	shelf   *config.ShelfConfig
}

func newMoveCmd() *cobra.Command {
	params := &moveParams{}

	cmd := &cobra.Command{
		Use:   "move [id]",
		Short: "Move book(s) to a different release or shelf",
		Long: `Move one or more books to a different release or shelf.

Within the same shelf (different release):
  shelfctl move book-id --to-release v2.0

To a different shelf:
  shelfctl move book-id --to-shelf other-shelf

Interactive mode (no arguments) with multi-select:
  shelfctl move
  • Spacebar to toggle selection
  • Enter to confirm
  • If no checkboxes selected, moves the current book`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var booksToMove []tui.BookItem

			// Interactive mode: pick book(s)
			if len(args) == 0 {
				if !tui.ShouldUseTUI(cmd) {
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
				selected, err := tui.RunBookPickerMulti(allItems, "Select books to move")
				if err != nil {
					return err
				}

				booksToMove = selected
			} else {
				// CLI mode: single book ID provided
				bookID := args[0]

				// Find the book
				var foundShelf *config.ShelfConfig
				var foundBook *catalog.Book

				if params.shelfName != "" {
					foundShelf = cfg.ShelfByName(params.shelfName)
					if foundShelf == nil {
						return fmt.Errorf("shelf %q not found in config", params.shelfName)
					}
				}

				// Search for the book
				for i := range cfg.Shelves {
					s := &cfg.Shelves[i]
					if params.shelfName != "" && s.Name != params.shelfName {
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
					if params.shelfName != "" {
						return fmt.Errorf("book %q not found in shelf %q", bookID, params.shelfName)
					}
					return fmt.Errorf("book %q not found in any shelf", bookID)
				}

				// Add to move list
				owner := foundShelf.EffectiveOwner(cfg.GitHub.Owner)
				cached := cacheMgr.Exists(owner, foundShelf.Repo, foundBook.ID, foundBook.Source.Asset)
				booksToMove = []tui.BookItem{{
					Book:      *foundBook,
					ShelfName: foundShelf.Name,
					Cached:    cached,
					Owner:     owner,
					Repo:      foundShelf.Repo,
				}}
			}

			// Interactive destination selection
			if params.toRelease == "" && params.toShelfName == "" && tui.ShouldUseTUI(cmd) {
				return runInteractiveBatchMove(booksToMove, params)
			}

			return runBatchMove(booksToMove, params)
		},
	}

	cmd.Flags().StringVar(&params.shelfName, "shelf", "", "Source shelf (if ID is ambiguous)")
	cmd.Flags().StringVar(&params.toRelease, "to-release", "", "Destination release tag (same repo)")
	cmd.Flags().StringVar(&params.toShelfName, "to-shelf", "", "Destination shelf name (different repo)")
	cmd.Flags().BoolVar(&params.dryRun, "dry-run", false, "Show what would happen without making changes")
	cmd.Flags().BoolVar(&params.keepOld, "keep-old", false, "Do not delete the old asset after copy")
	return cmd
}

func runInteractiveMove(bookID string, params *moveParams, _ *tui.BookItem) error {
	// Get source book details
	b, srcShelf, err := findBook(bookID, params.shelfName)
	if err != nil {
		return err
	}
	srcOwner := srcShelf.EffectiveOwner(cfg.GitHub.Owner)

	fmt.Println()
	header("Move Book: %s", b.Title)
	fmt.Printf("  ID: %s\n", b.ID)
	fmt.Printf("  Current location: %s/%s@%s\n", srcOwner, srcShelf.Repo, b.Source.Release)
	fmt.Println()

	// Ask move type
	fmt.Println("Move to:")
	fmt.Println("  1. Different shelf (different repository)")
	fmt.Println("  2. Different release (same shelf)")
	fmt.Println()
	fmt.Print("Your choice (1/2): ")
	var choice string
	_, _ = fmt.Scanln(&choice)
	fmt.Println()

	switch choice {
	case "1":
		// Moving to different shelf
		var shelfOptions []tui.ShelfOption
		for _, shelf := range cfg.Shelves {
			// Exclude current shelf
			if shelf.Name != srcShelf.Name {
				shelfOptions = append(shelfOptions, tui.ShelfOption{
					Name: shelf.Name,
					Repo: shelf.Repo,
				})
			}
		}

		if len(shelfOptions) == 0 {
			return fmt.Errorf("no other shelves available - create another shelf first")
		}

		// Pick destination shelf
		dstShelfName, err := tui.RunShelfPicker(shelfOptions)
		if err != nil {
			return err
		}

		params.toShelfName = dstShelfName

		// Optionally specify release
		fmt.Println()
		fmt.Println("Destination release tag (press Enter for default):")
		fmt.Print("Release tag: ")
		var releaseTag string
		_, _ = fmt.Scanln(&releaseTag)
		if releaseTag != "" {
			params.toRelease = releaseTag
		}

	case "2":
		// Moving to different release in same shelf
		fmt.Println("Enter destination release tag:")
		fmt.Printf("Release tag (current: %s): ", b.Source.Release)
		var releaseTag string
		_, _ = fmt.Scanln(&releaseTag)
		if releaseTag == "" {
			return fmt.Errorf("release tag required")
		}
		params.toRelease = releaseTag

	default:
		return fmt.Errorf("invalid choice")
	}

	// Determine destination for confirmation
	dst, err := determineDestination(srcShelf, srcOwner, params)
	if err != nil {
		return err
	}

	// Show confirmation
	fmt.Println()
	fmt.Println("Move summary:")
	fmt.Printf("  Book:  %s (%s)\n", b.Title, b.ID)
	fmt.Printf("  From:  %s/%s@%s\n", srcOwner, srcShelf.Repo, b.Source.Release)
	fmt.Printf("  To:    %s/%s@%s\n", dst.owner, dst.repo, dst.release)
	fmt.Println()
	fmt.Print("Proceed with move? (y/n): ")
	var confirm string
	_, _ = fmt.Scanln(&confirm)

	if confirm != "y" && confirm != "Y" && confirm != "yes" {
		fmt.Println("Move canceled")
		return nil
	}

	fmt.Println()
	return runMove(bookID, params)
}

func runInteractiveBatchMove(booksToMove []tui.BookItem, params *moveParams) error {
	if len(booksToMove) == 0 {
		return fmt.Errorf("no books selected")
	}

	// Show selected books
	fmt.Println()
	if len(booksToMove) == 1 {
		fmt.Println("Moving 1 book:")
	} else {
		fmt.Printf("Moving %d books:\n", len(booksToMove))
	}
	for _, item := range booksToMove {
		fmt.Printf("  • %s - %s [%s]\n", item.Book.ID, item.Book.Title, item.ShelfName)
	}
	fmt.Println()

	// Ask move type
	fmt.Println("Move to:")
	fmt.Println("  1. Different shelf (different repository)")
	fmt.Println("  2. Different release (same shelf)")
	fmt.Println()
	fmt.Print("Your choice (1/2): ")
	var choice string
	_, _ = fmt.Scanln(&choice)
	fmt.Println()

	switch choice {
	case "1":
		// Moving to different shelf
		var shelfOptions []tui.ShelfOption
		// Get unique shelves from selected books to exclude
		excludedShelves := make(map[string]bool)
		for _, item := range booksToMove {
			excludedShelves[item.ShelfName] = true
		}

		for _, shelf := range cfg.Shelves {
			// Only show shelves not in the excluded set
			if !excludedShelves[shelf.Name] {
				shelfOptions = append(shelfOptions, tui.ShelfOption{
					Name: shelf.Name,
					Repo: shelf.Repo,
				})
			}
		}

		if len(shelfOptions) == 0 {
			return fmt.Errorf("no other shelves available - create another shelf first")
		}

		// Pick destination shelf
		dstShelfName, err := tui.RunShelfPicker(shelfOptions)
		if err != nil {
			return err
		}

		params.toShelfName = dstShelfName

		// Optionally specify release
		fmt.Println()
		fmt.Println("Destination release tag (press Enter for default):")
		fmt.Print("Release tag: ")
		var releaseTag string
		_, _ = fmt.Scanln(&releaseTag)
		if releaseTag != "" {
			params.toRelease = releaseTag
		}

	case "2":
		// Moving to different release in same shelf
		// Check if all books are from the same shelf
		firstShelfName := booksToMove[0].ShelfName
		allSameShelf := true
		for _, item := range booksToMove {
			if item.ShelfName != firstShelfName {
				allSameShelf = false
				break
			}
		}

		if !allSameShelf {
			return fmt.Errorf("cannot move books from different shelves to a different release - use different shelf option instead")
		}

		fmt.Println("Enter destination release tag:")
		fmt.Printf("Release tag (current: %s): ", booksToMove[0].Book.Source.Release)
		var releaseTag string
		_, _ = fmt.Scanln(&releaseTag)
		if releaseTag == "" {
			return fmt.Errorf("release tag required")
		}
		params.toRelease = releaseTag

	default:
		return fmt.Errorf("invalid choice")
	}

	// Show destination summary and confirm
	fmt.Println()
	fmt.Println("Move summary:")
	if len(booksToMove) == 1 {
		fmt.Printf("  Book:  %s (%s)\n", booksToMove[0].Book.Title, booksToMove[0].Book.ID)
		fmt.Printf("  From:  %s/%s@%s\n", booksToMove[0].Owner, booksToMove[0].Repo, booksToMove[0].Book.Source.Release)
	} else {
		fmt.Printf("  Books: %d selected\n", len(booksToMove))
	}

	// Show destination info
	if params.toShelfName != "" {
		dstShelf := cfg.ShelfByName(params.toShelfName)
		if dstShelf != nil {
			owner := dstShelf.EffectiveOwner(cfg.GitHub.Owner)
			release := params.toRelease
			if release == "" {
				release = dstShelf.EffectiveRelease(cfg.Defaults.Release)
			}
			fmt.Printf("  To:    %s/%s@%s\n", owner, dstShelf.Repo, release)
		}
	} else {
		fmt.Printf("  To:    %s/%s@%s\n", booksToMove[0].Owner, booksToMove[0].Repo, params.toRelease)
	}

	fmt.Println()
	fmt.Print("Proceed with move? (y/n): ")
	var confirm string
	_, _ = fmt.Scanln(&confirm)

	if confirm != "y" && confirm != "Y" && confirm != "yes" {
		fmt.Println("Move canceled")
		return nil
	}

	fmt.Println()
	return runBatchMove(booksToMove, params)
}

func runBatchMove(booksToMove []tui.BookItem, params *moveParams) error {
	successCount := 0
	failCount := 0

	for i, item := range booksToMove {
		if len(booksToMove) > 1 {
			fmt.Printf("\n[%d/%d] Moving %s …\n", i+1, len(booksToMove), item.Book.ID)
		}

		// Create a copy of params for this book to avoid mutation issues
		bookParams := *params
		bookParams.shelfName = item.ShelfName

		if err := runMove(item.Book.ID, &bookParams); err != nil {
			warn("Failed to move %s: %v", item.Book.ID, err)
			failCount++
			continue
		}

		successCount++
	}

	// Summary
	fmt.Println()
	if len(booksToMove) == 1 {
		if successCount == 1 {
			ok("Book successfully moved: %s", booksToMove[0].Book.ID)
		}
	} else {
		if successCount > 0 {
			ok("Successfully moved %d books", successCount)
		}
		if failCount > 0 {
			warn("%d books failed to move", failCount)
		}
	}

	return nil
}

func runMove(id string, params *moveParams) error {
	if params.toRelease == "" && params.toShelfName == "" {
		return fmt.Errorf("one of --to-release or --to-shelf is required")
	}

	// Find source book
	b, srcShelf, err := findBook(id, params.shelfName)
	if err != nil {
		return err
	}
	srcOwner := srcShelf.EffectiveOwner(cfg.GitHub.Owner)

	// Determine destination
	dst, err := determineDestination(srcShelf, srcOwner, params)
	if err != nil {
		return err
	}

	// Check if destination is the same as source
	if srcOwner == dst.owner && srcShelf.Repo == dst.repo && b.Source.Release == dst.release {
		return fmt.Errorf("book is already at %s/%s@%s - nothing to move", dst.owner, dst.repo, dst.release)
	}

	fmt.Printf("Moving %s: %s/%s@%s → %s/%s@%s\n",
		id, srcOwner, srcShelf.Repo, b.Source.Release,
		dst.owner, dst.repo, dst.release)

	if params.dryRun {
		fmt.Println("(dry run — no changes made)")
		return nil
	}

	// Transfer asset
	if err := transferAsset(b, srcShelf, srcOwner, dst); err != nil {
		return err
	}

	// Delete old asset unless keeping
	if !params.keepOld {
		deleteOldAsset(srcOwner, srcShelf.Repo, b, srcShelf)
	}

	// Update catalogs
	if params.toShelfName != "" {
		return updateCatalogsForCrossShelfMove(id, b, srcShelf, srcOwner, dst)
	}
	return updateCatalogForSameShelfMove(id, b, srcShelf, srcOwner, dst)
}

func determineDestination(srcShelf *config.ShelfConfig, srcOwner string, params *moveParams) (*moveDestination, error) {
	dst := &moveDestination{
		owner:   srcOwner,
		repo:    srcShelf.Repo,
		release: params.toRelease,
	}

	if params.toShelfName != "" {
		dstShelf := cfg.ShelfByName(params.toShelfName)
		if dstShelf == nil {
			return nil, fmt.Errorf("destination shelf %q not found in config", params.toShelfName)
		}
		dst.owner = dstShelf.EffectiveOwner(cfg.GitHub.Owner)
		dst.repo = dstShelf.Repo
		dst.shelf = dstShelf
		if params.toRelease == "" {
			dst.release = dstShelf.EffectiveRelease(cfg.Defaults.Release)
		}
	} else if dst.release == "" {
		return nil, fmt.Errorf("--to-release is required when not using --to-shelf")
	}

	return dst, nil
}

func transferAsset(b *catalog.Book, srcShelf *config.ShelfConfig, srcOwner string, dst *moveDestination) error {
	// Ensure destination release exists
	dstRel, err := gh.EnsureRelease(dst.owner, dst.repo, dst.release)
	if err != nil {
		return fmt.Errorf("ensuring destination release: %w", err)
	}

	// Get source asset
	srcRel, err := gh.GetReleaseByTag(srcOwner, srcShelf.Repo, b.Source.Release)
	if err != nil {
		return err
	}
	srcAsset, err := gh.FindAsset(srcOwner, srcShelf.Repo, srcRel.ID, b.Source.Asset)
	if err != nil {
		return err
	}
	if srcAsset == nil {
		return fmt.Errorf("source asset %q not found", b.Source.Asset)
	}

	// Download and buffer
	tmpPath, size, err := downloadAndBuffer(srcOwner, srcShelf.Repo, srcAsset.ID)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmpPath) }()

	// Upload to destination
	uploadFile, err := os.Open(tmpPath)
	if err != nil {
		return err
	}
	defer func() { _ = uploadFile.Close() }()

	_, err = gh.UploadAsset(dst.owner, dst.repo, dstRel.ID, b.Source.Asset,
		uploadFile, size, "application/octet-stream")
	if err != nil {
		return fmt.Errorf("uploading to destination: %w", err)
	}

	ok("Uploaded to %s/%s@%s", dst.owner, dst.repo, dst.release)
	return nil
}

func downloadAndBuffer(owner, repo string, assetID int64) (string, int64, error) {
	rc, err := gh.DownloadAsset(owner, repo, assetID)
	if err != nil {
		return "", 0, fmt.Errorf("downloading source: %w", err)
	}

	tmp, err := os.CreateTemp("", "shelfctl-move-*")
	if err != nil {
		_ = rc.Close()
		return "", 0, err
	}
	tmpPath := tmp.Name()

	if _, err := io.Copy(tmp, rc); err != nil {
		_ = tmp.Close()
		_ = rc.Close()
		_ = os.Remove(tmpPath)
		return "", 0, err
	}
	_ = tmp.Close()
	_ = rc.Close()

	fi, err := os.Stat(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, err
	}

	return tmpPath, fi.Size(), nil
}

func deleteOldAsset(srcOwner, srcRepo string, b *catalog.Book, _ *config.ShelfConfig) {
	srcRel, err := gh.GetReleaseByTag(srcOwner, srcRepo, b.Source.Release)
	if err != nil {
		warn("Could not get source release: %v", err)
		return
	}

	srcAsset, err := gh.FindAsset(srcOwner, srcRepo, srcRel.ID, b.Source.Asset)
	if err != nil {
		warn("Could not find source asset: %v", err)
		return
	}

	if srcAsset == nil {
		warn("Source asset not found")
		return
	}

	if err := gh.DeleteAsset(srcOwner, srcRepo, srcAsset.ID); err != nil {
		warn("Could not delete old asset: %v", err)
	} else {
		ok("Deleted old asset from %s@%s", srcRepo, b.Source.Release)
	}
}

func updateCatalogsForCrossShelfMove(id string, b *catalog.Book, srcShelf *config.ShelfConfig, srcOwner string, dst *moveDestination) error {
	// Load and update source catalog
	srcCatalogPath := srcShelf.EffectiveCatalogPath()
	data, _, err := gh.GetFileContent(srcOwner, srcShelf.Repo, srcCatalogPath, "")
	if err != nil {
		return err
	}
	books, err := catalog.Parse(data)
	if err != nil {
		return err
	}

	// Load destination catalog to check for conflicts
	dstCatalogPath := dst.shelf.EffectiveCatalogPath()
	dstData, _, _ := gh.GetFileContent(dst.owner, dst.repo, dstCatalogPath, "")
	dstBooks, _ := catalog.Parse(dstData)

	// Check for ID conflict in destination
	for _, existing := range dstBooks {
		if existing.ID == id {
			warn("Book with ID %q already exists in destination shelf %q", id, dst.shelf.Name)
			warn("Existing book will be replaced")
			break
		}
	}

	// Clear local cache for this book (path will be invalid after move)
	if cacheMgr.Exists(srcOwner, srcShelf.Repo, b.ID, b.Source.Asset) {
		if err := cacheMgr.Remove(srcOwner, srcShelf.Repo, b.ID, b.Source.Asset); err != nil {
			warn("Could not clear cache: %v", err)
		} else {
			ok("Cleared from local cache (path will change after move)")
		}
	}

	// Remove from source
	books, _ = catalog.Remove(books, id)
	srcData, err := catalog.Marshal(books)
	if err != nil {
		return err
	}
	if err := gh.CommitFile(srcOwner, srcShelf.Repo, srcCatalogPath, srcData,
		fmt.Sprintf("move: remove %s (moved to %s)", id, dst.shelf.Name)); err != nil {
		return err
	}

	// Update book metadata for destination
	b.Source.Release = dst.release
	b.Source.Owner = dst.owner
	b.Source.Repo = dst.repo

	// Add to destination catalog (Append replaces if ID exists)
	dstBooks = catalog.Append(dstBooks, *b)
	dstCatalogData, err := catalog.Marshal(dstBooks)
	if err != nil {
		return err
	}
	if err := gh.CommitFile(dst.owner, dst.repo, dstCatalogPath, dstCatalogData,
		fmt.Sprintf("move: add %s (from %s)", id, srcShelf.Name)); err != nil {
		return err
	}

	ok("Catalog updated")

	// Update README files for both shelves
	updateREADMEAfterRemove(srcOwner, srcShelf.Repo, books, b.ID)
	updateREADMEAfterAdd(dst.owner, dst.repo, dstBooks, *b)

	// Handle catalog cover if it exists
	if b.Cover != "" {
		handleCatalogCoverMove(b, srcOwner, srcShelf.Repo, dst.owner, dst.repo)
	}

	return nil
}

func updateCatalogForSameShelfMove(id string, _ *catalog.Book, srcShelf *config.ShelfConfig, srcOwner string, dst *moveDestination) error {
	srcCatalogPath := srcShelf.EffectiveCatalogPath()
	data, _, err := gh.GetFileContent(srcOwner, srcShelf.Repo, srcCatalogPath, "")
	if err != nil {
		return err
	}
	books, err := catalog.Parse(data)
	if err != nil {
		return err
	}

	// Update release field for the book
	for i := range books {
		if books[i].ID == id {
			books[i].Source.Release = dst.release
			break
		}
	}

	newData, err := catalog.Marshal(books)
	if err != nil {
		return err
	}
	if err := gh.CommitFile(srcOwner, srcShelf.Repo, srcCatalogPath, newData,
		fmt.Sprintf("move: %s → release/%s", id, dst.release)); err != nil {
		return err
	}

	ok("Catalog updated")
	return nil
}

// handleCatalogCoverMove migrates catalog cover when moving between repos
func handleCatalogCoverMove(b *catalog.Book, srcOwner, srcRepo, dstOwner, dstRepo string) {
	// Download cover from source repo
	coverData, _, err := gh.GetFileContent(srcOwner, srcRepo, b.Cover, "")
	if err != nil {
		warn("Could not download catalog cover from source: %v", err)
		return
	}

	// Upload to destination repo (same relative path)
	commitMsg := fmt.Sprintf("move: copy cover for %s", b.ID)
	if err := gh.CommitFile(dstOwner, dstRepo, b.Cover, coverData, commitMsg); err != nil {
		warn("Could not upload catalog cover to destination: %v", err)
		return
	}

	ok("Catalog cover migrated")
}

// updateREADMEAfterRemove updates a shelf README after removing a book
func updateREADMEAfterRemove(owner, repo string, remainingBooks []catalog.Book, removedBookID string) {
	readmeData, _, err := gh.GetFileContent(owner, repo, "README.md", "")
	if err != nil {
		return // README doesn't exist or can't be read
	}

	originalContent := string(readmeData)
	readmeContent := updateShelfREADMEStats(originalContent, len(remainingBooks))
	readmeContent = removeFromShelfREADME(readmeContent, removedBookID)

	// Only commit if content actually changed
	if readmeContent == originalContent {
		return // No changes to commit
	}

	readmeMsg := fmt.Sprintf("Update README: remove %s", removedBookID)
	if err := gh.CommitFile(owner, repo, "README.md", []byte(readmeContent), readmeMsg); err != nil {
		warn("Could not update source README.md: %v", err)
	} else {
		ok("Source README.md updated")
	}
}

// updateREADMEAfterAdd updates a shelf README after adding a book
func updateREADMEAfterAdd(owner, repo string, books []catalog.Book, book catalog.Book) {
	readmeData, _, err := gh.GetFileContent(owner, repo, "README.md", "")
	if err != nil {
		return // README doesn't exist or can't be read
	}

	originalContent := string(readmeData)
	readmeContent := updateShelfREADMEStats(originalContent, len(books))
	readmeContent = appendToShelfREADME(readmeContent, book)

	// Only commit if content actually changed
	if readmeContent == originalContent {
		return // No changes to commit
	}

	readmeMsg := fmt.Sprintf("Update README: add %s", book.ID)
	if err := gh.CommitFile(owner, repo, "README.md", []byte(readmeContent), readmeMsg); err != nil {
		warn("Could not update destination README.md: %v", err)
	} else {
		ok("Destination README.md updated")
	}
}


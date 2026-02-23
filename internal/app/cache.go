package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage local book cache",
		Long:  "Manage the local cache of downloaded books without affecting shelf metadata or release assets.",
	}

	cmd.AddCommand(
		newCacheClearCmd(),
		newCacheInfoCmd(),
	)

	return cmd
}

func newCacheClearCmd() *cobra.Command {
	var (
		shelfName string
		all       bool
		force     bool
	)

	cmd := &cobra.Command{
		Use:   "clear [book-id...]",
		Short: "Remove books from local cache",
		Long: `Remove books from local cache without deleting them from shelves.

Modified files (with annotations/highlights) are protected by default.
Use --force to delete modified files.

Books will be re-downloaded when opened or browsed.

Examples:
  shelfctl cache clear book-id-1 book-id-2    Remove specific books
  shelfctl cache clear --shelf books          Remove all books from a shelf
  shelfctl cache clear --all                  Remove entire cache
  shelfctl cache clear --force                Remove including modified files
  shelfctl cache clear                        Interactive picker`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				return clearAllCache(force)
			}

			if shelfName != "" {
				return clearShelfCache(shelfName, force)
			}

			if len(args) > 0 {
				return clearSpecificBooks(args, force)
			}

			// Interactive mode
			return clearBooksInteractive(cmd, force)
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Clear cache for all books on this shelf")
	cmd.Flags().BoolVar(&all, "all", false, "Clear entire cache")
	cmd.Flags().BoolVar(&force, "force", false, "Delete modified files (annotations/highlights)")

	return cmd
}

func newCacheInfoCmd() *cobra.Command {
	var shelfName string

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show cache statistics",
		Long:  "Display information about cached books and disk usage.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if shelfName != "" {
				return showShelfCacheInfo(shelfName)
			}
			return showAllCacheInfo()
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Show stats for specific shelf only")

	return cmd
}

// clearAllCache removes the entire cache directory
func clearAllCache(force bool) error {
	cacheDir := cacheMgr.Path("", "", "", "")
	cacheDir = filepath.Dir(cacheDir) // Go up one level to get base dir

	// Check if cache exists
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		ok("Cache is already empty")
		return nil
	}

	// Check for modified files if not forcing
	if !force {
		allBooks := loadAllBooksAcrossShelves()
		modifiedCount := 0
		for _, item := range allBooks {
			if item.Cached && cacheMgr.HasBeenModified(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset, item.Book.Checksum.SHA256) {
				modifiedCount++
			}
		}

		if modifiedCount > 0 {
			warn("%d books have local changes (annotations/highlights)", modifiedCount)
			fmt.Printf("Run 'shelfctl sync --all' first, or use --force to delete anyway\n")
			return fmt.Errorf("modified files protected")
		}
	}

	// Get size before clearing
	size, count := calculateDirSize(cacheDir)

	// Confirm
	fmt.Printf("This will remove %d cached files (%s)\n", count, humanBytes(size))
	if force {
		fmt.Printf("%s This includes modified files with annotations\n", color.YellowString("⚠"))
	}
	fmt.Printf("Type 'CLEAR ALL CACHE' to confirm: ")
	reader := bufio.NewReader(os.Stdin)
	confirmation, _ := reader.ReadString('\n')
	confirmation = strings.TrimSpace(confirmation)

	if confirmation != "CLEAR ALL CACHE" {
		return fmt.Errorf("cancelled")
	}

	// Remove cache directory
	if err := os.RemoveAll(cacheDir); err != nil {
		return fmt.Errorf("removing cache: %w", err)
	}

	ok("Cleared %d files (%s)", count, humanBytes(size))
	return nil
}

// clearShelfCache removes all cached books from a specific shelf
func clearShelfCache(shelfName string, force bool) error {
	shelf := cfg.ShelfByName(shelfName)
	if shelf == nil {
		return fmt.Errorf("shelf %q not found", shelfName)
	}

	owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
	catalogPath := shelf.EffectiveCatalogPath()

	// Load catalog
	mgr := catalog.NewManager(gh, owner, shelf.Repo, catalogPath)
	books, err := mgr.Load()
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	// Count cached books and check for modifications
	cachedCount := 0
	modifiedCount := 0
	var cachedSize int64

	for i := range books {
		b := &books[i]
		if cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
			cachedCount++
			path := cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)
			if info, err := os.Stat(path); err == nil {
				cachedSize += info.Size()
			}

			// Check if modified
			if !force && cacheMgr.HasBeenModified(owner, shelf.Repo, b.ID, b.Source.Asset, b.Checksum.SHA256) {
				modifiedCount++
			}
		}
	}

	if cachedCount == 0 {
		ok("No cached books on shelf %s", shelfName)
		return nil
	}

	// Warn about modifications
	if modifiedCount > 0 && !force {
		warn("%d books have local changes (annotations/highlights)", modifiedCount)
		fmt.Printf("Run 'shelfctl sync --shelf %s' first, or use --force to delete anyway\n", shelfName)
		return fmt.Errorf("modified files protected")
	}

	// Confirm
	fmt.Printf("This will remove %d cached books from shelf %s (%s)\n", cachedCount, shelfName, humanBytes(cachedSize))
	if force && modifiedCount > 0 {
		fmt.Printf("%s This includes %d modified files with annotations\n", color.YellowString("⚠"), modifiedCount)
	}
	fmt.Printf("Type 'CLEAR CACHE' to confirm: ")
	reader := bufio.NewReader(os.Stdin)
	confirmation, _ := reader.ReadString('\n')
	confirmation = strings.TrimSpace(confirmation)

	if confirmation != "CLEAR CACHE" {
		return fmt.Errorf("cancelled")
	}

	// Remove each cached book
	removed := 0
	skipped := 0
	for i := range books {
		b := &books[i]
		if cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
			// Skip modified files unless forced
			if !force && cacheMgr.HasBeenModified(owner, shelf.Repo, b.ID, b.Source.Asset, b.Checksum.SHA256) {
				fmt.Printf("⚠ Skipped %s (modified)\n", b.ID)
				skipped++
				continue
			}

			if err := cacheMgr.Remove(owner, shelf.Repo, b.ID, b.Source.Asset); err != nil {
				warn("Failed to remove %s: %v", b.ID, err)
				continue
			}
			removed++
		}
	}

	ok("Removed %d books from cache (%s)", removed, humanBytes(cachedSize))
	if skipped > 0 {
		fmt.Printf("%s Skipped %d modified books (use --force to delete)\n", color.CyanString("ℹ"), skipped)
	}
	return nil
}

// clearSpecificBooks removes specific books from cache by ID
func clearSpecificBooks(bookIDs []string, force bool) error {
	// Load all shelves to find the books
	allBooks := loadAllBooksAcrossShelves()

	removedCount := 0
	skippedCount := 0
	var totalSize int64

	for _, bookID := range bookIDs {
		found := false
		for _, item := range allBooks {
			if item.Book.ID == bookID {
				found = true
				if !item.Cached {
					fmt.Printf("%s: not cached\n", bookID)
					continue
				}

				// Check if modified
				if !force && cacheMgr.HasBeenModified(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset, item.Book.Checksum.SHA256) {
					fmt.Printf("%s %s: modified (use --force to delete)\n", color.YellowString("⚠"), bookID)
					fmt.Printf("  Tip: Run 'shelfctl sync %s' to upload changes first\n", bookID)
					skippedCount++
					continue
				}

				path := cacheMgr.Path(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset)
				if info, err := os.Stat(path); err == nil {
					totalSize += info.Size()
				}

				if err := cacheMgr.Remove(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset); err != nil {
					warn("Failed to remove %s: %v", bookID, err)
					continue
				}

				fmt.Printf("%s: removed from cache\n", bookID)
				removedCount++
				break
			}
		}

		if !found {
			warn("Book %q not found", bookID)
		}
	}

	if removedCount > 0 {
		ok("Removed %d books from cache (%s)", removedCount, humanBytes(totalSize))
	}
	if skippedCount > 0 {
		fmt.Printf("\n%s Skipped %d modified books\n", color.CyanString("ℹ"), skippedCount)
	}

	return nil
}

// clearBooksInteractive launches interactive picker to select books to clear from cache
func clearBooksInteractive(cmd *cobra.Command, force bool) error {
	// Load all cached books
	allBooks := loadAllBooksAcrossShelves()

	// Filter to only cached books
	var cachedBooks []tui.BookItem
	modifiedCount := 0
	for _, item := range allBooks {
		if item.Cached {
			cachedBooks = append(cachedBooks, item)
			// Check if modified
			if cacheMgr.HasBeenModified(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset, item.Book.Checksum.SHA256) {
				modifiedCount++
			}
		}
	}

	if len(cachedBooks) == 0 {
		ok("No books in cache")
		return nil
	}

	// Warn about modified files
	if modifiedCount > 0 && !force {
		fmt.Printf("\n%s %d books have local changes (annotations/highlights)\n", color.YellowString("⚠"), modifiedCount)
		fmt.Printf("Modified books will be skipped unless you use --force\n\n")
	}

	// Launch multi-select picker
	selected, err := tui.RunBookPickerMulti(cachedBooks, "Select books to remove from cache")
	if err != nil {
		return err
	}

	if len(selected) == 0 {
		return fmt.Errorf("no books selected")
	}

	// Check for modified files in selection
	var toRemove []tui.BookItem
	var skipped []tui.BookItem
	var totalSize int64

	for _, item := range selected {
		path := cacheMgr.Path(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset)
		if info, err := os.Stat(path); err == nil {
			totalSize += info.Size()
		}

		// Check if modified
		if !force && cacheMgr.HasBeenModified(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset, item.Book.Checksum.SHA256) {
			skipped = append(skipped, item)
		} else {
			toRemove = append(toRemove, item)
		}
	}

	// Warn about skipped files
	if len(skipped) > 0 {
		fmt.Printf("\n%s The following books have local changes and will be skipped:\n", color.YellowString("⚠"))
		for _, item := range skipped {
			fmt.Printf("  - %s (%s)\n", item.Book.ID, item.Book.Title)
		}
		fmt.Printf("\nRun 'shelfctl sync --all' to upload changes, or use --force to delete\n")
	}

	if len(toRemove) == 0 {
		fmt.Printf("\nNo unmodified books selected. Nothing to remove.\n")
		return nil
	}

	// Confirm
	if len(toRemove) > 1 {
		fmt.Printf("\nYou are about to remove %d books from cache:\n", len(toRemove))
		for _, item := range toRemove {
			fmt.Printf("  - %s (%s)\n", item.Book.ID, item.Book.Title)
		}
		fmt.Printf("\nType 'CLEAR %d BOOKS' to confirm: ", len(toRemove))
		reader := bufio.NewReader(os.Stdin)
		confirmation, _ := reader.ReadString('\n')
		confirmation = strings.TrimSpace(confirmation)
		expected := fmt.Sprintf("CLEAR %d BOOKS", len(toRemove))
		if confirmation != expected {
			return fmt.Errorf("cancelled")
		}
	}

	// Remove from cache
	removedCount := 0
	for _, item := range toRemove {
		if err := cacheMgr.Remove(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset); err != nil {
			warn("Failed to remove %s: %v", item.Book.ID, err)
			continue
		}
		removedCount++
	}

	ok("Removed %d books from cache (%s)", removedCount, humanBytes(totalSize))
	if len(skipped) > 0 {
		fmt.Printf("%s Skipped %d modified books\n", color.CyanString("ℹ"), len(skipped))
	}
	return nil
}

// showAllCacheInfo displays cache statistics for all shelves
func showAllCacheInfo() error {
	allBooks := loadAllBooksAcrossShelves()

	totalBooks := len(allBooks)
	cachedCount := 0
	modifiedCount := 0
	var totalSize int64
	var modifiedBooks []tui.BookItem
	var uncachedBooks []tui.BookItem

	for _, item := range allBooks {
		if item.Cached {
			cachedCount++
			path := cacheMgr.Path(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset)
			if info, err := os.Stat(path); err == nil {
				totalSize += info.Size()
			}

			// Check if modified
			if cacheMgr.HasBeenModified(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset, item.Book.Checksum.SHA256) {
				modifiedCount++
				modifiedBooks = append(modifiedBooks, item)
			}
		} else {
			uncachedBooks = append(uncachedBooks, item)
		}
	}

	header("Cache Statistics")
	printField("total_books", fmt.Sprintf("%d", totalBooks))
	printField("cached_books", fmt.Sprintf("%d", cachedCount))
	if modifiedCount > 0 {
		printField("modified", fmt.Sprintf("%d (annotations/highlights)", modifiedCount))
	}
	printField("cache_size", humanBytes(totalSize))
	printField("cache_dir", filepath.Dir(cacheMgr.Path("", "", "", "")))

	uncachedCount := totalBooks - cachedCount
	if uncachedCount > 0 {
		fmt.Println()
		fmt.Printf("%s %d books not cached:\n", color.YellowString("⚠"), uncachedCount)
		for _, item := range uncachedBooks {
			fmt.Printf("  - %s (%s)\n", item.Book.ID, item.Book.Title)
		}
	}

	if modifiedCount > 0 {
		fmt.Println()
		fmt.Printf("%s %d books have local changes:\n", color.CyanString("ℹ"), modifiedCount)
		for _, item := range modifiedBooks {
			fmt.Printf("  - %s (%s)\n", item.Book.ID, item.Book.Title)
		}
		fmt.Printf("\n  Run 'shelfctl sync --all' to upload changes to GitHub\n")
	}

	return nil
}

// showShelfCacheInfo displays cache statistics for a specific shelf
func showShelfCacheInfo(shelfName string) error {
	shelf := cfg.ShelfByName(shelfName)
	if shelf == nil {
		return fmt.Errorf("shelf %q not found", shelfName)
	}

	owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
	catalogPath := shelf.EffectiveCatalogPath()

	// Load catalog
	mgr := catalog.NewManager(gh, owner, shelf.Repo, catalogPath)
	books, err := mgr.Load()
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	totalBooks := len(books)
	cachedCount := 0
	modifiedCount := 0
	var totalSize int64
	var modifiedBooks []*catalog.Book
	var uncachedBooks []*catalog.Book

	for i := range books {
		b := &books[i]
		if cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
			cachedCount++
			path := cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)
			if info, err := os.Stat(path); err == nil {
				totalSize += info.Size()
			}

			// Check if modified
			if cacheMgr.HasBeenModified(owner, shelf.Repo, b.ID, b.Source.Asset, b.Checksum.SHA256) {
				modifiedCount++
				modifiedBooks = append(modifiedBooks, b)
			}
		} else {
			uncachedBooks = append(uncachedBooks, b)
		}
	}

	header("Cache Statistics: %s", shelfName)
	printField("total_books", fmt.Sprintf("%d", totalBooks))
	printField("cached_books", fmt.Sprintf("%d", cachedCount))
	if modifiedCount > 0 {
		printField("modified", fmt.Sprintf("%d (annotations/highlights)", modifiedCount))
	}
	printField("cache_size", humanBytes(totalSize))
	printField("repository", fmt.Sprintf("%s/%s", owner, shelf.Repo))

	uncachedCount := totalBooks - cachedCount
	if uncachedCount > 0 {
		fmt.Println()
		fmt.Printf("%s %d books not cached:\n", color.YellowString("⚠"), uncachedCount)
		for _, b := range uncachedBooks {
			fmt.Printf("  - %s (%s)\n", b.ID, b.Title)
		}
	}

	if modifiedCount > 0 {
		fmt.Println()
		fmt.Printf("%s %d books have local changes:\n", color.CyanString("ℹ"), modifiedCount)
		for _, b := range modifiedBooks {
			fmt.Printf("  - %s (%s)\n", b.ID, b.Title)
		}
		fmt.Printf("\n  Run 'shelfctl sync --shelf %s' to upload changes\n", shelfName)
	}

	return nil
}

// calculateDirSize recursively calculates total size and file count
func calculateDirSize(path string) (int64, int) {
	var size int64
	count := 0

	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
			count++
		}
		return nil
	})

	return size, count
}

// loadAllBooksAcrossShelves loads all books from all configured shelves
func loadAllBooksAcrossShelves() []tui.BookItem {
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

			// Download catalog cover if specified and not already cached
			if b.Cover != "" && !cacheMgr.HasCatalogCover(shelf.Repo, b.ID) {
				if coverData, _, err := gh.GetFileContent(owner, shelf.Repo, b.Cover, ""); err == nil {
					_ = cacheMgr.StoreCatalogCover(shelf.Repo, b.ID, strings.NewReader(string(coverData)))
				}
			}

			// Get best available cover (catalog > extracted > none)
			coverPath := cacheMgr.GetCoverPath(shelf.Repo, b.ID)
			hasCover := coverPath != ""

			allItems = append(allItems, tui.BookItem{
				Book:        b,
				ShelfName:   shelf.Name,
				Cached:      cached,
				HasCover:    hasCover,
				CoverPath:   coverPath,
				Owner:       owner,
				Repo:        shelf.Repo,
				CatalogPath: catalogPath,
			})
		}
	}

	return allItems
}

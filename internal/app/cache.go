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
	)

	cmd := &cobra.Command{
		Use:   "clear [book-id...]",
		Short: "Remove books from local cache",
		Long: `Remove books from local cache without deleting them from shelves.

Books will be re-downloaded when opened or browsed.

Examples:
  shelfctl cache clear book-id-1 book-id-2    Remove specific books
  shelfctl cache clear --shelf books          Remove all books from a shelf
  shelfctl cache clear --all                  Remove entire cache
  shelfctl cache clear                        Interactive picker`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				return clearAllCache()
			}

			if shelfName != "" {
				return clearShelfCache(shelfName)
			}

			if len(args) > 0 {
				return clearSpecificBooks(args)
			}

			// Interactive mode
			return clearBooksInteractive(cmd)
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Clear cache for all books on this shelf")
	cmd.Flags().BoolVar(&all, "all", false, "Clear entire cache")

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
func clearAllCache() error {
	cacheDir := cacheMgr.Path("", "", "", "")
	cacheDir = filepath.Dir(cacheDir) // Go up one level to get base dir

	// Check if cache exists
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		ok("Cache is already empty")
		return nil
	}

	// Get size before clearing
	size, count := calculateDirSize(cacheDir)

	// Confirm
	fmt.Printf("This will remove %d cached files (%s)\n", count, humanBytes(size))
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
func clearShelfCache(shelfName string) error {
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

	// Count cached books
	cachedCount := 0
	var cachedSize int64

	for i := range books {
		b := &books[i]
		if cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
			cachedCount++
			path := cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)
			if info, err := os.Stat(path); err == nil {
				cachedSize += info.Size()
			}
		}
	}

	if cachedCount == 0 {
		ok("No cached books on shelf %s", shelfName)
		return nil
	}

	// Confirm
	fmt.Printf("This will remove %d cached books from shelf %s (%s)\n", cachedCount, shelfName, humanBytes(cachedSize))
	fmt.Printf("Type 'CLEAR CACHE' to confirm: ")
	reader := bufio.NewReader(os.Stdin)
	confirmation, _ := reader.ReadString('\n')
	confirmation = strings.TrimSpace(confirmation)

	if confirmation != "CLEAR CACHE" {
		return fmt.Errorf("cancelled")
	}

	// Remove each cached book
	removed := 0
	for i := range books {
		b := &books[i]
		if cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
			if err := cacheMgr.Remove(owner, shelf.Repo, b.ID, b.Source.Asset); err != nil {
				warn("Failed to remove %s: %v", b.ID, err)
				continue
			}
			removed++
		}
	}

	ok("Removed %d books from cache (%s)", removed, humanBytes(cachedSize))
	return nil
}

// clearSpecificBooks removes specific books from cache by ID
func clearSpecificBooks(bookIDs []string) error {
	// Load all shelves to find the books
	allBooks := loadAllBooksAcrossShelves()

	removedCount := 0
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

	return nil
}

// clearBooksInteractive launches interactive picker to select books to clear from cache
func clearBooksInteractive(cmd *cobra.Command) error {
	// Load all cached books
	allBooks := loadAllBooksAcrossShelves()

	// Filter to only cached books
	var cachedBooks []tui.BookItem
	for _, item := range allBooks {
		if item.Cached {
			cachedBooks = append(cachedBooks, item)
		}
	}

	if len(cachedBooks) == 0 {
		ok("No books in cache")
		return nil
	}

	// Launch multi-select picker
	selected, err := tui.RunBookPickerMulti(cachedBooks, "Select books to remove from cache")
	if err != nil {
		return err
	}

	if len(selected) == 0 {
		return fmt.Errorf("no books selected")
	}

	// Calculate total size
	var totalSize int64
	for _, item := range selected {
		path := cacheMgr.Path(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset)
		if info, err := os.Stat(path); err == nil {
			totalSize += info.Size()
		}
	}

	// Confirm
	if len(selected) > 1 {
		fmt.Printf("\nYou are about to remove %d books from cache (%s):\n", len(selected), humanBytes(totalSize))
		for _, item := range selected {
			fmt.Printf("  - %s (%s)\n", item.Book.ID, item.Book.Title)
		}
		fmt.Printf("\nType 'CLEAR %d BOOKS' to confirm: ", len(selected))
		reader := bufio.NewReader(os.Stdin)
		confirmation, _ := reader.ReadString('\n')
		confirmation = strings.TrimSpace(confirmation)
		expected := fmt.Sprintf("CLEAR %d BOOKS", len(selected))
		if confirmation != expected {
			return fmt.Errorf("cancelled")
		}
	}

	// Remove from cache
	removedCount := 0
	for _, item := range selected {
		if err := cacheMgr.Remove(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset); err != nil {
			warn("Failed to remove %s: %v", item.Book.ID, err)
			continue
		}
		removedCount++
	}

	ok("Removed %d books from cache (%s)", removedCount, humanBytes(totalSize))
	return nil
}

// showAllCacheInfo displays cache statistics for all shelves
func showAllCacheInfo() error {
	allBooks := loadAllBooksAcrossShelves()

	totalBooks := len(allBooks)
	cachedCount := 0
	var totalSize int64

	for _, item := range allBooks {
		if item.Cached {
			cachedCount++
			path := cacheMgr.Path(item.Owner, item.Repo, item.Book.ID, item.Book.Source.Asset)
			if info, err := os.Stat(path); err == nil {
				totalSize += info.Size()
			}
		}
	}

	header("Cache Statistics")
	printField("total_books", fmt.Sprintf("%d", totalBooks))
	printField("cached_books", fmt.Sprintf("%d", cachedCount))
	printField("cache_size", humanBytes(totalSize))
	printField("cache_dir", filepath.Dir(cacheMgr.Path("", "", "", "")))

	uncachedCount := totalBooks - cachedCount
	if uncachedCount > 0 {
		fmt.Println()
		fmt.Printf("%s %d books not cached\n", color.YellowString("⚠"), uncachedCount)
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
	var totalSize int64

	for i := range books {
		b := &books[i]
		if cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
			cachedCount++
			path := cacheMgr.Path(owner, shelf.Repo, b.ID, b.Source.Asset)
			if info, err := os.Stat(path); err == nil {
				totalSize += info.Size()
			}
		}
	}

	header("Cache Statistics: %s", shelfName)
	printField("total_books", fmt.Sprintf("%d", totalBooks))
	printField("cached_books", fmt.Sprintf("%d", cachedCount))
	printField("cache_size", humanBytes(totalSize))
	printField("repository", fmt.Sprintf("%s/%s", owner, shelf.Repo))

	uncachedCount := totalBooks - cachedCount
	if uncachedCount > 0 {
		fmt.Println()
		fmt.Printf("%s %d books not cached\n", color.YellowString("⚠"), uncachedCount)
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
				Book:      b,
				ShelfName: shelf.Name,
				Cached:    cached,
				HasCover:  hasCover,
				CoverPath: coverPath,
				Owner:     owner,
				Repo:      shelf.Repo,
			})
		}
	}

	return allItems
}

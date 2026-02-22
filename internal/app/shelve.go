package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/ingest"
	"github.com/blackwell-systems/shelfctl/internal/readme"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/blackwell-systems/shelfctl/internal/util"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)

type shelveParams struct {
	shelfName  string
	releaseTag string
	bookID     string
	title      string
	author     string
	year       int
	tagsCSV    string
	assetName  string
	noPush     bool
	useSHA12   bool
	force      bool
}

type ingestedFile struct {
	tmpPath     string
	sha256      string
	size        int64
	format      string
	srcName     string
	pdfMetadata *ingest.PDFMetadata
}

func newShelveCmd() *cobra.Command {
	params := &shelveParams{}

	cmd := &cobra.Command{
		Use:   "shelve [file|url|github:owner/repo@ref:path]",
		Short: "Add a book to your library",
		Long: `Add a book from a local file, HTTP URL, or GitHub repo path. Uploads to release assets and updates catalog.yml.

If no file is provided, launches an interactive file picker (when in terminal).

Examples:
  shelfctl shelve                                 # Interactive: picker → form → upload
  shelfctl shelve ~/Downloads/sicp.pdf            # Interactive form for metadata
  shelfctl shelve ~/Downloads/sicp.pdf --shelf programming --title "SICP" --author "Abelson & Sussman" --tags lisp,cs
  shelfctl shelve https://example.com/book.pdf --shelf history --title "..." --tags ancient
  shelfctl shelve github:user/repo@main:books/sicp.pdf --shelf programming`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShelve(cmd, args, params)
		},
	}

	cmd.Flags().StringVar(&params.shelfName, "shelf", "", "Target shelf name (interactive prompt if not provided)")
	cmd.Flags().StringVar(&params.releaseTag, "release", "", "Target release tag (default: shelf's default_release)")
	cmd.Flags().StringVar(&params.bookID, "id", "", "Book ID (default: prompt / slugified title)")
	cmd.Flags().BoolVar(&params.useSHA12, "id-sha12", false, "Use first 12 chars of sha256 as ID")
	cmd.Flags().StringVar(&params.title, "title", "", "Book title")
	cmd.Flags().StringVar(&params.author, "author", "", "Author")
	cmd.Flags().IntVar(&params.year, "year", 0, "Publication year")
	cmd.Flags().StringVar(&params.tagsCSV, "tags", "", "Comma-separated tags")
	cmd.Flags().StringVar(&params.assetName, "asset-name", "", "Override asset filename")
	cmd.Flags().BoolVar(&params.noPush, "no-push", false, "Update catalog locally only (do not push)")
	cmd.Flags().BoolVar(&params.force, "force", false, "Skip duplicate checks and overwrite existing assets")

	return cmd
}

func runShelve(cmd *cobra.Command, args []string, params *shelveParams) error {
	useTUIWorkflow := tui.ShouldUseTUI(cmd) && (params.shelfName == "" || len(args) == 0)

	// Step 1: Select shelf
	shelfName, err := selectShelf(params.shelfName, useTUIWorkflow)
	if err != nil {
		return err
	}

	shelf := cfg.ShelfByName(shelfName)
	if shelf == nil {
		return fmt.Errorf("shelf %q not found in config", shelfName)
	}

	// Step 2: Select files (may be multiple in TUI mode)
	inputs, err := selectFile(args, useTUIWorkflow)
	if err != nil {
		return err
	}

	owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
	if params.releaseTag == "" {
		params.releaseTag = shelf.EffectiveRelease(cfg.Defaults.Release)
	}

	// Step 3: Create catalog manager
	catalogPath := shelf.EffectiveCatalogPath()
	catalogMgr := catalog.NewManager(gh, owner, shelf.Repo, catalogPath)

	existingBooks, err := catalogMgr.Load()
	if err != nil {
		return err
	}

	// Step 4: Ensure release once
	rel, err := gh.EnsureRelease(owner, shelf.Repo, params.releaseTag)
	if err != nil {
		return fmt.Errorf("ensuring release: %w", err)
	}

	// Step 5: Process each file
	var newBooks []catalog.Book
	successCount := 0
	failCount := 0

	for i, input := range inputs {
		book, err := processSingleFile(cmd, params, input, i+1, len(inputs), owner, shelf, rel, &existingBooks, useTUIWorkflow)
		if err != nil {
			warn("Failed to process %s: %v", input, err)
			failCount++
			continue
		}

		newBooks = append(newBooks, *book)
		existingBooks = catalog.Append(existingBooks, *book)
		successCount++
	}

	// Step 6: Batch commit catalog and README
	if successCount > 0 {
		if err := batchCommitCatalog(cmd, catalogMgr, owner, shelf.Repo, existingBooks, newBooks, params.noPush); err != nil {
			return err
		}
	}

	// Print summary
	if len(inputs) > 1 {
		fmt.Println()
		if successCount > 0 {
			ok("Successfully added %d books", successCount)
		}
		if failCount > 0 {
			warn("%d books failed", failCount)
		}
	} else if successCount == 1 {
		// Single file: print detailed summary
		book := newBooks[0]
		printBookSummary(cmd, book.ID, book.Title, book.Checksum.SHA256, book.SizeBytes, book.Source.Asset)
	}

	return nil
}

// processSingleFile handles the complete workflow for one file in a batch.
// Returns the catalog book entry or an error.
func processSingleFile(cmd *cobra.Command, params *shelveParams, input string, fileNum, totalFiles int,
	owner string, shelf *config.ShelfConfig, rel *github.Release, existingBooks *[]catalog.Book, useTUI bool) (*catalog.Book, error) {

	// Resolve and ingest source
	src, err := ingest.Resolve(input, cfg.GitHub.Token, cfg.GitHub.APIBase)
	if err != nil {
		return nil, fmt.Errorf("resolve: %w", err)
	}

	ingested, err := ingestFile(src, input)
	if err != nil {
		return nil, fmt.Errorf("ingest: %w", err)
	}
	defer func() { _ = os.Remove(ingested.tmpPath) }()

	// Collect metadata with progress indicator
	metadata, err := collectMetadata(cmd, params, src.Name, ingested, useTUI, fileNum, totalFiles)
	if err != nil {
		return nil, fmt.Errorf("metadata: %w", err)
	}

	// Check duplicates
	if err := checkDuplicates(*existingBooks, ingested.sha256, params.force); err != nil {
		return nil, err
	}

	// Handle asset collisions
	if err := handleAssetCollision(owner, shelf.Repo, rel.ID, metadata.assetName, params.releaseTag, params.force); err != nil {
		return nil, err
	}

	// Upload asset
	if err := uploadAsset(cmd, owner, shelf.Repo, rel.ID, metadata.assetName, ingested.tmpPath, ingested.size, params.releaseTag); err != nil {
		return nil, fmt.Errorf("upload: %w", err)
	}

	// Build catalog entry
	book := buildCatalogEntry(metadata, ingested, owner, shelf.Repo, params.releaseTag)
	return &book, nil
}

func selectShelf(shelfName string, useTUI bool) (string, error) {
	if shelfName != "" {
		return shelfName, nil
	}

	if useTUI {
		if len(cfg.Shelves) == 0 {
			return "", fmt.Errorf("no shelves configured — run 'shelfctl init' first")
		}

		var options []tui.ShelfOption
		for _, s := range cfg.Shelves {
			options = append(options, tui.ShelfOption{
				Name: s.Name,
				Repo: s.Repo,
			})
		}

		return tui.RunShelfPicker(options)
	}

	return "", fmt.Errorf("--shelf flag required in non-interactive mode")
}

func selectFile(args []string, useTUI bool) ([]string, error) {
	if len(args) > 0 {
		return []string{args[0]}, nil
	}

	if useTUI {
		// Try current directory first (most likely to have files)
		startPath, err := os.Getwd()
		if err != nil {
			// Fall back to Downloads
			home := os.Getenv("HOME")
			startPath = filepath.Join(home, "Downloads")
			if _, err := os.Stat(startPath); err != nil {
				// Fall back to home
				startPath = home
			}
		}

		return tui.RunFilePickerMulti(startPath)
	}

	return nil, fmt.Errorf("file path required in non-interactive mode")
}

func ingestFile(src *ingest.Source, input string) (*ingestedFile, error) {
	if util.IsTTY() {
		fmt.Printf("Ingesting %s …\n", color.CyanString(input))
	}
	rc, err := src.Open()
	if err != nil {
		return nil, fmt.Errorf("opening source: %w", err)
	}

	tmp, err := os.CreateTemp("", "shelfctl-add-*")
	if err != nil {
		_ = rc.Close()
		return nil, err
	}
	tmpPath := tmp.Name()

	hr := ingest.NewReader(rc)
	if _, err := io.Copy(tmp, hr); err != nil {
		_ = tmp.Close()
		_ = rc.Close()
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("buffering source: %w", err)
	}
	_ = tmp.Close()
	_ = rc.Close()

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(src.Name), "."))

	result := &ingestedFile{
		tmpPath: tmpPath,
		sha256:  hr.SHA256(),
		size:    hr.Size(),
		format:  ext,
		srcName: src.Name,
	}

	// Extract PDF metadata if it's a PDF
	if ext == "pdf" {
		if pdfMeta, err := ingest.ExtractPDFMetadata(tmpPath); err == nil {
			result.pdfMetadata = pdfMeta
		}
		// Silently ignore errors - not all PDFs have metadata
	}

	return result, nil
}

type bookMetadata struct {
	bookID    string
	title     string
	author    string
	year      int
	tags      []string
	assetName string
}

func collectMetadata(cmd *cobra.Command, params *shelveParams, srcName string, ingested *ingestedFile, useTUI bool, fileNum, totalFiles int) (*bookMetadata, error) {
	defaultTitle := strings.TrimSuffix(srcName, filepath.Ext(srcName))
	defaultAuthor := ""

	// Use PDF metadata if available
	if ingested.pdfMetadata != nil {
		if ingested.pdfMetadata.Title != "" {
			defaultTitle = ingested.pdfMetadata.Title
		}
		if ingested.pdfMetadata.Author != "" {
			defaultAuthor = ingested.pdfMetadata.Author
		}
	}

	defaultID := slugify(defaultTitle)

	useTUIForm := useTUI && params.title == "" && params.bookID == "" && !params.useSHA12

	// Add progress indicator to filename for multi-file batches
	displayName := srcName
	if totalFiles > 1 {
		displayName = fmt.Sprintf("[%d/%d] %s", fileNum, totalFiles, srcName)
	}

	var title, author, bookID, tagsCSV string
	if useTUIForm {
		formData, err := tui.RunShelveForm(tui.ShelveFormDefaults{
			Filename: displayName,
			Title:    defaultTitle,
			Author:   defaultAuthor,
			ID:       defaultID,
		})
		if err != nil {
			return nil, fmt.Errorf("form canceled or failed: %w", err)
		}

		title = formData.Title
		author = formData.Author
		tagsCSV = formData.Tags
		bookID = formData.ID
	} else {
		title = params.title
		author = params.author
		tagsCSV = params.tagsCSV
		bookID = params.bookID

		if title == "" {
			title = promptOrDefault("Title", defaultTitle)
		}
	}

	// Determine asset filename
	assetName := params.assetName
	if assetName == "" {
		naming := cfg.Defaults.AssetNaming
		if naming == "original" {
			assetName = srcName
		}
	}

	// Determine book ID
	if bookID == "" {
		if params.useSHA12 {
			bookID = ingested.sha256[:12]
		} else {
			bookID = promptOrDefault("ID", slugify(title))
		}
	}

	if !idRe.MatchString(bookID) {
		return nil, fmt.Errorf("invalid ID %q — must match ^[a-z0-9][a-z0-9-]{1,62}$", bookID)
	}

	// Finalize asset name
	if assetName == "" {
		assetName = bookID + "." + ingested.format
	}

	// Parse tags
	var tags []string
	if tagsCSV != "" {
		for _, t := range strings.Split(tagsCSV, ",") {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}
	}

	return &bookMetadata{
		bookID:    bookID,
		title:     title,
		author:    author,
		year:      params.year,
		tags:      tags,
		assetName: assetName,
	}, nil
}

// loadCatalog is deprecated - use catalog.Manager.Load() instead.
// Kept for backward compatibility with other commands.
func loadCatalog(owner, repo, catalogPath string) ([]catalog.Book, error) {
	mgr := catalog.NewManager(gh, owner, repo, catalogPath)
	return mgr.Load()
}

func checkDuplicates(existingBooks []catalog.Book, sha256 string, force bool) error {
	if force {
		return nil
	}

	for _, b := range existingBooks {
		if b.Checksum.SHA256 == sha256 {
			warn("File with same SHA256 already exists: %s (%s)", b.ID, b.Title)
			fmt.Printf("Use --force to add anyway, or skip.\n")
			return fmt.Errorf("duplicate content detected")
		}
	}
	return nil
}

func handleAssetCollision(owner, repo string, releaseID int64, assetName, releaseTag string, force bool) error {
	existingAsset, err := gh.FindAsset(owner, repo, releaseID, assetName)
	if err != nil {
		return fmt.Errorf("checking existing assets: %w", err)
	}

	if existingAsset == nil {
		return nil
	}

	if !force {
		warn("Asset with name %q already exists in release %s", assetName, releaseTag)
		fmt.Printf("Use --force to overwrite, --asset-name for different name, or delete existing asset first.\n")
		return fmt.Errorf("asset name collision")
	}

	warn("Deleting existing asset %q", assetName)
	if err := gh.DeleteAsset(owner, repo, existingAsset.ID); err != nil {
		return fmt.Errorf("deleting existing asset: %w", err)
	}
	return nil
}

func uploadAsset(cmd *cobra.Command, owner, repo string, releaseID int64, assetName, tmpPath string, size int64, releaseTag string) error {
	uploadFile, err := os.Open(tmpPath)
	if err != nil {
		return err
	}
	defer func() { _ = uploadFile.Close() }()

	// Use progress bar in TTY mode
	var asset *github.Asset
	if util.IsTTY() && tui.ShouldUseTUI(cmd) {
		progressCh := make(chan int64, 10)
		errCh := make(chan error, 1)

		// Start upload in goroutine
		go func() {
			pr := tui.NewProgressReader(uploadFile, size, progressCh)
			a, err := gh.UploadAsset(owner, repo, releaseID, assetName, pr, size, "application/octet-stream")
			close(progressCh)
			asset = a
			errCh <- err
		}()

		// Show progress UI
		label := fmt.Sprintf("Uploading %s → %s/%s/%s", assetName, owner, repo, releaseTag)
		if err := tui.ShowProgress(label, size, progressCh); err != nil {
			return err // User cancelled
		}

		// Get result
		if err := <-errCh; err != nil {
			return fmt.Errorf("uploading: %w", err)
		}
	} else {
		// Non-interactive mode: just print and upload
		fmt.Printf("Uploading %s → %s/%s/%s …\n", assetName, owner, repo, releaseTag)
		asset, err = gh.UploadAsset(owner, repo, releaseID, assetName, uploadFile, size, "application/octet-stream")
		if err != nil {
			return fmt.Errorf("uploading: %w", err)
		}
	}

	if tui.ShouldUseTUI(cmd) {
		ok("Uploaded: %s", asset.BrowserDownloadURL)
	}
	return nil
}

func buildCatalogEntry(metadata *bookMetadata, ingested *ingestedFile, owner, repo, releaseTag string) catalog.Book {
	return catalog.Book{
		ID:        metadata.bookID,
		Title:     metadata.title,
		Author:    metadata.author,
		Year:      metadata.year,
		Tags:      metadata.tags,
		Format:    ingested.format,
		SizeBytes: ingested.size,
		Checksum:  catalog.Checksum{SHA256: ingested.sha256},
		Source: catalog.Source{
			Type:    "github_release",
			Owner:   owner,
			Repo:    repo,
			Release: releaseTag,
			Asset:   metadata.assetName,
		},
		Meta: catalog.Meta{
			AddedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}
}

func updateCatalog(cmd *cobra.Command, owner, repo, catalogPath string, existingBooks []catalog.Book, book catalog.Book, noPush bool) error {
	books := catalog.Append(existingBooks, book)
	newCatalog, err := catalog.Marshal(books)
	if err != nil {
		return err
	}

	if noPush {
		if err := os.WriteFile(catalogPath, newCatalog, 0600); err != nil {
			return err
		}
		if tui.ShouldUseTUI(cmd) {
			ok("Catalog updated locally (not pushed)")
		}
		return nil
	}

	msg := fmt.Sprintf("add: %s — %s", book.ID, book.Title)
	if err := gh.CommitFile(owner, repo, catalogPath, newCatalog, msg); err != nil {
		return fmt.Errorf("committing catalog: %w", err)
	}
	if tui.ShouldUseTUI(cmd) {
		ok("Catalog committed and pushed")
	}

	updateREADME(cmd, owner, repo, books, book)
	return nil
}

func updateREADME(cmd *cobra.Command, owner, repo string, books []catalog.Book, book catalog.Book) {
	readmeData, _, readmeErr := gh.GetFileContent(owner, repo, "README.md", "")
	if readmeErr != nil {
		return
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
		if tui.ShouldUseTUI(cmd) {
			warn("Could not update README.md: %v", err)
		}
	} else {
		if tui.ShouldUseTUI(cmd) {
			ok("README.md updated")
		}
	}
}

// batchCommitCatalog commits the catalog with all new books in a single commit.
func batchCommitCatalog(cmd *cobra.Command, catalogMgr *catalog.Manager, owner, repo string, allBooks []catalog.Book, newBooks []catalog.Book, noPush bool) error {
	if noPush {
		// Local-only mode - write to local file
		data, err := catalog.Marshal(allBooks)
		if err != nil {
			return err
		}
		// Write to catalog.yml in current directory
		if err := os.WriteFile("catalog.yml", data, 0600); err != nil {
			return err
		}
		if tui.ShouldUseTUI(cmd) {
			ok("Catalog updated locally (not pushed)")
		}
		return nil
	}

	// Create commit message based on count
	var msg string
	if len(newBooks) == 1 {
		msg = fmt.Sprintf("add: %s — %s", newBooks[0].ID, newBooks[0].Title)
	} else {
		msg = fmt.Sprintf("add: %d books", len(newBooks))
	}

	// Save catalog using manager
	if err := catalogMgr.Save(allBooks, msg); err != nil {
		return err
	}
	if tui.ShouldUseTUI(cmd) {
		ok("Catalog committed and pushed")
	}

	// Update README with all new books
	readmeMgr := readme.NewUpdater(gh, owner, repo)
	if err := readmeMgr.UpdateWithStats(len(allBooks), newBooks); err != nil {
		if tui.ShouldUseTUI(cmd) {
			warn("Could not update README.md: %v", err)
		}
	} else {
		if tui.ShouldUseTUI(cmd) {
			ok("README.md updated")
		}
	}

	return nil
}

// updateREADMEBatch is deprecated - use readme.Updater.UpdateWithStats() instead.
// Kept for backward compatibility.
func updateREADMEBatch(cmd *cobra.Command, owner, repo string, allBooks []catalog.Book, newBooks []catalog.Book) {
	readmeMgr := readme.NewUpdater(gh, owner, repo)
	if err := readmeMgr.UpdateWithStats(len(allBooks), newBooks); err != nil {
		if tui.ShouldUseTUI(cmd) {
			warn("Could not update README.md: %v", err)
		}
	} else {
		if tui.ShouldUseTUI(cmd) {
			ok("README.md updated")
		}
	}
}

func printBookSummary(cmd *cobra.Command, bookID, title, sha256 string, size int64, assetName string) {
	if tui.ShouldUseTUI(cmd) {
		// Interactive mode: verbose formatted output
		fmt.Println()
		fmt.Printf("  id:      %s\n", color.WhiteString(bookID))
		fmt.Printf("  title:   %s\n", title)
		fmt.Printf("  sha256:  %s\n", sha256)
		fmt.Printf("  size:    %s\n", humanBytes(size))
		fmt.Printf("  asset:   %s\n", assetName)
	} else {
		// Script mode: just print the book ID for easy parsing
		fmt.Println(bookID)
	}
}

// promptOrDefault reads a line from stdin, falling back to def on empty input.
func promptOrDefault(label, def string) string {
	fmt.Printf("%s [%s]: ", label, def)
	sc := bufio.NewScanner(os.Stdin)
	if sc.Scan() {
		if v := strings.TrimSpace(sc.Text()); v != "" {
			return v
		}
	}
	return def
}

// slugify converts a title to a lowercase, hyphenated ID candidate.
// Apostrophes and quotes are removed. Other non-alphanumeric characters
// collapse into a single hyphen.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevWasSep := false
	for _, r := range s {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		// Apostrophes and quotes are silently removed (don't separate words)
		isQuote := r == '\'' || r == '"' ||
			r == '\u2018' || r == '\u2019' || // Left/right single quotation marks
			r == '\u201C' || r == '\u201D' // Left/right double quotation marks

		if isAlnum {
			b.WriteRune(r)
			prevWasSep = false
		} else if isQuote {
			// Skip quotes entirely - don't write anything, don't mark as separator
			continue
		} else {
			// Other non-alphanumeric characters become separators
			if !prevWasSep && b.Len() > 0 {
				b.WriteRune('-')
			}
			prevWasSep = true
		}
	}
	result := strings.TrimRight(b.String(), "-")
	if len(result) > 63 {
		result = result[:63]
	}
	if result == "" {
		return "book"
	}
	return result
}

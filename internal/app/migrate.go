package app

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/ingest"
	"github.com/blackwell-systems/shelfctl/internal/migrate"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate books from an old monorepo",
	}
	cmd.AddCommand(newMigrateOneCmd(), newMigrateBatchCmd(), newMigrateScanCmd())
	return cmd
}

func newMigrateOneCmd() *cobra.Command {
	var (
		sourceSpec string
		noPush     bool
	)

	cmd := &cobra.Command{
		Use:   "one <old_path>",
		Short: "Migrate a single file from the configured source repo",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldPath := args[0]

			ledger, err := migrate.OpenLedger(migrate.DefaultLedgerPath())
			if err != nil {
				return err
			}

			if done, err := ledger.Contains(oldPath); err != nil {
				return err
			} else if done {
				ok("Already migrated: %s", oldPath)
				return nil
			}

			// Resolve migration sources to search.
			sources, err := resolveMigrationSources(sourceSpec)
			if err != nil {
				return err
			}

			// Find and migrate the file
			return migrateOneFile(oldPath, sources, ledger, noPush)
		},
	}

	cmd.Flags().StringVar(&sourceSpec, "source", "", "Override source as owner/repo")
	cmd.Flags().BoolVar(&noPush, "no-push", false, "Update catalog locally only")
	return cmd
}

func newMigrateBatchCmd() *cobra.Command {
	var (
		n      int
		cont   bool
		dryRun bool
		noPush bool
	)

	cmd := &cobra.Command{
		Use:   "batch <queue_file>",
		Short: "Migrate a queue of files (one path per line)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			queueFile := args[0]
			f, err := os.Open(queueFile)
			if err != nil {
				return err
			}
			defer func() { _ = f.Close() }()

			ledger, err := migrate.OpenLedger(migrate.DefaultLedgerPath())
			if err != nil {
				return err
			}

			processed, skipped := processMigrationQueue(f, ledger, n, cont, dryRun, noPush)
			fmt.Printf("\nDone. processed=%d skipped=%d\n", processed, skipped)
			return nil
		},
	}

	cmd.Flags().IntVarP(&n, "n", "n", 0, "Max files per run (0 = unlimited)")
	cmd.Flags().BoolVar(&cont, "continue", false, "Skip already-migrated files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print paths without migrating")
	cmd.Flags().BoolVar(&noPush, "no-push", false, "Update catalog locally only")
	return cmd
}

func newMigrateScanCmd() *cobra.Command {
	var (
		sourceSpec string
		extsCSV    string
		outFile    string
	)

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "List files in a source repo (outputs a queue for 'migrate batch')",
		RunE: func(cmd *cobra.Command, args []string) error {
			sources := cfg.Migration.Sources
			if len(sources) == 0 && sourceSpec == "" {
				return fmt.Errorf("no migration sources configured; use --source owner/repo")
			}

			exts := parseExtensions(extsCSV)
			files, err := scanMigrationSources(sourceSpec, sources, exts)
			if err != nil {
				return err
			}

			return writeFileList(files, outFile)
		},
	}

	cmd.Flags().StringVar(&sourceSpec, "source", "", "Scan owner/repo (overrides config sources)")
	cmd.Flags().StringVar(&extsCSV, "ext", "", "Filter by comma-separated extensions (e.g. pdf,epub)")
	cmd.Flags().StringVar(&outFile, "out", "", "Write to file instead of stdout")
	return cmd
}

// Helper functions for migrate one command

func resolveMigrationSources(sourceSpec string) ([]config.MigrationSource, error) {
	sources := cfg.Migration.Sources
	if sourceSpec == "" {
		return sources, nil
	}

	parts := strings.SplitN(sourceSpec, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("--source must be owner/repo")
	}

	var filtered []config.MigrationSource
	for _, s := range sources {
		if s.Owner == parts[0] && s.Repo == parts[1] {
			filtered = append(filtered, s)
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("source %q not found in migration config", sourceSpec)
	}
	return filtered, nil
}

func migrateOneFile(oldPath string, sources []config.MigrationSource, ledger *migrate.Ledger, noPush bool) error {
	src, shelfName, found := migrate.FindRoute(oldPath, sources)
	if !found {
		return fmt.Errorf("no migration mapping matches path %q", oldPath)
	}

	shelf, err := resolveOrCreateShelf(shelfName)
	if err != nil {
		return err
	}

	// Fetch and process the file
	fileData, sha256sum, size, err := fetchSourceFile(src, oldPath)
	if err != nil {
		return err
	}

	// Upload to destination
	suggestedID, assetName, err := uploadMigratedFile(shelf, oldPath, fileData, size)
	if err != nil {
		return err
	}

	// Update catalog
	book := buildMigratedBook(suggestedID, assetName, oldPath, sha256sum, size, shelf, src)
	if err := updateCatalogWithBook(shelf, book, src, noPush); err != nil {
		return err
	}

	// Update ledger
	if err := ledger.Append(migrate.LedgerEntry{
		Source: oldPath,
		BookID: suggestedID,
		Shelf:  shelfName,
	}); err != nil {
		warn("Could not update ledger: %v", err)
	}

	fmt.Printf("  %s → shelf/%s  id=%s\n",
		color.CyanString(oldPath), shelfName, color.WhiteString(suggestedID))
	return nil
}

func fetchSourceFile(src config.MigrationSource, oldPath string) ([]byte, string, int64, error) {
	ref := src.Ref
	if ref == "" {
		ref = "main"
	}

	fmt.Printf("Fetching %s from %s/%s@%s …\n", oldPath, src.Owner, src.Repo, ref)
	fileData, _, err := gh.GetFileContent(src.Owner, src.Repo, oldPath, ref)
	if err != nil {
		return nil, "", 0, fmt.Errorf("fetching source file: %w", err)
	}

	// Compute hash and size.
	hr := ingest.NewReader(bytes.NewReader(fileData))
	buf := make([]byte, 32*1024)
	for {
		_, readErr := hr.Read(buf)
		if readErr != nil {
			break
		}
	}

	return fileData, hr.SHA256(), hr.Size(), nil
}

func uploadMigratedFile(shelf *config.ShelfConfig, oldPath string, fileData []byte, size int64) (string, string, error) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(oldPath), "."))
	baseName := filepath.Base(oldPath)
	suggestedID := slugify(strings.TrimSuffix(baseName, filepath.Ext(baseName)))

	owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
	releaseTag := shelf.EffectiveRelease(cfg.Defaults.Release)
	assetName := suggestedID + "." + ext

	rel, err := gh.EnsureRelease(owner, shelf.Repo, releaseTag)
	if err != nil {
		return "", "", err
	}

	_, err = gh.UploadAsset(owner, shelf.Repo, rel.ID, assetName,
		bytes.NewReader(fileData), size, "application/octet-stream")
	if err != nil {
		return "", "", fmt.Errorf("uploading: %w", err)
	}
	ok("Uploaded %s", assetName)

	return suggestedID, assetName, nil
}

func buildMigratedBook(suggestedID, assetName, oldPath, sha256sum string, size int64,
	shelf *config.ShelfConfig, src config.MigrationSource) catalog.Book {
	owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
	releaseTag := shelf.EffectiveRelease(cfg.Defaults.Release)
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(oldPath), "."))
	baseName := filepath.Base(oldPath)

	return catalog.Book{
		ID:        suggestedID,
		Title:     strings.TrimSuffix(baseName, filepath.Ext(baseName)),
		Format:    ext,
		SizeBytes: size,
		Checksum:  catalog.Checksum{SHA256: sha256sum},
		Source: catalog.Source{
			Type:    "github_release",
			Owner:   owner,
			Repo:    shelf.Repo,
			Release: releaseTag,
			Asset:   assetName,
		},
		Meta: catalog.Meta{
			AddedAt:      time.Now().UTC().Format(time.RFC3339),
			MigratedFrom: fmt.Sprintf("%s/%s:%s", src.Owner, src.Repo, oldPath),
		},
	}
}

func updateCatalogWithBook(shelf *config.ShelfConfig, book catalog.Book, src config.MigrationSource, noPush bool) error {
	owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
	catalogPath := shelf.EffectiveCatalogPath()

	data, _, _ := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
	books, _ := catalog.Parse(data)
	books = catalog.Append(books, book)
	newData, err := catalog.Marshal(books)
	if err != nil {
		return err
	}

	if !noPush {
		msg := fmt.Sprintf("migrate: add %s (from %s/%s)", book.ID, src.Owner, src.Repo)
		if err := gh.CommitFile(owner, shelf.Repo, catalogPath, newData, msg); err != nil {
			return err
		}
		ok("Catalog updated")
	}

	return nil
}

// Helper functions for migrate batch command

func processMigrationQueue(f *os.File, ledger *migrate.Ledger, n int, cont, dryRun, noPush bool) (int, int) {
	sc := bufio.NewScanner(f)
	processed := 0
	skipped := 0

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if n > 0 && processed >= n {
			fmt.Printf("Limit of %d reached. Re-run to continue.\n", n)
			break
		}

		if cont {
			done, _ := ledger.Contains(line)
			if done {
				skipped++
				continue
			}
		}

		if dryRun {
			fmt.Printf("  would migrate: %s\n", line)
			processed++
			continue
		}

		fmt.Printf("[%d] %s …\n", processed+1, line)
		oneCmd := newMigrateOneCmd()
		oneArgs := []string{line}
		if noPush {
			oneArgs = append(oneArgs, "--no-push")
		}
		oneCmd.SetArgs(oneArgs)
		if err := oneCmd.Execute(); err != nil {
			warn("Failed: %v", err)
		}
		processed++
	}

	return processed, skipped
}

// Helper functions for migrate scan command

func parseExtensions(extsCSV string) []string {
	var exts []string
	if extsCSV != "" {
		for _, e := range strings.Split(extsCSV, ",") {
			exts = append(exts, strings.TrimSpace(e))
		}
	}
	return exts
}

func scanMigrationSources(sourceSpec string, sources []config.MigrationSource, exts []string) ([]migrate.FileEntry, error) {
	var files []migrate.FileEntry

	if sourceSpec != "" {
		return scanSingleSource(sourceSpec, sources, exts)
	}

	for _, src := range sources {
		ref := src.Ref
		if ref == "" {
			ref = "main"
		}
		f, err := migrate.ScanRepo(cfg.GitHub.Token, cfg.GitHub.APIBase, src.Owner, src.Repo, ref, exts)
		if err != nil {
			warn("Scan %s/%s: %v", src.Owner, src.Repo, err)
			continue
		}
		files = append(files, f...)
	}

	return files, nil
}

func scanSingleSource(sourceSpec string, sources []config.MigrationSource, exts []string) ([]migrate.FileEntry, error) {
	parts := strings.SplitN(sourceSpec, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("--source must be owner/repo")
	}

	ref := "main"
	for _, s := range sources {
		if s.Owner == parts[0] && s.Repo == parts[1] && s.Ref != "" {
			ref = s.Ref
		}
	}

	return migrate.ScanRepo(cfg.GitHub.Token, cfg.GitHub.APIBase, parts[0], parts[1], ref, exts)
}

func writeFileList(files []migrate.FileEntry, outFile string) error {
	out := os.Stdout
	if outFile != "" {
		var err error
		out, err = os.Create(outFile)
		if err != nil {
			return err
		}
		defer func() { _ = out.Close() }()
	}

	for _, fe := range files {
		_, _ = fmt.Fprintln(out, fe.Path)
	}

	if outFile != "" {
		ok("Wrote %d paths to %s", len(files), outFile)
	} else {
		fmt.Fprintf(os.Stderr, "# %d files\n", len(files))
	}
	return nil
}

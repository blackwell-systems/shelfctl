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
	"github.com/blackwell-systems/shelfctl/internal/ingest"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)

func newShelveCmd() *cobra.Command {
	var (
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
	)

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
			// If in TUI mode and missing required inputs, launch interactive workflow
			useTUIWorkflow := tui.ShouldUseTUI(cmd) && (shelfName == "" || len(args) == 0)

			// Step 1: Select shelf if not provided
			if shelfName == "" {
				if useTUIWorkflow {
					if len(cfg.Shelves) == 0 {
						return fmt.Errorf("no shelves configured — run 'shelfctl init' first")
					}

					// Build shelf options
					var options []tui.ShelfOption
					for _, s := range cfg.Shelves {
						options = append(options, tui.ShelfOption{
							Name: s.Name,
							Repo: s.Repo,
						})
					}

					selected, err := tui.RunShelfPicker(options)
					if err != nil {
						return err
					}
					shelfName = selected
				} else {
					return fmt.Errorf("--shelf flag required in non-interactive mode")
				}
			}

			shelf := cfg.ShelfByName(shelfName)
			if shelf == nil {
				return fmt.Errorf("shelf %q not found in config", shelfName)
			}

			// Step 2: Select file if not provided
			var input string
			if len(args) == 0 {
				if useTUIWorkflow {
					// Get starting directory (try Downloads first, then home)
					home := os.Getenv("HOME")
					startPath := filepath.Join(home, "Downloads")
					if _, err := os.Stat(startPath); err != nil {
						startPath = home
					}

					selected, err := tui.RunFilePicker(startPath)
					if err != nil {
						return err
					}
					input = selected
				} else {
					return fmt.Errorf("file path required in non-interactive mode")
				}
			} else {
				input = args[0]
			}
			owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
			if releaseTag == "" {
				releaseTag = shelf.EffectiveRelease(cfg.Defaults.Release)
			}

			// Resolve input source.
			src, err := ingest.Resolve(input, cfg.GitHub.Token, cfg.GitHub.APIBase)
			if err != nil {
				return err
			}

			// Determine extension and format.
			ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(src.Name), "."))
			format := ext

			// Calculate defaults for prompts/form
			defaultTitle := strings.TrimSuffix(src.Name, filepath.Ext(src.Name))
			defaultID := slugify(defaultTitle)

			// Check if we should use TUI form for metadata entry
			useTUIForm := tui.ShouldUseTUI(cmd) && title == "" && bookID == "" && !useSHA12

			if useTUIForm {
				// Launch interactive form
				formData, err := tui.RunShelveForm(tui.ShelveFormDefaults{
					Filename: src.Name,
					Title:    defaultTitle,
					ID:       defaultID,
				})
				if err != nil {
					return fmt.Errorf("form canceled or failed: %w", err)
				}

				// Use form data
				title = formData.Title
				author = formData.Author
				tagsCSV = formData.Tags
				bookID = formData.ID
			} else {
				// Non-TUI mode: use flags or bufio prompts
				// Determine title (required).
				if title == "" {
					title = promptOrDefault("Title", defaultTitle)
				}
			}

			// Determine asset filename.
			if assetName == "" {
				naming := cfg.Defaults.AssetNaming
				if naming == "original" {
					assetName = src.Name
				}
				// "id" naming determined after we have the ID.
			}

			// Open the source and compute hash + size.
			fmt.Printf("Ingesting %s …\n", color.CyanString(input))
			rc, err := src.Open()
			if err != nil {
				return fmt.Errorf("opening source: %w", err)
			}

			// Buffer to a temp file so we know the size before uploading.
			tmp, err := os.CreateTemp("", "shelfctl-add-*")
			if err != nil {
				_ = rc.Close()
				return err
			}
			tmpPath := tmp.Name()
			defer func() { _ = os.Remove(tmpPath) }()

			hr := ingest.NewReader(rc)
			if _, err := io.Copy(tmp, hr); err != nil {
				_ = tmp.Close()
				_ = rc.Close()
				return fmt.Errorf("buffering source: %w", err)
			}
			_ = tmp.Close()
			_ = rc.Close()

			sha256 := hr.SHA256()
			size := hr.Size()

			// Determine book ID (if not already set by TUI form or flags).
			if bookID == "" {
				if useSHA12 {
					bookID = sha256[:12]
				} else {
					bookID = promptOrDefault("ID", slugify(title))
				}
			}
			if !idRe.MatchString(bookID) {
				return fmt.Errorf("invalid ID %q — must match ^[a-z0-9][a-z0-9-]{1,62}$", bookID)
			}

			// Finalize asset name now that we have the ID.
			if assetName == "" {
				assetName = bookID + "." + ext
			}

			// Parse tags.
			var tags []string
			if tagsCSV != "" {
				for _, t := range strings.Split(tagsCSV, ",") {
					if t = strings.TrimSpace(t); t != "" {
						tags = append(tags, t)
					}
				}
			}

			// Load existing catalog for duplicate checks.
			catalogPath := shelf.EffectiveCatalogPath()
			existingData, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
			if err != nil && err.Error() != "not found" {
				return fmt.Errorf("reading catalog: %w", err)
			}
			existingBooks, _ := catalog.Parse(existingData)

			// Check for content duplication by SHA256.
			if !force {
				for _, b := range existingBooks {
					if b.Checksum.SHA256 == sha256 {
						warn("File with same SHA256 already exists: %s (%s)", b.ID, b.Title)
						fmt.Printf("Use --force to add anyway, or skip.\n")
						return fmt.Errorf("duplicate content detected")
					}
				}
			}

			// Ensure release exists.
			rel, err := gh.EnsureRelease(owner, shelf.Repo, releaseTag)
			if err != nil {
				return fmt.Errorf("ensuring release: %w", err)
			}

			// Check for asset name collision.
			if !force {
				existingAsset, err := gh.FindAsset(owner, shelf.Repo, rel.ID, assetName)
				if err != nil {
					return fmt.Errorf("checking existing assets: %w", err)
				}
				if existingAsset != nil {
					warn("Asset with name %q already exists in release %s", assetName, releaseTag)
					fmt.Printf("Use --force to overwrite, --asset-name for different name, or delete existing asset first.\n")
					return fmt.Errorf("asset name collision")
				}
			} else {
				// Force mode: delete existing asset if it exists.
				existingAsset, err := gh.FindAsset(owner, shelf.Repo, rel.ID, assetName)
				if err != nil {
					return fmt.Errorf("checking existing assets: %w", err)
				}
				if existingAsset != nil {
					warn("Deleting existing asset %q", assetName)
					if err := gh.DeleteAsset(owner, shelf.Repo, existingAsset.ID); err != nil {
						return fmt.Errorf("deleting existing asset: %w", err)
					}
				}
			}

			// Upload asset.
			fmt.Printf("Uploading %s → %s/%s/%s …\n",
				assetName, owner, shelf.Repo, releaseTag)

			uploadFile, err := os.Open(tmpPath)
			if err != nil {
				return err
			}
			defer func() { _ = uploadFile.Close() }()

			asset, err := gh.UploadAsset(owner, shelf.Repo, rel.ID, assetName, uploadFile, size, "application/octet-stream")
			if err != nil {
				return fmt.Errorf("uploading: %w", err)
			}
			ok("Uploaded: %s", asset.BrowserDownloadURL)

			// Build catalog entry.
			book := catalog.Book{
				ID:        bookID,
				Title:     title,
				Author:    author,
				Year:      year,
				Tags:      tags,
				Format:    format,
				SizeBytes: size,
				Checksum:  catalog.Checksum{SHA256: sha256},
				Source: catalog.Source{
					Type:    "github_release",
					Owner:   owner,
					Repo:    shelf.Repo,
					Release: releaseTag,
					Asset:   assetName,
				},
				Meta: catalog.Meta{
					AddedAt: time.Now().UTC().Format(time.RFC3339),
				},
			}

			// Append to catalog (we already loaded it earlier for duplicate checks).
			books := catalog.Append(existingBooks, book)

			newCatalog, err := catalog.Marshal(books)
			if err != nil {
				return err
			}

			if noPush {
				// Write locally only.
				if err := os.WriteFile(catalogPath, newCatalog, 0600); err != nil {
					return err
				}
				ok("Catalog updated locally (not pushed)")
			} else {
				msg := fmt.Sprintf("add: %s — %s", bookID, title)
				if err := gh.CommitFile(owner, shelf.Repo, catalogPath, newCatalog, msg); err != nil {
					return fmt.Errorf("committing catalog: %w", err)
				}
				ok("Catalog committed and pushed")
			}

			fmt.Println()
			fmt.Printf("  id:      %s\n", color.WhiteString(bookID))
			fmt.Printf("  title:   %s\n", title)
			fmt.Printf("  sha256:  %s\n", sha256)
			fmt.Printf("  size:    %s\n", humanBytes(size))
			fmt.Printf("  asset:   %s\n", assetName)
			return nil
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Target shelf name (interactive prompt if not provided)")
	cmd.Flags().StringVar(&releaseTag, "release", "", "Target release tag (default: shelf's default_release)")
	cmd.Flags().StringVar(&bookID, "id", "", "Book ID (default: prompt / slugified title)")
	cmd.Flags().BoolVar(&useSHA12, "id-sha12", false, "Use first 12 chars of sha256 as ID")
	cmd.Flags().StringVar(&title, "title", "", "Book title")
	cmd.Flags().StringVar(&author, "author", "", "Author")
	cmd.Flags().IntVar(&year, "year", 0, "Publication year")
	cmd.Flags().StringVar(&tagsCSV, "tags", "", "Comma-separated tags")
	cmd.Flags().StringVar(&assetName, "asset-name", "", "Override asset filename")
	cmd.Flags().BoolVar(&noPush, "no-push", false, "Update catalog locally only (do not push)")
	cmd.Flags().BoolVar(&force, "force", false, "Skip duplicate checks and overwrite existing assets")

	return cmd
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
// Consecutive non-alphanumeric characters collapse into a single hyphen.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevWasSep := false
	for _, r := range s {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlnum {
			b.WriteRune(r)
			prevWasSep = false
		} else {
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

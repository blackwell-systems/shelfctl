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
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)

func newAddCmd() *cobra.Command {
	var (
		shelfName   string
		releaseTag  string
		bookID      string
		title       string
		author      string
		year        int
		tagsCSV     string
		assetName   string
		noPush      bool
		useSHA12    bool
	)

	cmd := &cobra.Command{
		Use:   "add <file|url|github:owner/repo@ref:path>",
		Short: "Ingest a document into a shelf",
		Long: `Ingest a local file, HTTP URL, or GitHub repo path into a shelf's release assets and update catalog.yml.

Examples:
  shelfctl add ~/Downloads/sicp.pdf --shelf programming --title "SICP" --author "Abelson & Sussman" --tags lisp,cs
  shelfctl add https://example.com/book.pdf --shelf history --title "..." --tags ancient
  shelfctl add github:user/repo@main:books/sicp.pdf --shelf programming`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := args[0]

			shelf := cfg.ShelfByName(shelfName)
			if shelf == nil {
				return fmt.Errorf("shelf %q not found in config", shelfName)
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

			// Determine title (required).
			if title == "" {
				title = promptOrDefault("Title", strings.TrimSuffix(src.Name, filepath.Ext(src.Name)))
			}

			// Determine extension and format.
			ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(src.Name), "."))
			format := ext

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
				rc.Close()
				return err
			}
			tmpPath := tmp.Name()
			defer os.Remove(tmpPath)

			hr := ingest.NewReader(rc)
			if _, err := io.Copy(tmp, hr); err != nil {
				tmp.Close()
				rc.Close()
				return fmt.Errorf("buffering source: %w", err)
			}
			tmp.Close()
			rc.Close()

			sha256 := hr.SHA256()
			size := hr.Size()

			// Determine book ID.
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

			// Ensure release exists.
			rel, err := gh.EnsureRelease(owner, shelf.Repo, releaseTag)
			if err != nil {
				return fmt.Errorf("ensuring release: %w", err)
			}

			// Upload asset.
			fmt.Printf("Uploading %s → %s/%s/%s …\n",
				assetName, owner, shelf.Repo, releaseTag)

			uploadFile, err := os.Open(tmpPath)
			if err != nil {
				return err
			}
			defer uploadFile.Close()

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

			// Load existing catalog.
			catalogPath := shelf.EffectiveCatalogPath()
			data, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
			if err != nil && err.Error() != "not found" {
				return fmt.Errorf("reading catalog: %w", err)
			}
			books, _ := catalog.Parse(data)
			books = catalog.Append(books, book)

			newCatalog, err := catalog.Marshal(books)
			if err != nil {
				return err
			}

			if noPush {
				// Write locally only.
				if err := os.WriteFile(catalogPath, newCatalog, 0644); err != nil {
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

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Target shelf name (required)")
	cmd.Flags().StringVar(&releaseTag, "release", "", "Target release tag (default: shelf's default_release)")
	cmd.Flags().StringVar(&bookID, "id", "", "Book ID (default: prompt / slugified title)")
	cmd.Flags().BoolVar(&useSHA12, "id-sha12", false, "Use first 12 chars of sha256 as ID")
	cmd.Flags().StringVar(&title, "title", "", "Book title")
	cmd.Flags().StringVar(&author, "author", "", "Author")
	cmd.Flags().IntVar(&year, "year", 0, "Publication year")
	cmd.Flags().StringVar(&tagsCSV, "tags", "", "Comma-separated tags")
	cmd.Flags().StringVar(&assetName, "asset-name", "", "Override asset filename")
	cmd.Flags().BoolVar(&noPush, "no-push", false, "Update catalog locally only (do not push)")

	_ = cmd.MarkFlagRequired("shelf")
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
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if b.Len() > 0 {
			b.WriteRune('-')
		}
	}
	result := strings.Trim(b.String(), "-")
	if len(result) > 63 {
		result = result[:63]
	}
	if result == "" {
		return "book"
	}
	return result
}

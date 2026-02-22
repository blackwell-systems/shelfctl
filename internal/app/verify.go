package app

import (
	"fmt"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newVerifyCmd() *cobra.Command {
	var (
		shelfName string
		fix       bool
	)

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Detect catalog vs release mismatches",
		Long: `Check for orphaned catalog entries (in catalog but asset missing)
and orphaned assets (in release but not in catalog).
Use --fix to automatically clean up issues.

Examples:
  shelfctl verify              # Check all shelves
  shelfctl verify --shelf books
  shelfctl verify --fix        # Auto-fix issues`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var shelves []config.ShelfConfig
			if shelfName != "" {
				shelf := cfg.ShelfByName(shelfName)
				if shelf == nil {
					return fmt.Errorf("shelf %q not found in config", shelfName)
				}
				shelves = []config.ShelfConfig{*shelf}
			} else {
				shelves = cfg.Shelves
			}

			if len(shelves) == 0 {
				warn("No shelves configured")
				return nil
			}

			totalIssues := 0
			for i := range shelves {
				shelf := &shelves[i]
				issues := verifySingleShelf(shelf, fix)
				totalIssues += len(issues)
			}

			fmt.Println()
			if totalIssues == 0 {
				ok("No issues found")
			} else if !fix {
				warn("%d issues found. Run with --fix to repair.", totalIssues)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Verify specific shelf only")
	cmd.Flags().BoolVar(&fix, "fix", false, "Automatically fix issues")
	return cmd
}

type verifyIssue struct {
	Type        string // "orphaned_catalog" or "orphaned_asset"
	BookID      string
	AssetName   string
	Description string
}

func verifySingleShelf(shelf *config.ShelfConfig, fix bool) []verifyIssue {
	owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
	catalogPath := shelf.EffectiveCatalogPath()
	releaseTag := shelf.EffectiveRelease(cfg.Defaults.Release)

	fmt.Println()
	header("Verifying shelf: %s", shelf.Name)
	fmt.Printf("  Owner/Repo: %s/%s\n", owner, shelf.Repo)
	fmt.Printf("  Release: %s\n", releaseTag)

	// 1. Load catalog
	catalogData, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
	if err != nil {
		warn("Could not load catalog for %s: %v", shelf.Name, err)
		return nil
	}
	books, err := catalog.Parse(catalogData)
	if err != nil {
		warn("Could not parse catalog for %s: %v", shelf.Name, err)
		return nil
	}

	// 2. Get release and all assets
	rel, err := gh.GetReleaseByTag(owner, shelf.Repo, releaseTag)
	if err != nil {
		warn("Could not get release for %s: %v", shelf.Name, err)
		return nil
	}

	assets, err := gh.ListReleaseAssets(owner, shelf.Repo, rel.ID)
	if err != nil {
		warn("Could not list assets for %s: %v", shelf.Name, err)
		return nil
	}

	fmt.Printf("  Catalog: %d entries\n", len(books))
	fmt.Printf("  Release: %d assets\n", len(assets))

	// 3. Build maps for comparison
	assetNames := make(map[string]*github.Asset)
	for i := range assets {
		assetNames[assets[i].Name] = &assets[i]
	}

	catalogAssets := make(map[string]*catalog.Book)
	for i := range books {
		catalogAssets[books[i].Source.Asset] = &books[i]
	}

	var issues []verifyIssue
	var catalogModified bool

	// 4. Find orphaned catalog entries (in catalog but asset missing)
	for i := range books {
		b := &books[i]
		if _, exists := assetNames[b.Source.Asset]; !exists {
			issues = append(issues, verifyIssue{
				Type:        "orphaned_catalog",
				BookID:      b.ID,
				AssetName:   b.Source.Asset,
				Description: "In catalog but asset missing from release",
			})

			if fix {
				// Remove from catalog
				books, _ = catalog.Remove(books, b.ID)
				catalogModified = true
				ok("Removed %s from catalog", b.ID)

				// Clear from cache if exists
				if cacheMgr.Exists(owner, shelf.Repo, b.ID, b.Source.Asset) {
					if err := cacheMgr.Remove(owner, shelf.Repo, b.ID, b.Source.Asset); err != nil {
						warn("Could not clear cache for %s: %v", b.ID, err)
					}
				}
			}
		}
	}

	// 5. Find orphaned assets (in release but not in catalog)
	for i := range assets {
		asset := &assets[i]
		if _, exists := catalogAssets[asset.Name]; !exists {
			issues = append(issues, verifyIssue{
				Type:        "orphaned_asset",
				AssetName:   asset.Name,
				Description: "In release but not referenced in catalog",
			})

			if fix {
				// Delete asset from release
				if err := gh.DeleteAsset(owner, shelf.Repo, asset.ID); err != nil {
					warn("Could not delete asset %s: %v", asset.Name, err)
				} else {
					ok("Deleted %s from release", asset.Name)
				}
			}
		}
	}

	// 6. Commit catalog if modified
	if fix && catalogModified {
		updatedData, err := catalog.Marshal(books)
		if err != nil {
			warn("Could not marshal catalog: %v", err)
		} else {
			commitMsg := fmt.Sprintf("verify: clean up %d orphaned entries", len(issues))
			if err := gh.CommitFile(owner, shelf.Repo, catalogPath, updatedData, commitMsg); err != nil {
				warn("Could not commit catalog: %v", err)
			} else {
				ok("Catalog committed")

				// Update README with new count
				readmeData, _, readmeErr := gh.GetFileContent(owner, shelf.Repo, "README.md", "")
				if readmeErr == nil {
					originalContent := string(readmeData)
					readmeContent := updateShelfREADMEStats(originalContent, len(books))

					if readmeContent != originalContent {
						readmeMsg := "Update README: verify cleanup"
						if err := gh.CommitFile(owner, shelf.Repo, "README.md", []byte(readmeContent), readmeMsg); err != nil {
							warn("Could not update README.md: %v", err)
						}
					}
				}
			}
		}
	}

	// 7. Display issues
	if len(issues) > 0 && !fix {
		fmt.Println()
		fmt.Println(color.YellowString("Issues found:"))
		for _, issue := range issues {
			if issue.Type == "orphaned_catalog" {
				fmt.Printf("  %s Orphaned catalog entry: %s\n", color.RedString("✗"), color.WhiteString(issue.BookID))
				fmt.Printf("    - Asset %q missing from release\n", issue.AssetName)
				fmt.Printf("    - Fix: Remove from catalog\n")
			} else {
				fmt.Printf("  %s Orphaned release asset: %s\n", color.RedString("✗"), color.WhiteString(issue.AssetName))
				fmt.Printf("    - In release but not referenced in catalog\n")
				fmt.Printf("    - Fix: Delete from release\n")
			}
			fmt.Println()
		}
	} else if len(issues) == 0 {
		fmt.Printf("  %s All in sync\n", color.GreenString("✓"))
	}

	return issues
}

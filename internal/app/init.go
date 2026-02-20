package app

import (
	"fmt"
	"os"

	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var (
		owner         string
		repoName      string
		shelfName     string
		createRepo    bool
		private       bool
		createRelease bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Bootstrap a shelf repo and add it to your config",
		Long: `Bootstrap a shelf repository for storing books.

shelfctl organizes your books into "shelves" - GitHub repos where:
  • Book metadata lives in catalog.yml (tracked in Git)
  • Book files live in GitHub Release assets (not tracked in Git)
  • You can have multiple shelves for different topics

This command creates or registers a shelf repo in your config.

Quick start:
  1. Run: shelfctl init --repo shelf-books --name books --create-repo --create-release
  2. Then: shelfctl shelve (launches interactive workflow)
  3. Or: shelfctl shelve ~/book.pdf --shelf books --title "My Book"

For more details, see: shelfctl --help or docs/TUTORIAL.md`,
		Example: `  # Create a new shelf with repo and release
  shelfctl init --repo shelf-programming --name programming --create-repo --create-release

  # Register an existing repo as a shelf
  shelfctl init --repo shelf-history --name history`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve owner.
			effectiveOwner := owner
			if effectiveOwner == "" && cfg != nil {
				effectiveOwner = cfg.GitHub.Owner
			}
			if effectiveOwner == "" {
				return fmt.Errorf("--owner is required (or set github.owner in config)")
			}

			if repoName == "" {
				return fmt.Errorf("--repo is required (run 'shelfctl init --help' for examples)")
			}
			if shelfName == "" {
				// Default shelf name from repo: shelf-<name> → <name>
				shelfName = repoName
				if len(repoName) > 6 && repoName[:6] == "shelf-" {
					shelfName = repoName[6:]
				}
			}

			if createRepo {
				header("Creating repo %s/%s …", effectiveOwner, repoName)
				repo, err := gh.CreateRepo(repoName, private)
				if err != nil {
					return fmt.Errorf("create repo: %w", err)
				}
				ok("Created %s", repo.HTMLURL)
			}

			if createRelease {
				header("Creating release 'library' in %s/%s …", effectiveOwner, repoName)
				rel, err := gh.EnsureRelease(effectiveOwner, repoName, "library")
				if err != nil {
					return fmt.Errorf("create release: %w", err)
				}
				ok("Release ready: %s", rel.HTMLURL)
			}

			// Create README.md for the shelf
			if createRepo {
				header("Creating README.md …")
				readmeContent := generateShelfREADME(shelfName, repoName, effectiveOwner)
				readmeBytes := []byte(readmeContent)

				commitMsg := "Initial commit: Add shelf README"
				if err := gh.CommitFile(effectiveOwner, repoName, "README.md", readmeBytes, commitMsg); err != nil {
					warn("Could not create README.md: %v", err)
				} else {
					ok("README.md created")
				}
			}

			// Add shelf to config file.
			currentCfg, err := config.Load()
			if err != nil {
				currentCfg = &config.Config{}
			}

			// Don't duplicate.
			for _, s := range currentCfg.Shelves {
				if s.Name == shelfName {
					warn("Shelf %q already in config — skipping", shelfName)
					return nil
				}
			}

			currentCfg.Shelves = append(currentCfg.Shelves, config.ShelfConfig{
				Name:  shelfName,
				Owner: effectiveOwner,
				Repo:  repoName,
			})

			if currentCfg.GitHub.Owner == "" {
				currentCfg.GitHub.Owner = effectiveOwner
			}
			if currentCfg.GitHub.TokenEnv == "" {
				currentCfg.GitHub.TokenEnv = "GITHUB_TOKEN"
			}
			if currentCfg.GitHub.APIBase == "" {
				currentCfg.GitHub.APIBase = "https://api.github.com"
			}
			if currentCfg.Defaults.Release == "" {
				currentCfg.Defaults.Release = "library"
			}
			if currentCfg.Defaults.AssetNaming == "" {
				currentCfg.Defaults.AssetNaming = "id"
			}

			if err := config.Save(currentCfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			configPath := config.DefaultPath()
			ok("Added shelf %q to config", shelfName)
			fmt.Printf("  config: %s\n", color.CyanString(configPath))
			fmt.Printf("  owner:  %s\n", effectiveOwner)
			fmt.Printf("  repo:   %s\n", repoName)

			if !createRepo {
				hint := fmt.Sprintf("Make sure %s/%s exists on GitHub.", effectiveOwner, repoName)
				fmt.Fprintln(os.Stderr, color.YellowString("hint:"), hint)
			}

			// Show next steps
			fmt.Println()
			fmt.Println(color.GreenString("✓ Shelf initialized!"))
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Printf("  1. Add your first book (interactive):\n")
			fmt.Printf("     %s\n\n", color.CyanString("shelfctl shelve"))
			fmt.Printf("  2. Or with all details:\n")
			fmt.Printf("     %s\n\n", color.CyanString(fmt.Sprintf("shelfctl shelve ~/book.pdf --shelf %s --title \"My Book\"", shelfName)))
			fmt.Printf("  3. Browse your library:\n")
			fmt.Printf("     %s\n\n", color.CyanString("shelfctl browse"))
			fmt.Printf("  4. View your shelves:\n")
			fmt.Printf("     %s\n", color.CyanString("shelfctl shelves"))

			return nil
		},
	}

	cmd.Flags().StringVar(&owner, "owner", "", "GitHub owner (defaults to github.owner in config)")
	cmd.Flags().StringVar(&repoName, "repo", "", "GitHub repo name (e.g. shelf-programming)")
	cmd.Flags().StringVar(&shelfName, "name", "", "Local shelf name (default: repo without 'shelf-' prefix)")
	cmd.Flags().BoolVar(&createRepo, "create-repo", false, "Create the GitHub repo via API")
	cmd.Flags().BoolVar(&private, "private", true, "Make the repo private (with --create-repo)")
	cmd.Flags().BoolVar(&createRelease, "create-release", true, "Create the 'library' release")

	return cmd
}

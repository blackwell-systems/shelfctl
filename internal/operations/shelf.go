package operations

import (
	"fmt"

	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
)

// CreateShelf performs shelf creation without terminal output
// This function is shared by both CLI (init.go) and TUI (create_shelf.go)
func CreateShelf(gh *github.Client, cfg *config.Config, shelfName, repoName string, createRepo, private bool) error {
	// Validate and resolve parameters
	effectiveOwner, effectiveShelfName, err := validateParams(cfg, "", repoName, shelfName)
	if err != nil {
		return err
	}

	// Create repo and release if requested
	if createRepo {
		if err := createRepoAndRelease(gh, effectiveOwner, repoName, private); err != nil {
			return err
		}

		// Create README
		createShelfREADME(gh, effectiveShelfName, repoName, effectiveOwner)
	}

	// Update config file
	if err := addShelfToConfig(cfg, effectiveShelfName, effectiveOwner, repoName); err != nil {
		return err
	}

	return nil
}

func validateParams(cfg *config.Config, owner, repoName, shelfName string) (string, string, error) {
	// Resolve owner
	effectiveOwner := owner
	if effectiveOwner == "" && cfg != nil {
		effectiveOwner = cfg.GitHub.Owner
	}
	if effectiveOwner == "" {
		return "", "", fmt.Errorf("owner is required (set github.owner in config)")
	}

	if repoName == "" {
		return "", "", fmt.Errorf("repo name is required")
	}

	effectiveShelfName := shelfName
	if effectiveShelfName == "" {
		// Default shelf name from repo: shelf-<name> → <name>
		effectiveShelfName = repoName
		if len(repoName) > 6 && repoName[:6] == "shelf-" {
			effectiveShelfName = repoName[6:]
		}
	}

	return effectiveOwner, effectiveShelfName, nil
}

func createRepoAndRelease(gh *github.Client, owner, repoName string, private bool) error {
	// Check if repo already exists first
	existingRepo, err := gh.GetRepo(owner, repoName)
	if err == nil && existingRepo != nil {
		// Repo already exists - use it
		return nil
	}

	// Repo doesn't exist, create it
	_, err = gh.CreateRepo(repoName, private)
	if err != nil {
		return fmt.Errorf("create repo: %w", err)
	}

	// Create release
	_, err = gh.EnsureRelease(owner, repoName, "library")
	if err != nil {
		return fmt.Errorf("create release: %w", err)
	}

	return nil
}

func createShelfREADME(gh *github.Client, shelfName, repoName, owner string) {
	// Check if README.md already exists
	_, _, err := gh.GetFileContent(owner, repoName, "README.md", "")
	if err == nil {
		return
	}

	readmeContent := generateShelfREADME(shelfName, repoName, owner)
	readmeBytes := []byte(readmeContent)

	commitMsg := "Initial commit: Add shelf README"
	_ = gh.CommitFile(owner, repoName, "README.md", readmeBytes, commitMsg)
}

func generateShelfREADME(shelfName, repoName, owner string) string {
	return fmt.Sprintf(`# %s

A shelf managed by [shelfctl](https://github.com/blackwell-systems/shelfctl).

## About

This repository stores book metadata for the "%s" shelf.

- **Catalog:** Book metadata tracked in Git (catalog.yml)
- **Files:** Book files stored as GitHub Release assets
- **Owner:** %s
- **Repository:** %s

## Usage

Add books to this shelf:
`, shelfName, shelfName, owner, repoName) +
		"```bash\n" +
		fmt.Sprintf("shelfctl shelve book.pdf --shelf %s --title \"Book Title\"\n", shelfName) +
		"```\n\n" +
		"Browse your library:\n" +
		"```bash\n" +
		"shelfctl browse\n" +
		"```\n"
}

func addShelfToConfig(cfg *config.Config, shelfName, owner, repoName string) error {
	currentCfg, err := config.Load()
	if err != nil {
		currentCfg = &config.Config{}
	}

	// Don't duplicate
	for _, s := range currentCfg.Shelves {
		if s.Name == shelfName {
			return nil
		}
	}

	currentCfg.Shelves = append(currentCfg.Shelves, config.ShelfConfig{
		Name:  shelfName,
		Owner: owner,
		Repo:  repoName,
	})

	setConfigDefaults(currentCfg, owner)

	return config.Save(currentCfg)
}

func setConfigDefaults(cfg *config.Config, owner string) {
	if cfg.GitHub.Owner == "" {
		cfg.GitHub.Owner = owner
	}
	if cfg.GitHub.TokenEnv == "" {
		cfg.GitHub.TokenEnv = "GITHUB_TOKEN"
	}
	if cfg.GitHub.APIBase == "" {
		cfg.GitHub.APIBase = "https://api.github.com"
	}
	if cfg.Defaults.Release == "" {
		cfg.Defaults.Release = "library"
	}
	if cfg.Defaults.AssetNaming == "" {
		cfg.Defaults.AssetNaming = "id"
	}
}

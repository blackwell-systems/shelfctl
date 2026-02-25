package app

import (
	"fmt"

	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/operations"
	"github.com/blackwell-systems/shelfctl/internal/util"
	"github.com/fatih/color"
)

// resolveOrCreateShelf looks up a shelf by name. If not found:
//   - With --create-shelf flag: creates the shelf automatically (repo + release + config)
//   - With TTY and no flag: prompts the user to create it
//   - Non-TTY without flag: returns an error with a hint
func resolveOrCreateShelf(name string) (*config.ShelfConfig, error) {
	if shelf := cfg.ShelfByName(name); shelf != nil {
		return shelf, nil
	}

	if flagCreateShelf {
		return createAndReloadShelf(name)
	}

	if util.IsTTY() {
		fmt.Printf("Shelf %q not found. Create it? (Y/n): ", name)
		var response string
		_, _ = fmt.Scanln(&response)
		if response == "" || response == "y" || response == "Y" || response == "yes" {
			return createAndReloadShelf(name)
		}
		return nil, fmt.Errorf("shelf %q not found in config", name)
	}

	return nil, fmt.Errorf("shelf %q not found in config (use --create-shelf to auto-create)", name)
}

func createAndReloadShelf(name string) (*config.ShelfConfig, error) {
	repoName := "shelf-" + name
	fmt.Printf("Creating shelf %q (repo: %s, private)...\n", name, repoName)
	if err := operations.CreateShelf(gh, cfg, name, repoName, true, true); err != nil {
		return nil, fmt.Errorf("creating shelf: %w", err)
	}
	color.Green("✓ Shelf %q created", name)

	// Reload config since CreateShelf writes directly to the config file
	newCfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("reloading config after shelf creation: %w", err)
	}
	cfg = newCfg

	shelf := cfg.ShelfByName(name)
	if shelf == nil {
		return nil, fmt.Errorf("shelf %q not found after creation", name)
	}
	return shelf, nil
}

package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/util"
)

// configFileExists returns true if a config file exists at the
// effective config path (SHELFCTL_CONFIG env or default path).
func configFileExists() bool {
	path := os.Getenv("SHELFCTL_CONFIG")
	if path == "" {
		path = config.DefaultPath()
	}
	_, err := os.Stat(path)
	return err == nil
}

// PromptAndCreateConfig checks if config exists at the default path.
// If missing AND stdin is a TTY, prompts the user for GitHub owner and
// token env var name, writes a minimal config.yml, and returns the
// loaded config. If stdin is not a TTY, returns (nil, nil) so the
// caller can fall through to existing non-interactive error handling.
func PromptAndCreateConfig() (*config.Config, error) {
	if !util.IsTTY() {
		return nil, nil
	}
	return promptAndCreateConfigFromReader(os.Stdin)
}

// promptAndCreateConfigFromReader is the testable core of PromptAndCreateConfig.
// It reads interactive input from r instead of os.Stdin.
func promptAndCreateConfigFromReader(r io.Reader) (*config.Config, error) {
	fmt.Print("No config found. Let's set up shelfctl.\n")

	reader := bufio.NewReader(r)

	// Prompt for GitHub owner (required, loop until non-empty)
	var owner string
	for {
		fmt.Print("GitHub username or org: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("reading input: %w", err)
		}
		owner = strings.TrimSpace(line)
		if owner != "" {
			break
		}
	}

	// Prompt for token env var name (default: SHELFCTL_GITHUB_TOKEN)
	fmt.Print("Token env var name [SHELFCTL_GITHUB_TOKEN]: ")
	tokenLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	tokenEnv := strings.TrimSpace(tokenLine)
	if tokenEnv == "" {
		tokenEnv = "SHELFCTL_GITHUB_TOKEN"
	}

	// Build minimal config
	newCfg := &config.Config{
		GitHub: config.GitHubConfig{
			Owner:    owner,
			TokenEnv: tokenEnv,
			APIBase:  "https://api.github.com",
		},
		Defaults: config.DefaultsConfig{
			Release:     "library",
			AssetNaming: "id",
		},
		Shelves: []config.ShelfConfig{},
	}

	// Write config to disk
	if err := config.Save(newCfg); err != nil {
		return nil, fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("Config created at %s\n", config.DefaultPath())

	// Reload via config.Load() to get resolved values (token from env, etc.)
	return config.Load()
}

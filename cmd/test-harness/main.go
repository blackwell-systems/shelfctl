package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/blackwell-systems/shelfctl/test/fixtures"
	"github.com/blackwell-systems/shelfctl/test/mockserver"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	tempDir    string
	configPath string
	server     *mockserver.MockServer
)

type Config struct {
	GitHub struct {
		Owner    string `yaml:"owner"`
		TokenEnv string `yaml:"token_env"`
		APIBase  string `yaml:"api_base"`
	} `yaml:"github"`
	Defaults struct {
		Release  string `yaml:"release"`
		CacheDir string `yaml:"cache_dir"`
	} `yaml:"defaults"`
	Shelves []Shelf `yaml:"shelves"`
}

type Shelf struct {
	Name        string `yaml:"name"`
	Repo        string `yaml:"repo"`
	CatalogPath string `yaml:"catalog_path"`
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "test-harness",
		Short: "shelfctl test harness coordinator",
		Long:  "Manages mock server and test environment for shelfctl integration testing",
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start test harness",
		Long:  "Launches mock GitHub API server, generates config.yml, and prints test instructions",
		RunE:  runStart,
	}

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop test harness",
		Long:  "Cleanup temp files and shutdown mock server",
		RunE:  runStop,
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check harness status",
		Long:  "Check if test harness is running",
		RunE:  runStatus,
	}

	rootCmd.AddCommand(startCmd, stopCmd, statusCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runStart(cmd *cobra.Command, args []string) error {
	// Create temp directory
	var err error
	tempDir, err = os.MkdirTemp("", "shelfctl-test-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	fmt.Printf("Created temp directory: %s\n", tempDir)

	// Initialize fixtures (not directly used in config, but available for mock server)
	fixtures := fixtures.DefaultFixtures()
	fmt.Printf("Loaded %d fixture shelves\n", len(fixtures.Shelves))

	// Create and start mock server
	server, err = mockserver.NewServer()
	if err != nil {
		return fmt.Errorf("failed to create mock server: %w", err)
	}

	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start mock server: %w", err)
	}

	serverURL := server.URL()
	fmt.Printf("Mock server started at: %s\n", serverURL)

	// Generate config.yml
	config := Config{}
	config.GitHub.Owner = "testuser"
	config.GitHub.TokenEnv = "GITHUB_TOKEN"
	config.GitHub.APIBase = serverURL
	config.Defaults.Release = "library"
	config.Defaults.CacheDir = filepath.Join(tempDir, "cache")

	config.Shelves = []Shelf{
		{
			Name:        "tech",
			Repo:        "shelf-tech",
			CatalogPath: "catalog.yml",
		},
	}

	configPath = filepath.Join(tempDir, "config.yml")
	configData, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Generated config at: %s\n\n", configPath)

	// Print usage instructions
	printInstructions(serverURL, configPath, tempDir)

	// Set up signal handling for clean shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\nPress Ctrl+C to stop the test harness")

	<-sigChan
	fmt.Println("\nShutting down...")
	return cleanup()
}

func runStop(cmd *cobra.Command, args []string) error {
	fmt.Println("Stopping test harness...")
	return cleanup()
}

func runStatus(cmd *cobra.Command, args []string) error {
	if server != nil && server.URL() != "" {
		fmt.Printf("Test harness is running\n")
		fmt.Printf("Server URL: %s\n", server.URL())
		fmt.Printf("Config path: %s\n", configPath)
		fmt.Printf("Temp directory: %s\n", tempDir)
		return nil
	}

	fmt.Println("Test harness is not running")
	return nil
}

func cleanup() error {
	var errs []error

	if server != nil {
		if err := server.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop server: %w", err))
		} else {
			fmt.Println("Mock server stopped")
		}
	}

	if tempDir != "" {
		if err := os.RemoveAll(tempDir); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove temp directory: %w", err))
		} else {
			fmt.Printf("Removed temp directory: %s\n", tempDir)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}

	return nil
}

func printInstructions(serverURL, configPath, tempDir string) {
	fmt.Println("================================================================================")
	fmt.Println("Test Harness Ready!")
	fmt.Println("================================================================================")
	fmt.Println()
	fmt.Println("Mock GitHub API Server:")
	fmt.Printf("  URL: %s\n", serverURL)
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Printf("  Config file: %s\n", configPath)
	fmt.Printf("  Temp directory: %s\n", tempDir)
	fmt.Println()
	fmt.Println("Example Commands:")
	fmt.Println()
	fmt.Printf("  # Use with shelfctl (set config path)\n")
	fmt.Printf("  export SHELFCTL_CONFIG=%s\n", configPath)
	fmt.Println("  shelfctl list")
	fmt.Println()
	fmt.Println("  # Or pass config directly")
	fmt.Printf("  shelfctl --config %s list\n", configPath)
	fmt.Println()
	fmt.Println("  # Test catalog operations")
	fmt.Printf("  shelfctl --config %s catalog show tech\n", configPath)
	fmt.Println()
	fmt.Println("  # Set mock GitHub token (can be any value)")
	fmt.Println("  export GITHUB_TOKEN=mock-token-12345")
	fmt.Println()
	fmt.Println("================================================================================")
}

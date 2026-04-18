package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/cache"
	ghclient "github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/blackwell-systems/shelfctl/internal/unified"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
)

// runUnifiedTUI launches the unified TUI with seamless view switching
func runUnifiedTUI() error {
	// Auto-create config if missing
	if !configFileExists() {
		newCfg, err := PromptAndCreateConfig()
		if err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
		if newCfg != nil {
			cfg = newCfg
			// Initialize GitHub client if token is available
			if cfg.GitHub.Token != "" {
				gh = ghclient.New(cfg.GitHub.Token, cfg.GitHub.APIBase)
				cacheMgr = cache.New(cfg.Defaults.CacheDir)
			}
			fmt.Println()
		}
	}

	// Check configuration status
	hasToken := cfg != nil && cfg.GitHub.Token != ""
	hasShelves := cfg != nil && len(cfg.Shelves) > 0

	// If not fully configured, show welcome/setup message
	if !hasToken || !hasShelves {
		if !hasToken {
			fmt.Println(color.YellowString("GitHub token not configured."))
			fmt.Println("Set GITHUB_TOKEN or add token to your config file.")
			fmt.Println("Run 'shelfctl init' to create a config.")
			return nil
		}
		// hasShelves == false at this point
		fmt.Println(color.YellowString("No shelves configured."))
		fmt.Println()
		fmt.Println("Run this to create your first shelf:")
		fmt.Printf("  %s\n", color.CyanString("shelfctl init --repo shelf-books --name books --create-repo --create-release"))
		return nil
	}

	// Build hub context
	ctx := buildHubContext()

	// If all shelves were deleted, exit gracefully
	if ctx.ShelfCount == 0 {
		fmt.Println()
		fmt.Println(color.YellowString("No shelves configured."))
		fmt.Println()
		fmt.Println("To use shelfctl, you need at least one shelf.")
		fmt.Println()
		fmt.Println("Run this to create your first shelf:")
		fmt.Printf("  %s\n", color.CyanString("shelfctl init --repo shelf-books --name books --create-repo --create-release"))
		fmt.Println()
		fmt.Println("Or run 'shelfctl' to use the interactive setup wizard.")
		return nil
	}

	// Run unified TUI in a loop to handle actions that need to exit/restart
	startView := unified.ViewHub
	for {
		// Create and run unified model
		m := unified.NewAtView(ctx, gh, cfg, cacheMgr, startView)
		p := tea.NewProgram(m, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		// Check if there's a pending action
		if unifiedModel, ok := finalModel.(unified.Model); ok {
			// Handle book actions (open, edit)
			if action := unifiedModel.GetPendingAction(); action != nil {
				// Perform the action (TUI has exited, we're back in normal terminal)
				if err := unified.PerformPendingAction(action, gh, cfg, cacheMgr); err != nil {
					// Suppress cancellation errors (user canceled is not a failure)
					errMsg := err.Error()
					if errMsg != "canceled" && errMsg != "canceled by user" && errMsg != "cancelled by user" {
						warn("Action failed: %v", err)
					}
				}

				// Check if we should restart
				if unifiedModel.ShouldRestart() {
					// Rebuild context and restart at the specified view
					ctx = buildHubContext()
					startView = unifiedModel.GetRestartView()
					continue
				}
			}

			// Handle command request (non-TUI commands)
			if cmdReq := unifiedModel.GetPendingCommand(); cmdReq != nil {

				// Run the command (TUI has exited, we're back in normal terminal)
				var cmdErr error
				switch cmdReq.Command {
				case "shelves":
					cmd := newShelvesCmd()
					cmd.SetArgs([]string{"--table"})
					cmdErr = cmd.Execute()

				case "index":
					cmdErr = newIndexCmd().Execute()

				case "cache-info":
					cmd := newCacheCmd()
					cmd.SetArgs([]string{"info"})
					cmdErr = cmd.Execute()

				case "shelve-url":
					cmdErr = runShelveFromURL()

				case "import-repo":
					cmdErr = runImportFromRepo()

				case "delete-shelf":
					cmdErr = newDeleteShelfCmd().Execute()

				default:
					warn("Unknown command: %s", cmdReq.Command)
				}

				// Show result (suppress cancellations)
				wasCanceled := false
				if cmdErr != nil {
					errMsg := cmdErr.Error()
					if errMsg == "canceled" || errMsg == "canceled by user" || errMsg == "cancelled by user" {
						wasCanceled = true
					} else {
						warn("Command failed: %v", cmdErr)
					}
				}

				// Wait for user to press Enter (skip if canceled)
				if !wasCanceled {
					fmt.Println("\nPress Enter to return to menu...")
					fmt.Scanln() //nolint:errcheck
				}

				// Check if we should restart
				if unifiedModel.ShouldRestart() {
					// Rebuild context and restart at the specified view
					ctx = buildHubContext()
					startView = unifiedModel.GetRestartView()
					continue
				}
			}
		}

		// No pending action or no restart needed, exit
		break
	}

	return nil
}

func runShelveFromURL() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	header("Add Book from URL")
	fmt.Println()
	fmt.Print("Enter URL: ")
	url, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Call shelve command with URL
	cmd := newShelveCmd()
	cmd.SetArgs([]string{url})
	return cmd.Execute()
}

func runImportFromRepo() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	header("Import Books from Repository")
	fmt.Println()
	fmt.Println("This will scan a repository for PDFs and migrate them to your shelves.")
	fmt.Println()
	fmt.Print("Enter source repository (owner/repo): ")
	source, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	source = strings.TrimSpace(source)
	if source == "" {
		return fmt.Errorf("repository cannot be empty")
	}

	// Validate format
	if !strings.Contains(source, "/") {
		return fmt.Errorf("repository must be in format: owner/repo")
	}

	fmt.Println()
	fmt.Println(color.CyanString("Scanning repository for PDFs..."))

	// Run migrate scan to create queue file
	queueFile := fmt.Sprintf("/tmp/shelfctl-import-%d.txt", os.Getpid())
	scanCmd := newMigrateScanCmd()
	scanCmd.SetArgs([]string{"--source", source, "--out", queueFile})
	if err := scanCmd.Execute(); err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Check if queue file has content
	data, err := os.ReadFile(queueFile)
	if err != nil {
		return fmt.Errorf("reading queue file: %w", err)
	}
	if len(data) == 0 {
		fmt.Println()
		fmt.Println(color.YellowString("No PDFs found in repository."))
		_ = os.Remove(queueFile)
		return nil
	}

	fileCount := strings.Count(string(data), "\n")
	fmt.Println()
	fmt.Printf("Found %s files to migrate.\n", color.WhiteString("%d", fileCount))
	fmt.Println()
	fmt.Print("Proceed with migration? (Y/n): ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	if confirm != "" && confirm != "y" && confirm != "yes" {
		_ = os.Remove(queueFile)
		return fmt.Errorf("canceled by user")
	}

	// Run migrate batch
	fmt.Println()
	fmt.Println(color.CyanString("Starting migration..."))
	batchCmd := newMigrateBatchCmd()
	batchCmd.SetArgs([]string{queueFile, "--continue"})
	err = batchCmd.Execute()

	// Clean up queue file
	_ = os.Remove(queueFile)

	return err
}


func buildHubContext() tui.HubContext {
	ctx := unified.BuildContextFast(cfg)
	ctx.Version = appVersion
	return ctx
}

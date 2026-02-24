package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/blackwell-systems/shelfctl/internal/unified"
	"github.com/blackwell-systems/shelfctl/internal/util"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
)

// runUnifiedTUI launches the unified TUI with seamless view switching
func runUnifiedTUI() error {
	// Check configuration status
	hasToken := cfg != nil && cfg.GitHub.Token != ""
	hasShelves := cfg != nil && len(cfg.Shelves) > 0

	// If not fully configured, show welcome/setup message
	if !hasToken || !hasShelves {
		// Same setup flow as runHub()
		// For now, fall back to legacy runHub() for setup
		return runHub()
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

// runHub launches the interactive hub menu and routes to selected action
// DEPRECATED: This is the legacy implementation. Use runUnifiedTUI() for new code.
func runHub() error {
	// Check configuration status
	hasToken := cfg != nil && cfg.GitHub.Token != ""
	hasShelves := cfg != nil && len(cfg.Shelves) > 0

	// If not fully configured, show welcome/setup message
	if !hasToken || !hasShelves {
		fmt.Println(color.YellowString("⚠ Welcome to shelfctl!"))
		fmt.Println()
		fmt.Println("Setup status:")
		fmt.Println()

		// Token status
		if hasToken {
			fmt.Printf("  %s GitHub token configured\n", color.GreenString("✓"))
		} else {
			fmt.Printf("  %s GitHub token not found\n", color.RedString("✗"))
		}

		// Shelves status
		if hasShelves {
			fmt.Printf("  %s %d shelf(s) configured\n", color.GreenString("✓"), len(cfg.Shelves))
		} else {
			fmt.Printf("  %s No shelves configured\n", color.RedString("✗"))
		}

		fmt.Println()

		// Show specific next steps based on what's missing
		if !hasToken {
			fmt.Println("Next step: Set your GitHub token")
			fmt.Printf("  %s\n\n", color.CyanString("export GITHUB_TOKEN=ghp_your_token_here"))
			fmt.Println("Then run 'shelfctl' again.")
			fmt.Println()
			fmt.Println("For more details, see docs/guides/tutorial.md or run 'shelfctl init --help'")
			return nil
		} else if !hasShelves {
			fmt.Println("Next step: Create your first shelf")
			fmt.Println()

			// Offer to guide them through init
			if util.IsTTY() {
				fmt.Print("Would you like to create a shelf now? (y/n): ")
				var response string
				_, _ = fmt.Scanln(&response)
				if response == "y" || response == "Y" || response == "yes" {
					fmt.Println()
					if err := runInteractiveInit(); err != nil {
						return err
					}

					// Successfully created shelf - reload config and continue to hub
					var err error
					cfg, err = config.Load()
					if err != nil {
						return fmt.Errorf("reloading config: %w", err)
					}

					fmt.Println()
					fmt.Println(color.CyanString("Press Enter to open the menu..."))
					var dummy string
					_, _ = fmt.Scanln(&dummy)

					// Continue to hub loop below
				} else {
					fmt.Println()
					fmt.Println("Or run manually:")
					fmt.Printf("  %s\n\n", color.CyanString("shelfctl init --repo shelf-books --name books --create-repo --create-release"))
					fmt.Println("Then run 'shelfctl' again to use the interactive menu.")
					fmt.Println()
					fmt.Println("For more details, see docs/guides/tutorial.md or run 'shelfctl init --help'")
					return nil
				}
			} else {
				fmt.Println()
				fmt.Println("Or run manually:")
				fmt.Printf("  %s\n\n", color.CyanString("shelfctl init --repo shelf-books --name books --create-repo --create-release"))
				fmt.Println("Then run 'shelfctl' again to use the interactive menu.")
				fmt.Println()
				fmt.Println("For more details, see docs/guides/tutorial.md or run 'shelfctl init --help'")
				return nil
			}
		}
	}

	// Hub loop - keep showing menu until user quits or command succeeds
	for {
		// Gather context for the hub (refresh each time)
		ctx := buildHubContext()

		// If all shelves were deleted, exit hub gracefully
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

		action, err := tui.RunHub(ctx)
		if err != nil {
			return err
		}

		// Determine if this action is a TUI command (no "Press Enter" needed)
		isTUIAction := action == "browse" || action == "shelve" || action == "edit-book" ||
			action == "move" || action == "delete-book" || action == "cache-clear"

		// Route to the appropriate command based on action
		var cmdErr error

		switch action {
		case "browse":
			cmdErr = newBrowseCmd().Execute()
		case "shelves":
			cmd := newShelvesCmd()
			cmd.SetArgs([]string{"--table"})
			cmdErr = cmd.Execute()
		case "index":
			cmdErr = newIndexCmd().Execute()
		case "shelve":
			cmdErr = newShelveCmd().Execute()
		case "shelve-url":
			cmdErr = runShelveFromURL()
		case "import-repo":
			cmdErr = runImportFromRepo()
		case "edit-book":
			cmdErr = newEditBookCmd().Execute()
		case "move":
			cmdErr = newMoveCmd().Execute()
		case "delete-book":
			cmdErr = newDeleteBookCmd().Execute()
		case "delete-shelf":
			cmdErr = newDeleteShelfCmd().Execute()
		case "cache-info":
			cmd := newCacheCmd()
			cmd.SetArgs([]string{"info"})
			cmdErr = cmd.Execute()
		case "cache-clear":
			cmd := newCacheCmd()
			cmd.SetArgs([]string{"clear"})
			cmdErr = cmd.Execute()
		case "quit":
			return nil
		default:
			return fmt.Errorf("unknown action: %s", action)
		}

		// Handle command result
		if cmdErr == nil {
			// Command succeeded - reload config to pick up any changes
			var err error
			cfg, err = config.Load()
			if err != nil {
				return fmt.Errorf("reloading config: %w", err)
			}

			// For TUI actions, return directly to hub without "Press Enter" prompt
			if isTUIAction {
				// Clear screen to reduce flicker between TUI transitions
				fmt.Print("\033[2J\033[H")
				continue
			}

			// For non-TUI actions, show success message and wait for Enter
			fmt.Println()
			fmt.Println(color.CyanString("Press Enter to return to menu..."))
			var dummy string
			_, _ = fmt.Scanln(&dummy)
			continue
		}

		// Command failed/canceled
		// For TUI actions, return directly to hub (user already quit from TUI)
		if isTUIAction {
			// Clear screen to reduce flicker between TUI transitions
			fmt.Print("\033[2J\033[H")
			continue
		}

		// For non-TUI actions, show error and wait for Enter
		fmt.Println()
		fmt.Println(color.RedString("Operation failed or canceled: %v", cmdErr))
		fmt.Println()
		fmt.Println(color.CyanString("Press Enter to return to menu..."))
		var dummy string
		_, _ = fmt.Scanln(&dummy)
	}
}

func buildHubContext() tui.HubContext {
	return unified.BuildContext(gh, cfg, cacheMgr)
}

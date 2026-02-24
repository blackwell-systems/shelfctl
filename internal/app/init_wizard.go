package app

import (
	"fmt"

	"github.com/fatih/color"
)

// showShelfArchitectureHelp displays information about shelf structure and organization
func showShelfArchitectureHelp() {
	fmt.Println()
	fmt.Println(color.CyanString("═══════════════════════════════════════════════════════════"))
	fmt.Println(color.CyanString("  How Shelves Work"))
	fmt.Println(color.CyanString("═══════════════════════════════════════════════════════════"))
	fmt.Println()

	fmt.Println(color.YellowString("Structure:"))
	fmt.Println("  Each shelf is a GitHub repository with:")
	fmt.Println("    • catalog.yml (in Git) - Metadata for your books")
	fmt.Println("    • Release assets (not in Git) - The actual PDF/EPUB files")
	fmt.Println()

	fmt.Println(color.YellowString("Organization Strategy:"))
	fmt.Println("  1. " + color.GreenString("Start broad") + " - One shelf is often enough at first")
	fmt.Println("     Example: shelf-books (general collection)")
	fmt.Println()
	fmt.Println("  2. " + color.GreenString("Use tags") + " - Organize books within a shelf using tags")
	fmt.Println("     Example: --tags programming,golang,textbook")
	fmt.Println()
	fmt.Println("  3. " + color.GreenString("Split later") + " - When a shelf grows large, split it")
	fmt.Println("     Use: shelfctl split (interactive wizard)")
	fmt.Println()

	fmt.Println(color.YellowString("When to Create Multiple Shelves:"))
	fmt.Println("  ✓ Different topics with distinct audiences")
	fmt.Println("    Example: shelf-work, shelf-personal, shelf-research")
	fmt.Println()
	fmt.Println("  ✓ When a shelf exceeds ~200-300 books")
	fmt.Println("    GitHub releases work best with moderate asset counts")
	fmt.Println()
	fmt.Println("  ✓ Different access requirements")
	fmt.Println("    Example: shelf-public (public repo), shelf-private (private repo)")
	fmt.Println()

	fmt.Println(color.YellowString("Two Names? Why?"))
	fmt.Println("  You'll provide two names:")
	fmt.Println()
	fmt.Println("  1. " + color.CyanString("Repository name") + " - The GitHub repo (e.g., shelf-programming)")
	fmt.Println("     • This is what appears on GitHub")
	fmt.Println("     • Use pattern: shelf-<topic>")
	fmt.Println()
	fmt.Println("  2. " + color.CyanString("Shelf name") + " - Short name for commands (e.g., programming)")
	fmt.Println("     • This is what you type in commands")
	fmt.Println("     • Usually just the topic without 'shelf-' prefix")
	fmt.Println()
	fmt.Println("  Example:")
	fmt.Println("    Repository: " + color.WhiteString("shelf-programming"))
	fmt.Println("    Shelf name: " + color.WhiteString("programming"))
	fmt.Println("    Command:    " + color.CyanString("shelfctl shelve --shelf programming"))
	fmt.Println()

	fmt.Println(color.YellowString("Advanced: Sub-organization with Releases"))
	fmt.Println("  You can use multiple releases within one shelf:")
	fmt.Println()
	fmt.Println("  shelf-programming/")
	fmt.Println("    release: library (default)")
	fmt.Println("    release: textbooks")
	fmt.Println("    release: papers")
	fmt.Println()
	fmt.Println("  Move books between releases with: shelfctl move --to-release")
	fmt.Println()

	fmt.Println(color.YellowString("Splitting Shelves:"))
	fmt.Println("  Don't worry about perfect organization now!")
	fmt.Println()
	fmt.Println("  Later, you can run: " + color.CyanString("shelfctl split"))
	fmt.Println()
	fmt.Println("  This launches a wizard that:")
	fmt.Println("    • Groups books by tags or other criteria")
	fmt.Println("    • Helps you decide which books go where")
	fmt.Println("    • Moves everything automatically")
	fmt.Println()

	fmt.Println(color.YellowString("Recommendation for First Shelf:"))
	fmt.Println("  Start with " + color.GreenString("shelf-books") + " or " + color.GreenString("shelf-library"))
	fmt.Println("  → Simple, general-purpose name")
	fmt.Println("  → Easy to split later by topic")
	fmt.Println("  → Use tags for organization in the meantime")
	fmt.Println()

	fmt.Println(color.CyanString("═══════════════════════════════════════════════════════════"))
	fmt.Println()
}

// runInteractiveInit guides the user through creating their first shelf
func runInteractiveInit() error {
	fmt.Println(color.CyanString("📚 Let's set up your first shelf!"))
	fmt.Println()
	fmt.Println(color.GreenString("Tip:") + " Type 'help' or '?' at any prompt for detailed guidance")
	fmt.Println()

	// Offer architecture info upfront
	for {
		fmt.Print("Want to learn about shelf architecture first? (y/n/?): ")
		var wantInfo string
		_, _ = fmt.Scanln(&wantInfo)

		if wantInfo == "help" || wantInfo == "?" {
			showShelfArchitectureHelp()
			fmt.Println()
			continue
		}

		if wantInfo == "y" || wantInfo == "Y" || wantInfo == "yes" {
			showShelfArchitectureHelp()
		}
		break
	}
	fmt.Println()

	// Get repo name with help option
	fmt.Println("This will create a GitHub repository to store your books.")
	fmt.Println()

	var repoName string
	for {
		fmt.Print("GitHub repository name (e.g., shelf-books) [?=help]: ")
		_, _ = fmt.Scanln(&repoName)

		if repoName == "help" || repoName == "?" {
			showShelfArchitectureHelp()
			fmt.Println()
			continue
		}

		if repoName == "" {
			repoName = "shelf-books"
			fmt.Printf("  Using default: %s\n", color.GreenString(repoName))
		}

		// Check if repo already exists
		existingRepo, err := gh.GetRepo(cfg.GitHub.Owner, repoName)
		if err == nil && existingRepo != nil {
			// Repo exists - give user options
			fmt.Println()
			fmt.Println(color.YellowString("⚠ Repository %s/%s already exists", cfg.GitHub.Owner, repoName))
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  1. Use existing repository (just add to config)")
			fmt.Println("  2. Enter a different repository name")
			fmt.Println()
			fmt.Print("Choose (1/2): ")
			var choice string
			_, _ = fmt.Scanln(&choice)

			if choice == "2" {
				fmt.Println()
				continue // Go back to repo name prompt
			}
			// Choice 1 or Enter: use existing repo
			fmt.Println()
			fmt.Println(color.GreenString("Will use existing repository: %s", existingRepo.HTMLURL))
		}
		break
	}

	// Calculate smart default for shelf name
	defaultShelfName := repoName
	if len(repoName) > 6 && repoName[:6] == "shelf-" {
		defaultShelfName = repoName[6:]
	}

	// Get shelf name with help option
	fmt.Println()
	fmt.Printf("The shelf name is a short nickname used in commands like:\n")
	fmt.Printf("  %s\n", color.CyanString(fmt.Sprintf("shelfctl shelve book.pdf --shelf %s", defaultShelfName)))
	fmt.Println()

	var shelfName string
	for {
		fmt.Printf("Shelf name for commands (default: %s) [?=help]: ", color.GreenString(defaultShelfName))
		_, _ = fmt.Scanln(&shelfName)

		if shelfName == "help" || shelfName == "?" {
			showShelfArchitectureHelp()
			fmt.Println()
			continue
		}

		if shelfName == "" {
			shelfName = defaultShelfName
			fmt.Printf("  Using: %s\n", color.GreenString(shelfName))
		}

		// Check if shelf name already exists in config
		if cfg.ShelfByName(shelfName) != nil {
			fmt.Println()
			fmt.Println(color.RedString("✗ Shelf name %q is already in your config", shelfName))
			fmt.Println()
			fmt.Print("Enter a different shelf name: ")
			continue
		}

		break
	}

	// Ask about repository visibility
	fmt.Println()
	fmt.Println(color.CyanString("Repository visibility:"))
	fmt.Println()
	fmt.Println(color.GreenString("  1. Private") + " (default) - Only you can see this repo")
	fmt.Println(color.YellowString("  2. Public") + " - Anyone can see this repo")
	fmt.Println()
	fmt.Print("Your choice (1/2): ")
	var visibilityChoice string
	_, _ = fmt.Scanln(&visibilityChoice)

	isPrivate := true // default
	visibilityLabel := "Private"
	if visibilityChoice == "2" {
		isPrivate = false
		visibilityLabel = "Public"
	}

	// Confirm creation
	fmt.Println()
	fmt.Println(color.CyanString("Summary:"))
	fmt.Printf("  GitHub repository:  %s/%s\n", cfg.GitHub.Owner, color.WhiteString(repoName))
	fmt.Printf("  Visibility:         %s\n", color.WhiteString(visibilityLabel))
	fmt.Printf("  Release tag:        %s\n", color.WhiteString("library"))
	fmt.Printf("  Shelf name (config):%s\n", color.WhiteString(shelfName))
	fmt.Println()
	fmt.Printf("You'll use the shelf name in commands: %s\n",
		color.CyanString(fmt.Sprintf("shelfctl shelve --shelf %s", shelfName)))
	fmt.Println()
	fmt.Print("Proceed? (y/n): ")
	var confirm string
	_, _ = fmt.Scanln(&confirm)

	if confirm != "y" && confirm != "Y" && confirm != "yes" {
		fmt.Println(color.YellowString("Cancelled."))
		fmt.Println()
		fmt.Println("You can run this manually anytime:")
		fmt.Printf("  %s\n", color.CyanString(fmt.Sprintf("shelfctl init --repo %s --name %s --create-repo --create-release", repoName, shelfName)))
		return nil
	}

	// Run init command
	fmt.Println()
	initCmd := newInitCmd()
	args := []string{
		"--repo", repoName,
		"--name", shelfName,
		"--create-repo",
		"--create-release",
	}
	if !isPrivate {
		args = append(args, "--private=false")
	}
	initCmd.SetArgs(args)

	if err := initCmd.Execute(); err != nil {
		return fmt.Errorf("init failed: %w", err)
	}

	fmt.Println()
	fmt.Println(color.GreenString("✓ Shelf created successfully!"))
	fmt.Println()
	fmt.Println("What's next?")
	fmt.Printf("  1. Add your first book:\n")
	fmt.Printf("     %s\n\n", color.CyanString("shelfctl shelve"))
	fmt.Printf("  2. Or run the interactive menu:\n")
	fmt.Printf("     %s\n", color.CyanString("shelfctl"))

	return nil
}

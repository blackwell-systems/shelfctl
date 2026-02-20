package app

import (
	"fmt"

	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newDeleteShelfCmd() *cobra.Command {
	var (
		deleteRepo  bool
		skipConfirm bool
	)

	cmd := &cobra.Command{
		Use:   "delete-shelf <name>",
		Short: "Remove a shelf from your configuration",
		Long: `Remove a shelf from your shelfctl configuration.

By default, this only removes the shelf from your local config file.
The GitHub repository and its contents remain untouched.

Use --delete-repo to also delete the GitHub repository (DESTRUCTIVE).

Examples:
  # Interactive shelf picker
  shelfctl delete-shelf

  # Remove specific shelf from config only
  shelfctl delete-shelf old-books

  # Remove shelf AND delete the GitHub repo
  shelfctl delete-shelf old-books --delete-repo --yes
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var shelfName string

			// If no shelf name provided, show picker
			if len(args) == 0 {
				if len(cfg.Shelves) == 0 {
					return fmt.Errorf("no shelves configured")
				}

				// Build shelf options
				var options []tui.ShelfOption
				for _, s := range cfg.Shelves {
					options = append(options, tui.ShelfOption{
						Name: s.Name,
						Repo: s.Repo,
					})
				}

				selected, err := tui.RunShelfPicker(options)
				if err != nil {
					return err
				}
				shelfName = selected
			} else {
				shelfName = args[0]
			}

			// Find the shelf in config
			shelf := cfg.ShelfByName(shelfName)
			if shelf == nil {
				return fmt.Errorf("shelf %q not found in config", shelfName)
			}

			owner := shelf.EffectiveOwner(cfg.GitHub.Owner)

			fmt.Println(color.YellowString("⚠ Warning: You are about to delete a shelf"))
			fmt.Println()
			fmt.Printf("Shelf name: %s\n", color.WhiteString(shelfName))
			fmt.Printf("Repository: %s/%s\n", owner, color.WhiteString(shelf.Repo))
			fmt.Println()

			if deleteRepo {
				fmt.Println(color.RedString("--delete-repo is set: The GitHub repository will be PERMANENTLY DELETED"))
				fmt.Println()
				fmt.Println("This will:")
				fmt.Println("  • Remove shelf from your config")
				fmt.Println("  • " + color.RedString("DELETE") + " the GitHub repository " + color.RedString(fmt.Sprintf("%s/%s", owner, shelf.Repo)))
				fmt.Println("  • " + color.RedString("DELETE") + " all books (release assets)")
				fmt.Println("  • " + color.RedString("DELETE") + " all catalog history")
				fmt.Println()
				fmt.Println(color.RedString("THIS CANNOT BE UNDONE"))
			} else {
				fmt.Println("This will:")
				fmt.Println("  • Remove shelf from your config")
				fmt.Println("  • " + color.GreenString("Keep") + " the GitHub repository (you can re-add it later)")
				fmt.Println("  • " + color.GreenString("Keep") + " all books and catalog data")
			}

			fmt.Println()

			// Confirm
			if !skipConfirm {
				fmt.Print("Type the shelf name to confirm deletion: ")
				var confirmation string
				_, _ = fmt.Scanln(&confirmation)

				if confirmation != shelfName {
					return fmt.Errorf("confirmation did not match - aborted")
				}
			}

			// Delete from config
			newShelves := make([]config.ShelfConfig, 0, len(cfg.Shelves)-1)
			for _, s := range cfg.Shelves {
				if s.Name != shelfName {
					newShelves = append(newShelves, s)
				}
			}

			currentCfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			currentCfg.Shelves = newShelves

			if err := config.Save(currentCfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			ok("Removed shelf %q from config", shelfName)

			// Delete GitHub repo if requested
			if deleteRepo {
				header("Deleting GitHub repository %s/%s …", owner, shelf.Repo)

				if err := gh.DeleteRepo(owner, shelf.Repo); err != nil {
					return fmt.Errorf("deleting repository: %w", err)
				}

				ok("Repository deleted: %s/%s", owner, shelf.Repo)
			} else {
				fmt.Println()
				fmt.Println(color.GreenString("GitHub repository preserved: https://github.com/%s/%s", owner, shelf.Repo))
				fmt.Println()
				fmt.Println("To delete the repository later, use:")
				fmt.Printf("  %s\n", color.CyanString(fmt.Sprintf("shelfctl delete-shelf %s --delete-repo", shelfName)))
			}

			// Note about re-adding
			if !deleteRepo {
				fmt.Println()
				fmt.Println("To re-add this shelf later:")
				fmt.Printf("  %s\n", color.CyanString(fmt.Sprintf("shelfctl init --repo %s --name %s", shelf.Repo, shelfName)))
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&deleteRepo, "delete-repo", false, "Also delete the GitHub repository (DESTRUCTIVE)")
	cmd.Flags().BoolVar(&skipConfirm, "yes", false, "Skip confirmation prompt")

	return cmd
}

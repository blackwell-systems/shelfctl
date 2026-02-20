package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/spf13/cobra"
)

func newSplitCmd() *cobra.Command {
	var (
		shelfName string
		byTag     bool
		dryRun    bool
		maxN      int
	)

	cmd := &cobra.Command{
		Use:   "split --shelf <name>",
		Short: "Interactive wizard to split a shelf into sub-releases by tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			if shelfName == "" {
				return fmt.Errorf("--shelf is required")
			}

			books, err := loadShelfBooks(shelfName)
			if err != nil {
				return err
			}

			if len(books) == 0 {
				fmt.Println("No books in this shelf.")
				return nil
			}

			if !byTag {
				return fmt.Errorf("currently only --by-tag splitting is supported")
			}

			tagMap := buildTagMap(books)
			if len(tagMap) == 0 {
				fmt.Println("No tags found — cannot split by tag.")
				return nil
			}

			proposals := collectSplitProposals(shelfName, tagMap)
			if len(proposals) == 0 {
				fmt.Println("Nothing to move.")
				return nil
			}

			displayProposedMoves(proposals, maxN)

			if dryRun {
				fmt.Println("\n(dry run — no changes made)")
				return nil
			}

			return executeSplitMoves(shelfName, proposals, maxN)
		},
	}

	cmd.Flags().StringVar(&shelfName, "shelf", "", "Shelf to split")
	cmd.Flags().BoolVar(&byTag, "by-tag", true, "Group books by tag and propose sub-releases")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show proposed moves without executing")
	cmd.Flags().IntVar(&maxN, "n", 0, "Limit number of books processed per run (0 = unlimited)")
	return cmd
}

type splitProposal struct {
	release string
	bookIDs []string
}

func loadShelfBooks(shelfName string) ([]catalog.Book, error) {
	shelf := cfg.ShelfByName(shelfName)
	if shelf == nil {
		return nil, fmt.Errorf("shelf %q not found in config", shelfName)
	}
	owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
	catalogPath := shelf.EffectiveCatalogPath()

	data, _, err := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
	if err != nil {
		return nil, err
	}
	return catalog.Parse(data)
}

func buildTagMap(books []catalog.Book) map[string][]string {
	tagMap := map[string][]string{} // tag → book IDs
	for _, b := range books {
		for _, t := range b.Tags {
			tagMap[t] = append(tagMap[t], b.ID)
		}
	}
	return tagMap
}

func collectSplitProposals(shelfName string, tagMap map[string][]string) []splitProposal {
	header("Proposed split for shelf: %s", shelfName)
	var proposals []splitProposal

	sc := bufio.NewScanner(os.Stdin)
	for tag, ids := range tagMap {
		fmt.Printf("\n  Tag: %s  (%d books)\n", tag, len(ids))
		for _, id := range ids {
			fmt.Printf("    - %s\n", id)
		}
		fmt.Printf("  Move to release (enter tag name, or blank to skip): ")
		if !sc.Scan() {
			break
		}
		rel := strings.TrimSpace(sc.Text())
		if rel == "" {
			fmt.Println("  (skipped)")
			continue
		}
		proposals = append(proposals, splitProposal{release: rel, bookIDs: ids})
	}

	return proposals
}

func displayProposedMoves(proposals []splitProposal, maxN int) {
	header("\nProposed moves:")
	count := 0
	for _, p := range proposals {
		for _, id := range p.bookIDs {
			if maxN > 0 && count >= maxN {
				fmt.Printf("  (limit of %d reached)\n", maxN)
				break
			}
			fmt.Printf("  %s → release/%s\n", id, p.release)
			count++
		}
	}
}

func executeSplitMoves(shelfName string, proposals []splitProposal, maxN int) error {
	sc := bufio.NewScanner(os.Stdin)
	fmt.Print("\nProceed? [y/N]: ")
	if !sc.Scan() || strings.ToLower(strings.TrimSpace(sc.Text())) != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	count := 0
	for _, p := range proposals {
		for _, id := range p.bookIDs {
			if maxN > 0 && count >= maxN {
				break
			}
			fmt.Printf("Moving %s → %s …\n", id, p.release)
			moveCmd := newMoveCmd()
			moveCmd.SetArgs([]string{id, "--shelf", shelfName, "--to-release", p.release})
			if err := moveCmd.Execute(); err != nil {
				warn("Failed to move %s: %v", id, err)
			}
			count++
		}
	}
	ok("Split complete")
	return nil
}

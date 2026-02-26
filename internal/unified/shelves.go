package unified

import (
	"fmt"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type shelvesStatus struct {
	name      string
	repo      string
	owner     string
	bookCount int
	repoOK    bool
	catalogOK bool
	releaseOK bool
	errorMsg  string
}

type shelvesLoadCompleteMsg struct {
	statuses []shelvesStatus
}

// ShelvesModel displays configured shelves and their health in the TUI
type ShelvesModel struct {
	gh  *github.Client
	cfg *config.Config

	width, height int
	loading       bool
	statuses      []shelvesStatus
}

// NewShelvesModel creates a new shelves view
func NewShelvesModel(gh *github.Client, cfg *config.Config) ShelvesModel {
	return ShelvesModel{
		gh:      gh,
		cfg:     cfg,
		loading: true,
	}
}

// Init starts async loading of shelf statuses
func (m ShelvesModel) Init() tea.Cmd {
	return m.loadAsync()
}

// Update handles messages
func (m ShelvesModel) Update(msg tea.Msg) (ShelvesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			if msg.String() == "ctrl+c" {
				return m, func() tea.Msg { return QuitAppMsg{} }
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "esc", "enter":
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		case "ctrl+c":
			return m, func() tea.Msg { return QuitAppMsg{} }
		}

	case shelvesLoadCompleteMsg:
		m.loading = false
		m.statuses = msg.statuses
		return m, nil
	}

	return m, nil
}

func (m ShelvesModel) loadAsync() tea.Cmd {
	gh := m.gh
	cfg := m.cfg

	return func() tea.Msg {
		var statuses []shelvesStatus

		for _, shelf := range cfg.Shelves {
			owner := shelf.EffectiveOwner(cfg.GitHub.Owner)
			release := shelf.EffectiveRelease(cfg.Defaults.Release)
			catalogPath := shelf.EffectiveCatalogPath()

			s := shelvesStatus{
				name:  shelf.Name,
				repo:  shelf.Repo,
				owner: owner,
			}

			// Check repo
			exists, err := gh.RepoExists(owner, shelf.Repo)
			if err != nil {
				s.errorMsg = fmt.Sprintf("repo error: %v", err)
				statuses = append(statuses, s)
				continue
			}
			if !exists {
				s.errorMsg = "repo not found"
				statuses = append(statuses, s)
				continue
			}
			s.repoOK = true

			// Check catalog
			catalogData, _, catalogErr := gh.GetFileContent(owner, shelf.Repo, catalogPath, "")
			if catalogErr != nil {
				s.errorMsg = "catalog.yml missing"
				statuses = append(statuses, s)
				continue
			}
			s.catalogOK = true
			if books, err := catalog.Parse(catalogData); err == nil {
				s.bookCount = len(books)
			}

			// Check release
			_, err = gh.GetReleaseByTag(owner, shelf.Repo, release)
			if err != nil {
				s.errorMsg = fmt.Sprintf("release '%s' missing", release)
			} else {
				s.releaseOK = true
			}

			statuses = append(statuses, s)
		}

		return shelvesLoadCompleteMsg{statuses: statuses}
	}
}

// View renders the shelves view
func (m ShelvesModel) View() string {
	style := lipgloss.NewStyle().Padding(2, 4)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("28"))
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	var b strings.Builder

	b.WriteString(tui.StyleHeader.Render("Configured Shelves"))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString(tui.StyleNormal.Render("  Loading shelf statuses..."))
		b.WriteString("\n")
	} else if len(m.statuses) == 0 {
		b.WriteString(tui.StyleNormal.Render("  No shelves configured."))
		b.WriteString("\n")
		b.WriteString(dim.Render("  Run: shelfctl init --repo shelf-<topic> --name <topic>"))
		b.WriteString("\n")
	} else {
		for _, s := range m.statuses {
			// Shelf name + repo
			b.WriteString(tui.StyleHighlight.Render(fmt.Sprintf("  %s", s.name)))
			b.WriteString(dim.Render(fmt.Sprintf("  %s/%s", s.owner, s.repo)))
			b.WriteString("\n")

			// Book count
			if s.catalogOK {
				booksLabel := "books"
				if s.bookCount == 1 {
					booksLabel = "book"
				}
				b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("    %d %s", s.bookCount, booksLabel)))
			} else {
				b.WriteString(dim.Render("    -"))
			}

			// Status
			b.WriteString("  ")
			if s.repoOK && s.catalogOK && s.releaseOK {
				b.WriteString(green.Render("healthy"))
			} else if s.errorMsg != "" {
				if !s.repoOK || !s.catalogOK {
					b.WriteString(red.Render(s.errorMsg))
				} else {
					b.WriteString(yellow.Render(s.errorMsg))
				}
			}
			b.WriteString("\n\n")
		}
	}

	b.WriteString(tui.StyleHelp.Render("Press Enter to return to menu"))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

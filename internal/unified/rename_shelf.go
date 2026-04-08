package unified

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/config"
	"github.com/blackwell-systems/shelfctl/internal/github"
	"github.com/blackwell-systems/shelfctl/internal/tui"
	"github.com/blackwell-systems/shelfctl/internal/tui/delegate"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type renameShelfPhase int

const (
	renameShelfPicking    renameShelfPhase = iota // pick which shelf
	renameShelfInput                              // enter new name + repo
	renameShelfProcessing                         // async: rename repo + update config + move cache
	renameShelfDone                               // summary
)

type renameShelfCompleteMsg struct {
	newName string
	newRepo string
	err     error
}

// RenameShelfModel is the unified view for renaming a shelf
type RenameShelfModel struct {
	phase renameShelfPhase

	gh       *github.Client
	cfg      *config.Config
	cacheDir string

	width, height int
	err           error
	empty         bool

	// Shelf picker
	shelfList    list.Model
	shelfOptions []tui.ShelfOption

	// Selected shelf
	shelfName  string
	shelfRepo  string
	shelfOwner string

	// Input fields: [0] = display name, [1] = repo name suffix
	inputs  []textinput.Model
	focused int

	// Result
	message   string
	activeCmd string
}

// NewRenameShelfModel creates a new rename-shelf view
func NewRenameShelfModel(gh *github.Client, cfg *config.Config, cacheDir string) RenameShelfModel {
	if len(cfg.Shelves) == 0 {
		return RenameShelfModel{gh: gh, cfg: cfg, cacheDir: cacheDir, empty: true}
	}

	var options []tui.ShelfOption
	for _, s := range cfg.Shelves {
		options = append(options, tui.ShelfOption{Name: s.Name, Repo: s.Repo})
	}

	m := RenameShelfModel{
		phase:        renameShelfPicking,
		gh:           gh,
		cfg:          cfg,
		cacheDir:     cacheDir,
		shelfOptions: options,
	}

	// Auto-select if single shelf
	if len(options) == 1 {
		m.shelfName = options[0].Name
		m.shelfRepo = options[0].Repo
		shelf := cfg.ShelfByName(m.shelfName)
		if shelf != nil {
			m.shelfOwner = shelf.EffectiveOwner(cfg.GitHub.Owner)
		}
		m.phase = renameShelfInput
		m.inputs = m.buildInputs()
	} else {
		m.shelfList = m.buildShelfList()
	}

	return m
}

func (m RenameShelfModel) buildShelfList() list.Model {
	items := make([]list.Item, len(m.shelfOptions))
	for i, s := range m.shelfOptions {
		items[i] = s
	}
	d := delegate.New(func(w io.Writer, lm list.Model, index int, item list.Item) {
		so, ok := item.(tui.ShelfOption)
		if !ok {
			return
		}
		isSelected := index == lm.Index()
		label := fmt.Sprintf("%-20s  %s", so.Name, tui.StyleHelp.Render(so.Repo))
		if isSelected {
			fmt.Fprint(w, tui.StyleHighlight.Render("› "+label))
		} else {
			fmt.Fprint(w, "  "+tui.StyleNormal.Render(label))
		}
	})
	l := list.New(items, d, 0, 0)
	l.Title = "Select Shelf to Rename"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = tui.StyleHeader
	l.Styles.HelpStyle = tui.StyleHelp
	return l
}

func (m RenameShelfModel) buildInputs() []textinput.Model {
	inputs := make([]textinput.Model, 2)
	const inputWidth = 40

	// Strip "shelf-" prefix for display in repo field
	repoSuffix := strings.TrimPrefix(m.shelfRepo, "shelf-")

	// Display name
	inputs[0] = textinput.New()
	inputs[0].SetValue(m.shelfName)
	inputs[0].CharLimit = 50
	inputs[0].Width = inputWidth
	inputs[0].Prompt = ""
	inputs[0].Focus()

	// Repo name (suffix after "shelf-")
	inputs[1] = textinput.New()
	inputs[1].SetValue(repoSuffix)
	inputs[1].CharLimit = 100
	inputs[1].Width = inputWidth
	inputs[1].Prompt = ""

	return inputs
}

func (m RenameShelfModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m RenameShelfModel) Update(msg tea.Msg) (RenameShelfModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.phase == renameShelfPicking {
			h, v := tui.StyleBorder.GetFrameSize()
			m.shelfList.SetSize(msg.Width-h, msg.Height-v)
		}
		return m, nil

	case tui.ClearActiveCmdMsg:
		m.activeCmd = ""
		return m, nil

	case tea.KeyMsg:
		if m.empty {
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		}
		if m.phase == renameShelfProcessing {
			if msg.String() == "ctrl+c" {
				return m, func() tea.Msg { return QuitAppMsg{} }
			}
			return m, nil
		}
		if m.phase == renameShelfDone {
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		}
		switch m.phase {
		case renameShelfPicking:
			return m.updatePicking(msg)
		case renameShelfInput:
			return m.updateInput(msg)
		}

	case renameShelfCompleteMsg:
		m.phase = renameShelfDone
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.message = fmt.Sprintf("Renamed to %q (repo: shelf-%s)", msg.newName, msg.newRepo)
		return m, nil
	}

	// Forward non-key messages to sub-models
	switch m.phase {
	case renameShelfPicking:
		var cmd tea.Cmd
		m.shelfList, cmd = m.shelfList.Update(msg)
		return m, cmd
	case renameShelfInput:
		return m.updateFormInputs(msg)
	}

	return m, nil
}

func (m RenameShelfModel) updatePicking(msg tea.KeyMsg) (RenameShelfModel, tea.Cmd) {
	if m.shelfList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.shelfList, cmd = m.shelfList.Update(msg)
		return m, cmd
	}
	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }
	case "q", "esc":
		return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
	case "enter":
		if item, ok := m.shelfList.SelectedItem().(tui.ShelfOption); ok {
			m.shelfName = item.Name
			m.shelfRepo = item.Repo
			shelf := m.cfg.ShelfByName(m.shelfName)
			if shelf != nil {
				m.shelfOwner = shelf.EffectiveOwner(m.cfg.GitHub.Owner)
			}
			m.phase = renameShelfInput
			m.inputs = m.buildInputs()
			m.focused = 0
			return m, textinput.Blink
		}
	}
	var cmd tea.Cmd
	m.shelfList, cmd = m.shelfList.Update(msg)
	return m, cmd
}

func (m RenameShelfModel) updateInput(msg tea.KeyMsg) (RenameShelfModel, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return QuitAppMsg{} }
	case "esc":
		if len(m.shelfOptions) > 1 {
			m.phase = renameShelfPicking
			return m, nil
		}
		return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
	case "enter":
		newName := strings.TrimSpace(m.inputs[0].Value())
		newRepoSuffix := strings.TrimSpace(m.inputs[1].Value())
		if newName == "" {
			m.err = fmt.Errorf("shelf name cannot be empty")
			return m, nil
		}
		if newRepoSuffix == "" {
			m.err = fmt.Errorf("repo name cannot be empty")
			return m, nil
		}
		newRepo := "shelf-" + newRepoSuffix
		// Check for name collision (ignore current shelf)
		for _, s := range m.cfg.Shelves {
			if s.Name == newName && s.Name != m.shelfName {
				m.err = fmt.Errorf("a shelf named %q already exists", newName)
				return m, nil
			}
			if s.Repo == newRepo && s.Repo != m.shelfRepo {
				m.err = fmt.Errorf("a shelf with repo %q already exists", newRepo)
				return m, nil
			}
		}
		m.err = nil
		m.activeCmd = "enter"
		m.phase = renameShelfProcessing
		return m, tea.Batch(
			m.renameAsync(newName, newRepo),
			tui.HighlightCmd(),
		)
	case "tab", "down":
		m.activeCmd = "tab"
		if m.focused < len(m.inputs) {
			m.inputs[m.focused].Blur()
		}
		m.focused = (m.focused + 1) % len(m.inputs)
		return m, tea.Batch(m.inputs[m.focused].Focus(), tui.HighlightCmd())
	case "shift+tab", "up":
		m.activeCmd = "tab"
		if m.focused < len(m.inputs) {
			m.inputs[m.focused].Blur()
		}
		m.focused--
		if m.focused < 0 {
			m.focused = len(m.inputs) - 1
		}
		return m, tea.Batch(m.inputs[m.focused].Focus(), tui.HighlightCmd())
	}
	return m.updateFormInputs(msg)
}

func (m RenameShelfModel) updateFormInputs(msg tea.Msg) (RenameShelfModel, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}

func (m RenameShelfModel) renameAsync(newName, newRepo string) tea.Cmd {
	gh := m.gh
	owner := m.shelfOwner
	oldRepo := m.shelfRepo
	oldName := m.shelfName
	cacheDir := m.cacheDir

	return func() tea.Msg {
		// 1. Rename GitHub repo
		if _, err := gh.RenameRepo(owner, oldRepo, newRepo); err != nil {
			return renameShelfCompleteMsg{err: fmt.Errorf("renaming repo: %w", err)}
		}

		// 2. Update config
		currentCfg, err := config.Load()
		if err != nil {
			return renameShelfCompleteMsg{err: fmt.Errorf("loading config: %w", err)}
		}
		for i := range currentCfg.Shelves {
			if currentCfg.Shelves[i].Name == oldName {
				currentCfg.Shelves[i].Name = newName
				currentCfg.Shelves[i].Repo = newRepo
				break
			}
		}
		if err := config.Save(currentCfg); err != nil {
			return renameShelfCompleteMsg{err: fmt.Errorf("saving config: %w", err)}
		}

		// 3. Rename local cache directory (best-effort — don't fail rename if cache move fails)
		oldCacheDir := cacheDir + "/" + oldRepo
		newCacheDir := cacheDir + "/" + newRepo
		if _, err := os.Stat(oldCacheDir); err == nil {
			_ = os.Rename(oldCacheDir, newCacheDir)
		}

		// Strip "shelf-" prefix for display
		newRepoSuffix := strings.TrimPrefix(newRepo, "shelf-")
		return renameShelfCompleteMsg{newName: newName, newRepo: newRepoSuffix}
	}
}

func (m RenameShelfModel) View() string {
	outerStyle := lipgloss.NewStyle().Padding(2, 4)
	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)

	var b strings.Builder

	switch m.phase {
	case renameShelfPicking:
		return outerStyle.Render(tui.StyleBorder.Render(innerPadding.Render(m.shelfList.View())))

	case renameShelfInput:
		b.WriteString(tui.StyleHeader.Render("Rename Shelf"))
		b.WriteString("\n\n")
		b.WriteString(tui.StyleHelp.Render(fmt.Sprintf("Renaming: %s  (%s)", m.shelfName, m.shelfRepo)))
		b.WriteString("\n\n")

		if m.err != nil {
			b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("Error: %v", m.err)))
			b.WriteString("\n\n")
		}

		// Display name field
		if m.focused == 0 {
			b.WriteString(tui.StyleHighlight.Render("› Display Name:"))
		} else {
			b.WriteString(tui.StyleNormal.Render("  Display Name:"))
		}
		b.WriteString("\n  ")
		b.WriteString(m.inputs[0].View())
		b.WriteString("\n\n")

		// Repo name field
		if m.focused == 1 {
			b.WriteString(tui.StyleHighlight.Render("› Repository Name:"))
		} else {
			b.WriteString(tui.StyleNormal.Render("  Repository Name:"))
		}
		b.WriteString("\n  ")
		b.WriteString(tui.StyleHelp.Render("shelf-"))
		b.WriteString(m.inputs[1].View())
		b.WriteString("\n\n")

		b.WriteString(tui.RenderFooterBar([]tui.ShortcutEntry{
			{Key: "tab", Label: "Tab/↑↓ Navigate"},
			{Key: "enter", Label: "Enter Rename"},
			{Key: "esc", Label: "Esc Cancel"},
		}, m.activeCmd))

	case renameShelfProcessing:
		b.WriteString(tui.StyleHeader.Render("Rename Shelf"))
		b.WriteString("\n\n")
		b.WriteString(tui.StyleHighlight.Render("⏳ Renaming..."))
		b.WriteString("\n")
		b.WriteString(tui.StyleHelp.Render("Renaming GitHub repo and updating config"))

	case renameShelfDone:
		b.WriteString(tui.StyleHeader.Render("Rename Shelf"))
		b.WriteString("\n\n")
		if m.err != nil {
			b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("✗ Error: %v", m.err)))
		} else {
			b.WriteString(tui.StyleCached.Render("✓ " + m.message))
		}
		b.WriteString("\n\n")
		b.WriteString(tui.StyleHelp.Render("Press any key to return to menu"))
	}

	return outerStyle.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

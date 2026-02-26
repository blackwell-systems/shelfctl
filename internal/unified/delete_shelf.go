package unified

import (
	"fmt"
	"io"
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

type deleteShelfPhase int

const (
	deleteShelfPicking    deleteShelfPhase = iota // shelf list picker
	deleteShelfConfirming                         // confirm: keep repo or delete repo
	deleteShelfTypeName                           // type shelf name to confirm
	deleteShelfProcessing                         // async: remove from config + optional repo delete
	deleteShelfDone                               // summary
)

type deleteShelfCompleteMsg struct{ err error }

// DeleteShelfModel is the unified view for removing a shelf
type DeleteShelfModel struct {
	phase deleteShelfPhase

	gh  *github.Client
	cfg *config.Config

	width, height int
	err           error
	empty         bool

	// Shelf picker
	shelfList    list.Model
	shelfOptions []tui.ShelfOption

	// Selected shelf
	shelfName  string
	shelfOwner string
	shelfRepo  string

	// Confirm options
	deleteRepo   bool
	optionIdx    int // 0 = keep, 1 = delete
	confirmInput textinput.Model

	// Result
	done    bool
	message string

	activeCmd string
}

// NewDeleteShelfModel creates a new delete-shelf view
func NewDeleteShelfModel(gh *github.Client, cfg *config.Config) DeleteShelfModel {
	if len(cfg.Shelves) == 0 {
		return DeleteShelfModel{gh: gh, cfg: cfg, empty: true}
	}

	var options []tui.ShelfOption
	for _, s := range cfg.Shelves {
		options = append(options, tui.ShelfOption{Name: s.Name, Repo: s.Repo})
	}

	m := DeleteShelfModel{
		phase:        deleteShelfPicking,
		gh:           gh,
		cfg:          cfg,
		shelfOptions: options,
	}

	// Auto-select if single shelf
	if len(options) == 1 {
		shelf := cfg.ShelfByName(options[0].Name)
		if shelf != nil {
			m.shelfName = shelf.Name
			m.shelfOwner = shelf.EffectiveOwner(cfg.GitHub.Owner)
			m.shelfRepo = shelf.Repo
			m.phase = deleteShelfConfirming
		}
	} else {
		m.shelfList = m.createShelfList()
	}

	return m
}

func (m DeleteShelfModel) Init() tea.Cmd { return nil }

func (m DeleteShelfModel) Update(msg tea.Msg) (DeleteShelfModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.phase == deleteShelfPicking {
			h, v := tui.StyleBorder.GetFrameSize()
			m.shelfList.SetSize(msg.Width-h, msg.Height-v)
		}
		return m, nil

	case tui.ClearActiveCmdMsg:
		m.activeCmd = ""
		return m, nil

	case tea.KeyMsg:
		if m.empty || m.phase == deleteShelfDone {
			if msg.String() == "enter" || msg.String() == "esc" || msg.String() == "q" {
				return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
			}
			return m, nil
		}

		switch m.phase {
		case deleteShelfPicking:
			return m.updatePicking(msg)
		case deleteShelfConfirming:
			return m.updateConfirming(msg)
		case deleteShelfTypeName:
			return m.updateTypeName(msg)
		case deleteShelfProcessing:
			if msg.String() == "ctrl+c" {
				return m, func() tea.Msg { return QuitAppMsg{} }
			}
			return m, nil
		}

	case deleteShelfCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		m.phase = deleteShelfDone
		return m, nil
	}

	// Forward non-key messages
	switch m.phase {
	case deleteShelfPicking:
		var cmd tea.Cmd
		m.shelfList, cmd = m.shelfList.Update(msg)
		return m, cmd
	case deleteShelfTypeName:
		var cmd tea.Cmd
		m.confirmInput, cmd = m.confirmInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m DeleteShelfModel) updatePicking(msg tea.KeyMsg) (DeleteShelfModel, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
	case "enter":
		if item, ok := m.shelfList.SelectedItem().(tui.ShelfOption); ok {
			shelf := m.cfg.ShelfByName(item.Name)
			if shelf != nil {
				m.shelfName = shelf.Name
				m.shelfOwner = shelf.EffectiveOwner(m.cfg.GitHub.Owner)
				m.shelfRepo = shelf.Repo
				m.phase = deleteShelfConfirming
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.shelfList, cmd = m.shelfList.Update(msg)
	return m, cmd
}

func (m DeleteShelfModel) updateConfirming(msg tea.KeyMsg) (DeleteShelfModel, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		if len(m.shelfOptions) == 1 {
			return m, func() tea.Msg { return NavigateMsg{Target: "hub"} }
		}
		m.phase = deleteShelfPicking
		return m, nil
	case "up", "k":
		if m.optionIdx > 0 {
			m.optionIdx--
		}
	case "down", "j":
		if m.optionIdx < 1 {
			m.optionIdx++
		}
	case "1":
		m.optionIdx = 0
	case "2":
		m.optionIdx = 1
	case "enter":
		m.deleteRepo = m.optionIdx == 1

		// For "keep repo" — go straight to processing (safe, no extra confirm)
		if !m.deleteRepo {
			m.phase = deleteShelfProcessing
			return m, m.executeAsync()
		}

		// For "delete repo" — require typing the shelf name
		ti := textinput.New()
		ti.Placeholder = m.shelfName
		ti.Focus()
		ti.CharLimit = 100
		ti.Width = 40
		m.confirmInput = ti
		m.phase = deleteShelfTypeName
		return m, textinput.Blink
	}
	return m, nil
}

func (m DeleteShelfModel) updateTypeName(msg tea.KeyMsg) (DeleteShelfModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.phase = deleteShelfConfirming
		m.err = nil
		return m, nil
	case "enter":
		typed := strings.TrimSpace(m.confirmInput.Value())
		if typed != m.shelfName {
			m.err = fmt.Errorf("name does not match — type %q to confirm", m.shelfName)
			return m, nil
		}
		m.err = nil
		m.phase = deleteShelfProcessing
		return m, m.executeAsync()
	}

	var cmd tea.Cmd
	m.confirmInput, cmd = m.confirmInput.Update(msg)
	return m, cmd
}

func (m DeleteShelfModel) executeAsync() tea.Cmd {
	gh := m.gh
	shelfName := m.shelfName
	shelfOwner := m.shelfOwner
	shelfRepo := m.shelfRepo
	deleteRepo := m.deleteRepo

	return func() tea.Msg {
		// Remove from config
		currentCfg, err := config.Load()
		if err != nil {
			return deleteShelfCompleteMsg{err: fmt.Errorf("loading config: %w", err)}
		}

		newShelves := make([]config.ShelfConfig, 0, len(currentCfg.Shelves))
		for _, s := range currentCfg.Shelves {
			if s.Name != shelfName {
				newShelves = append(newShelves, s)
			}
		}
		currentCfg.Shelves = newShelves

		if err := config.Save(currentCfg); err != nil {
			return deleteShelfCompleteMsg{err: fmt.Errorf("saving config: %w", err)}
		}

		// Delete GitHub repo if requested
		if deleteRepo {
			if err := gh.DeleteRepo(shelfOwner, shelfRepo); err != nil {
				return deleteShelfCompleteMsg{err: fmt.Errorf("removed from config, but failed to delete repo: %w", err)}
			}
		}

		return deleteShelfCompleteMsg{}
	}
}

func (m DeleteShelfModel) createShelfList() list.Model {
	items := make([]list.Item, len(m.shelfOptions))
	for i := range m.shelfOptions {
		items[i] = m.shelfOptions[i]
	}
	d := delegate.New(func(w io.Writer, ml list.Model, index int, item list.Item) {
		opt, ok := item.(tui.ShelfOption)
		if !ok {
			return
		}
		cursor := "  "
		if index == ml.Index() {
			cursor = "> "
		}
		_, _ = fmt.Fprintf(w, "%s%s (%s)", cursor, opt.Name, opt.Repo)
	})
	l := list.New(items, d, 0, 0)
	l.Title = "Select shelf to delete"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = tui.StyleHeader
	return l
}

// View renders the delete-shelf view
func (m DeleteShelfModel) View() string {
	if m.empty {
		return m.renderMsg("No shelves configured", "Press Enter to return to menu")
	}

	switch m.phase {
	case deleteShelfPicking:
		return tui.RenderWithFooter(m.shelfList.View(), []tui.ShortcutEntry{
			{Key: "enter", Label: "enter select"},
			{Key: "q", Label: "q/esc back"},
		}, m.activeCmd)
	case deleteShelfConfirming:
		return m.renderConfirm()
	case deleteShelfTypeName:
		return m.renderTypeName()
	case deleteShelfProcessing:
		action := "Removing shelf from config..."
		if m.deleteRepo {
			action = "Deleting shelf and GitHub repository..."
		}
		return m.renderMsg(action, "Please wait")
	case deleteShelfDone:
		return m.renderDone()
	}
	return ""
}

func (m DeleteShelfModel) renderMsg(title, help string) string {
	style := lipgloss.NewStyle().Padding(2, 4)
	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render(title))
	b.WriteString("\n\n")
	b.WriteString(tui.StyleHelp.Render(help))
	b.WriteString("\n")
	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

func (m DeleteShelfModel) renderConfirm() string {
	style := lipgloss.NewStyle().Padding(2, 4)
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	var b strings.Builder
	b.WriteString(tui.StyleHeader.Render("Delete Shelf"))
	b.WriteString("\n\n")

	b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  Shelf: %s", m.shelfName)))
	b.WriteString("\n")
	b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  Repo:  %s/%s", m.shelfOwner, m.shelfRepo)))
	b.WriteString("\n\n")

	b.WriteString(tui.StyleNormal.Render("What should happen to the GitHub repository?"))
	b.WriteString("\n\n")

	options := []struct {
		label string
		desc  string
	}{
		{"Keep repository", "Remove from config only — repo and books stay on GitHub"},
		{"Delete permanently", "Delete repo and all books forever (CANNOT BE UNDONE)"},
	}

	for i, opt := range options {
		if i == m.optionIdx {
			if i == 1 {
				b.WriteString(tui.StyleHighlight.Render(fmt.Sprintf("› %d. %s", i+1, opt.label)))
			} else {
				b.WriteString(tui.StyleHighlight.Render(fmt.Sprintf("› %d. %s", i+1, opt.label)))
			}
		} else {
			if i == 1 {
				b.WriteString(red.Render(fmt.Sprintf("  %d. %s", i+1, opt.label)))
			} else {
				b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  %d. %s", i+1, opt.label)))
			}
		}
		b.WriteString("\n")
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		b.WriteString(dim.Render(fmt.Sprintf("     %s", opt.desc)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(tui.RenderFooterBar([]tui.ShortcutEntry{
		{Key: "1", Label: "↑↓/1-2 Select"},
		{Key: "enter", Label: "Enter Confirm"},
		{Key: "q", Label: "q/Esc Cancel"},
	}, m.activeCmd))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

func (m DeleteShelfModel) renderTypeName() string {
	style := lipgloss.NewStyle().Padding(2, 4)
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	var b strings.Builder
	b.WriteString(red.Render("Confirm Repository Deletion"))
	b.WriteString("\n\n")

	b.WriteString(red.Render(fmt.Sprintf("  This will permanently delete %s/%s", m.shelfOwner, m.shelfRepo)))
	b.WriteString("\n")
	b.WriteString(red.Render("  All books, catalog history, and release assets will be lost."))
	b.WriteString("\n\n")

	b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("Type \"%s\" to confirm:", m.shelfName)))
	b.WriteString("\n  ")
	b.WriteString(m.confirmInput.View())
	b.WriteString("\n\n")

	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	b.WriteString(tui.RenderFooterBar([]tui.ShortcutEntry{
		{Key: "enter", Label: "Enter Confirm"},
		{Key: "q", Label: "Esc Cancel"},
	}, m.activeCmd))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

func (m DeleteShelfModel) renderDone() string {
	style := lipgloss.NewStyle().Padding(2, 4)
	var b strings.Builder

	if m.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	} else if m.deleteRepo {
		b.WriteString(tui.StyleHeader.Render("Shelf Deleted"))
		b.WriteString("\n\n")
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  Removed %q from config", m.shelfName)))
		b.WriteString("\n")
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  Deleted repository %s/%s", m.shelfOwner, m.shelfRepo)))
	} else {
		b.WriteString(tui.StyleHeader.Render("Shelf Removed"))
		b.WriteString("\n\n")
		b.WriteString(tui.StyleNormal.Render(fmt.Sprintf("  Removed %q from config", m.shelfName)))
		b.WriteString("\n")
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		b.WriteString(dim.Render(fmt.Sprintf("  Repository preserved: %s/%s", m.shelfOwner, m.shelfRepo)))
		b.WriteString("\n\n")
		b.WriteString(tui.StyleHelp.Render(fmt.Sprintf("  To re-add: shelfctl init --repo %s --name %s", m.shelfRepo, m.shelfName)))
	}

	b.WriteString("\n\n")
	b.WriteString(tui.StyleHelp.Render("Press Enter to return to menu"))
	b.WriteString("\n")

	innerPadding := lipgloss.NewStyle().Padding(0, 2, 0, 1)
	return style.Render(tui.StyleBorder.Render(innerPadding.Render(b.String())))
}

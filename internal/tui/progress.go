package tui

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// ProgressReader wraps an io.Reader and reports progress through a channel.
type ProgressReader struct {
	reader      io.Reader
	total       int64
	read        int64
	progressMsg chan int64
}

// NewProgressReader creates a reader that reports progress.
func NewProgressReader(r io.Reader, total int64, progressMsg chan int64) *ProgressReader {
	return &ProgressReader{
		reader:      r,
		total:       total,
		read:        0,
		progressMsg: progressMsg,
	}
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.read += int64(n)
	if pr.progressMsg != nil {
		select {
		case pr.progressMsg <- pr.read:
		default:
		}
	}
	return n, err
}

// progressMsg is sent when progress updates
type progressMsg int64

// tickMsg is sent periodically to refresh the UI
type tickMsg time.Time

// progressModel is the Bubble Tea model for showing progress
type progressModel struct {
	progress   progress.Model
	total      int64
	current    int64
	label      string
	done       bool
	err        error
	progressCh <-chan int64
}

func (m progressModel) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		waitForProgress(m.progressCh),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func waitForProgress(ch <-chan int64) tea.Cmd {
	return func() tea.Msg {
		return progressMsg(<-ch)
	}
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Allow Ctrl+C to quit
		if msg.String() == "ctrl+c" {
			m.done = true
			return m, tea.Quit
		}

	case tickMsg:
		if m.done {
			return m, tea.Quit
		}
		return m, tickCmd()

	case progressMsg:
		m.current = int64(msg)
		if m.current >= m.total {
			m.done = true
			return m, tea.Quit
		}
		return m, waitForProgress(m.progressCh)

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 20
		if m.progress.Width > 80 {
			m.progress.Width = 80
		}
		return m, nil
	}

	return m, nil
}

func (m progressModel) View() string {
	if m.done {
		return ""
	}

	percent := 0.0
	if m.total > 0 {
		percent = float64(m.current) / float64(m.total)
	}

	// Format bytes
	currentMB := float64(m.current) / 1024 / 1024
	totalMB := float64(m.total) / 1024 / 1024

	return fmt.Sprintf(
		"%s\n%s\n%.2f MB / %.2f MB (%.0f%%)\n",
		m.label,
		m.progress.ViewAs(percent),
		currentMB,
		totalMB,
		percent*100,
	)
}

// ShowProgress displays a progress bar while performing an operation.
// The operation should wrap its io.Reader/Writer with a ProgressReader
// and send progress updates through the channel.
func ShowProgress(label string, total int64, progressCh <-chan int64) error {
	prog := progress.New(progress.WithDefaultGradient())

	m := progressModel{
		progress:   prog,
		total:      total,
		current:    0,
		label:      label,
		progressCh: progressCh,
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

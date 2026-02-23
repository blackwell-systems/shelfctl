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
	lastReport  int64 // Last byte count we reported
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

	if pr.progressMsg != nil && n > 0 {
		// Only send progress updates every 1MB (or on completion)
		// This reduces updates from ~1060 to ~17 for a 17MB file
		const updateInterval = 1024 * 1024 // 1MB
		sinceLast := pr.read - pr.lastReport
		isComplete := err == io.EOF || pr.read >= pr.total

		if sinceLast >= updateInterval || isComplete {
			select {
			case pr.progressMsg <- pr.read:
				pr.lastReport = pr.read
			default:
				// Channel full, skip this update
			}
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
	cancelled  bool
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
		// Block on channel read - UI stays alive via tickCmd
		n, ok := <-ch
		if !ok {
			// Channel closed, operation complete
			return progressMsg(-1)
		}
		return progressMsg(n)
	}
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Allow Ctrl+C to quit
		if msg.String() == "ctrl+c" {
			m.done = true
			m.cancelled = true
			return m, tea.Quit
		}

	case tickMsg:
		if m.done {
			return m, tea.Quit
		}
		// Just refresh UI on tick
		return m, tickCmd()

	case progressMsg:
		if int64(msg) == -1 {
			// Channel closed, operation complete
			m.done = true
			return m, tea.Quit
		}
		m.current = int64(msg)
		if m.current >= m.total {
			m.done = true
			return m, tea.Quit
		}
		// Wait for next progress update (tick is already running from Init)
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
// Returns error if cancelled by user (Ctrl+C).
func ShowProgress(label string, total int64, progressCh <-chan int64) error {
	prog := progress.New(progress.WithDefaultGradient())

	m := progressModel{
		progress:   prog,
		total:      total,
		current:    0,
		label:      label,
		progressCh: progressCh,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Check if user cancelled
	if fm, ok := finalModel.(progressModel); ok && fm.cancelled {
		return fmt.Errorf("cancelled by user")
	}

	return nil
}

package tui

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// TerminalImageProtocol represents the image protocol supported by the terminal
type TerminalImageProtocol int

// Terminal image protocol types
const (
	// ProtocolNone indicates no image protocol support
	ProtocolNone TerminalImageProtocol = iota
	// ProtocolKitty indicates Kitty terminal graphics protocol
	ProtocolKitty
	// ProtocolITerm2 indicates iTerm2 inline images protocol
	ProtocolITerm2
)

// DetectImageProtocol detects which terminal image protocol is supported.
// Works correctly inside tmux by checking Ghostty-specific env vars that
// survive through the tmux session (TERM_PROGRAM becomes "tmux" inside tmux,
// but GHOSTTY_RESOURCES_DIR is inherited from the parent shell).
func DetectImageProtocol() TerminalImageProtocol {
	termProgram := os.Getenv("TERM_PROGRAM")
	term := os.Getenv("TERM")

	// Kitty terminal (direct session)
	if strings.Contains(term, "kitty") {
		return ProtocolKitty
	}

	// Ghostty supports Kitty graphics protocol natively, including inside tmux.
	// TERM_PROGRAM="ghostty" in direct sessions; inside tmux it becomes "tmux",
	// but GHOSTTY_RESOURCES_DIR is always set in Ghostty-spawned shells.
	if termProgram == "ghostty" || os.Getenv("GHOSTTY_RESOURCES_DIR") != "" {
		return ProtocolKitty
	}

	// iTerm2
	if termProgram == "iTerm.app" {
		return ProtocolITerm2
	}

	return ProtocolNone
}

// RenderInlineImage renders an image inline using the terminal's protocol.
// Returns the terminal escape sequence to display the image, or empty string on error.
// Ghostty handles Kitty APC sequences transparently even inside tmux — no
// DCS passthrough wrapping required.
func RenderInlineImage(imagePath string, protocol TerminalImageProtocol) string {
	if protocol == ProtocolNone {
		return ""
	}

	data, err := os.ReadFile(imagePath)
	if err != nil {
		return ""
	}

	switch protocol {
	case ProtocolKitty:
		return renderKittyImage(data)
	case ProtocolITerm2:
		return renderITerm2Image(data)
	}
	return ""
}

// renderKittyImage encodes image data using the Kitty graphics protocol.
// - a=T: transmit and display
// - f=100: PNG/JPEG format (terminal decodes)
// - t=d: inline base64 (direct); t=f would mean a file path
func renderKittyImage(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("\x1b_Ga=T,f=100,t=d;%s\x1b\\", encoded)
}

// renderITerm2Image encodes image data using the iTerm2 inline images protocol.
func renderITerm2Image(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("\x1b]1337;File=inline=1;width=80%%:%s\x07", encoded)
}

// coverPreviewExec implements tea.ExecCommand to show a cover image outside
// the alt-screen. tea.Exec suspends the Bubble Tea TUI (exits alt-screen),
// runs Run() with raw terminal I/O, then resumes the TUI. This avoids the
// renderer's line-erasing behaviour, which would overwrite image cells.
type coverPreviewExec struct {
	imagePath string
	protocol  TerminalImageProtocol
	stdout    io.Writer
	stdin     io.Reader
}

func (c *coverPreviewExec) SetStdin(r io.Reader)  { c.stdin = r }
func (c *coverPreviewExec) SetStdout(w io.Writer) { c.stdout = w }
func (c *coverPreviewExec) SetStderr(_ io.Writer) {}

func (c *coverPreviewExec) Run() error {
	img := RenderInlineImage(c.imagePath, c.protocol)
	if img == "" || c.stdout == nil {
		return nil
	}
	fmt.Fprint(c.stdout, img)
	fmt.Fprint(c.stdout, "\n\n  Press any key to close...\n")
	if c.stdin != nil {
		buf := make([]byte, 1)
		_, _ = c.stdin.Read(buf)
	}
	return nil
}

// CoverPreviewCmd returns a tea.Cmd that suspends the TUI and shows the cover
// image at imagePath using the given protocol, then waits for a keypress.
func CoverPreviewCmd(imagePath string, protocol TerminalImageProtocol) tea.Cmd {
	return tea.Exec(&coverPreviewExec{imagePath: imagePath, protocol: protocol}, func(err error) tea.Msg {
		return nil
	})
}

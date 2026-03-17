package tui

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
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

// insideTmux reports whether the process is running inside a tmux session.
func insideTmux() bool {
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")
	return strings.HasPrefix(term, "tmux") || termProgram == "tmux" || os.Getenv("TMUX") != ""
}

// DetectImageProtocol detects which terminal image protocol is supported.
// Works correctly inside tmux by checking Ghostty/Kitty-specific env vars
// that survive through the tmux session.
func DetectImageProtocol() TerminalImageProtocol {
	termProgram := os.Getenv("TERM_PROGRAM")
	term := os.Getenv("TERM")

	// Check for Kitty terminal (direct or via TERM)
	if strings.Contains(term, "kitty") {
		return ProtocolKitty
	}

	// Check for Ghostty (supports Kitty protocol).
	// TERM_PROGRAM is set to "ghostty" in direct sessions; inside tmux it
	// becomes "tmux", so we also check GHOSTTY_RESOURCES_DIR which survives.
	if termProgram == "ghostty" || os.Getenv("GHOSTTY_RESOURCES_DIR") != "" {
		return ProtocolKitty
	}

	// Check for iTerm2
	if termProgram == "iTerm.app" {
		return ProtocolITerm2
	}

	return ProtocolNone
}

// RenderInlineImage renders an image inline using the terminal's protocol.
// Returns the terminal escape sequences to display the image, or empty string on error.
// Automatically wraps sequences in tmux DCS passthrough when inside tmux
// (requires `set -g allow-passthrough on` in tmux.conf).
func RenderInlineImage(imagePath string, protocol TerminalImageProtocol) string {
	if protocol == ProtocolNone {
		return ""
	}

	data, err := os.ReadFile(imagePath)
	if err != nil {
		return ""
	}

	var seq string
	switch protocol {
	case ProtocolKitty:
		seq = renderKittyImage(data)
	case ProtocolITerm2:
		seq = renderITerm2Image(data)
	default:
		return ""
	}

	if insideTmux() {
		seq = wrapTmuxPassthrough(seq)
	}
	return seq
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

// wrapTmuxPassthrough wraps a terminal escape sequence in tmux's DCS passthrough.
// tmux intercepts most escape sequences; passthrough forwards them to the
// outer terminal. Requires `set -g allow-passthrough on` in tmux.conf.
// Each ESC byte inside the payload must be doubled (\x1b → \x1b\x1b).
func wrapTmuxPassthrough(seq string) string {
	// Double every ESC byte inside the payload so tmux doesn't treat them as
	// the end of the DCS string.
	inner := strings.ReplaceAll(seq, "\x1b", "\x1b\x1b")
	return fmt.Sprintf("\x1bPtmux;%s\x1b\\", inner)
}

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

// DetectImageProtocol detects which terminal image protocol is supported.
func DetectImageProtocol() TerminalImageProtocol {
	termProgram := os.Getenv("TERM_PROGRAM")
	term := os.Getenv("TERM")

	// Check for Kitty terminal
	if strings.Contains(term, "kitty") {
		return ProtocolKitty
	}

	// Check for Ghostty (supports Kitty protocol)
	if termProgram == "ghostty" {
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
func RenderInlineImage(imagePath string, protocol TerminalImageProtocol) string {
	if protocol == ProtocolNone {
		return ""
	}

	// Read image file
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

// RenderInlineImageBytes renders image data inline using the terminal's protocol.
// Returns the terminal escape sequences to display the image, or empty string on error.
func RenderInlineImageBytes(data []byte, protocol TerminalImageProtocol) string {
	if protocol == ProtocolNone {
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

// renderKittyImage uses Kitty's graphics protocol
// Format: \x1b_Ga=T,f=100,t=f;<base64>\x1b\\
func renderKittyImage(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)

	// Kitty protocol:
	// - a=T: transmit image
	// - f=100: format is png/jpeg (100)
	// - t=f: data is transmitted inline
	return fmt.Sprintf("\x1b_Ga=T,f=100,t=f;%s\x1b\\", encoded)
}

// renderITerm2Image uses iTerm2's inline images protocol
// Format: \x1b]1337;File=inline=1:<base64>\x07
func renderITerm2Image(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("\x1b]1337;File=inline=1;width=30px:%s\x07", encoded)
}

package ingest

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"unicode/utf16"
)

// PDFMetadata holds extracted PDF metadata
type PDFMetadata struct {
	Title   string
	Author  string
	Subject string
}

// ExtractPDFMetadata attempts to extract basic metadata from a PDF file.
// This is a best-effort implementation using stdlib - it works for most PDFs
// but may fail on encrypted, linearized, or heavily compressed PDFs.
func ExtractPDFMetadata(path string) (*PDFMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read first 8KB looking for Info dictionary
	// Most PDFs have metadata near the beginning or in the trailer
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 8192), 8192)

	var content strings.Builder
	lineCount := 0
	for scanner.Scan() && lineCount < 500 {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
		lineCount++
	}

	text := content.String()

	// Also read the trailer (last 8KB)
	stat, err := f.Stat()
	if err == nil && stat.Size() > 8192 {
		f.Seek(-8192, 2) // Seek to last 8KB
		trailerScanner := bufio.NewScanner(f)
		trailerScanner.Buffer(make([]byte, 8192), 8192)
		var trailer strings.Builder
		for trailerScanner.Scan() {
			trailer.WriteString(trailerScanner.Text())
			trailer.WriteString("\n")
		}
		text += trailer.String()
	}

	return &PDFMetadata{
		Title:   extractField(text, "Title"),
		Author:  extractField(text, "Author"),
		Subject: extractField(text, "Subject"),
	}, nil
}

// extractField looks for /FieldName (value) or /FieldName <hex> patterns
func extractField(text, field string) string {
	// Pattern 1: /Title (Some Title Here)
	// Pattern 2: /Title <FEFF...> (UTF-16BE hex string)

	// Try parentheses format first (most common)
	pattern := `\/` + field + `\s*\(([^)]+)\)`
	re := regexp.MustCompile(pattern)
	if match := re.FindStringSubmatch(text); len(match) > 1 {
		return decodePDFString(match[1])
	}

	// Try hex format (UTF-16BE)
	hexPattern := `\/` + field + `\s*<([0-9A-Fa-f]+)>`
	reHex := regexp.MustCompile(hexPattern)
	if match := reHex.FindStringSubmatch(text); len(match) > 1 {
		return decodeHexString(match[1])
	}

	return ""
}

// decodePDFString handles basic PDF string escaping
func decodePDFString(s string) string {
	// Handle common escape sequences
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\r`, "\r")
	s = strings.ReplaceAll(s, `\t`, "\t")
	s = strings.ReplaceAll(s, `\(`, "(")
	s = strings.ReplaceAll(s, `\)`, ")")
	s = strings.ReplaceAll(s, `\\`, "\\")

	// Trim whitespace
	return strings.TrimSpace(s)
}

// decodeHexString decodes UTF-16BE hex strings (common in PDFs)
func decodeHexString(hex string) string {
	// Remove FEFF BOM if present
	hex = strings.TrimPrefix(hex, "FEFF")
	hex = strings.TrimPrefix(hex, "feff")

	// Convert hex to bytes
	if len(hex)%2 != 0 {
		return "" // Invalid hex
	}

	rawBytes := make([]byte, len(hex)/2)
	for i := 0; i < len(rawBytes); i++ {
		rawBytes[i] = hexValue(hex[i*2])<<4 | hexValue(hex[i*2+1])
	}

	// Try to decode as UTF-16BE
	if len(rawBytes)%2 != 0 {
		// Odd number of bytes - might be ASCII
		return string(rawBytes)
	}

	u16 := make([]uint16, len(rawBytes)/2)
	for i := 0; i < len(u16); i++ {
		u16[i] = uint16(rawBytes[i*2])<<8 | uint16(rawBytes[i*2+1])
	}

	return string(utf16.Decode(u16))
}

func hexValue(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

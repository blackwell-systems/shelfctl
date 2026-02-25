package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractPDFMetadata(t *testing.T) {
	// Create a minimal PDF with metadata for testing
	// This is a valid minimal PDF with an Info dictionary
	minimalPDF := `%PDF-1.4
1 0 obj
<<
/Type /Catalog
/Pages 2 0 R
>>
endobj
2 0 obj
<<
/Type /Pages
/Kids [3 0 R]
/Count 1
>>
endobj
3 0 obj
<<
/Type /Page
/Parent 2 0 R
/MediaBox [0 0 612 792]
>>
endobj
4 0 obj
<<
/Title (Test Document Title)
/Author (John Doe)
/Subject (Test Subject)
>>
endobj
xref
0 5
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
0000000190 00000 n
trailer
<<
/Size 5
/Root 1 0 R
/Info 4 0 R
>>
startxref
280
%%EOF`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.pdf")

	if err := os.WriteFile(testFile, []byte(minimalPDF), 0600); err != nil {
		t.Fatalf("Failed to create test PDF: %v", err)
	}

	meta, err := ExtractPDFMetadata(testFile)
	if err != nil {
		t.Fatalf("ExtractPDFMetadata failed: %v", err)
	}

	if meta.Title != "Test Document Title" {
		t.Errorf("Expected title 'Test Document Title', got %q", meta.Title)
	}

	if meta.Author != "John Doe" {
		t.Errorf("Expected author 'John Doe', got %q", meta.Author)
	}

	if meta.Subject != "Test Subject" {
		t.Errorf("Expected subject 'Test Subject', got %q", meta.Subject)
	}
}

func TestExtractPDFMetadata_NoMetadata(t *testing.T) {
	// PDF without Info dictionary
	minimalPDF := `%PDF-1.4
1 0 obj
<<
/Type /Catalog
>>
endobj
xref
0 2
0000000000 65535 f
0000000009 00000 n
trailer
<<
/Size 2
/Root 1 0 R
>>
startxref
50
%%EOF`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.pdf")

	if err := os.WriteFile(testFile, []byte(minimalPDF), 0600); err != nil {
		t.Fatalf("Failed to create test PDF: %v", err)
	}

	meta, err := ExtractPDFMetadata(testFile)
	if err != nil {
		t.Fatalf("ExtractPDFMetadata failed: %v", err)
	}

	// Should return empty metadata without error
	if meta.Title != "" || meta.Author != "" || meta.Subject != "" {
		t.Errorf("Expected empty metadata, got Title=%q Author=%q Subject=%q",
			meta.Title, meta.Author, meta.Subject)
	}
}

func TestDecodePDFString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Simple Title", "Simple Title"},
		{"Title\\nWith\\nNewlines", "Title\nWith\nNewlines"},
		{"Title with \\(parens\\)", "Title with (parens)"},
		{"Path\\\\with\\\\backslash", "Path\\with\\backslash"},
		{"  Spaces  ", "Spaces"},
	}

	for _, tt := range tests {
		got := decodePDFString(tt.input)
		if got != tt.want {
			t.Errorf("decodePDFString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- sanitizeForTerminal ---

func TestSanitizeForTerminal_CurlyQuotes(t *testing.T) {
	got := sanitizeForTerminal("\u201CHello\u201D")
	if got != `"Hello"` {
		t.Errorf("curly double quotes: got %q, want %q", got, `"Hello"`)
	}
}

func TestSanitizeForTerminal_SmartApostrophe(t *testing.T) {
	got := sanitizeForTerminal("it\u2019s")
	if got != "it's" {
		t.Errorf("smart apostrophe: got %q, want %q", got, "it's")
	}
}

func TestSanitizeForTerminal_Dashes(t *testing.T) {
	got := sanitizeForTerminal("a\u2013b\u2014c")
	if got != "a-b--c" {
		t.Errorf("dashes: got %q, want %q", got, "a-b--c")
	}
}

func TestSanitizeForTerminal_Ellipsis(t *testing.T) {
	got := sanitizeForTerminal("wait\u2026")
	if got != "wait..." {
		t.Errorf("ellipsis: got %q, want %q", got, "wait...")
	}
}

func TestSanitizeForTerminal_Nbsp(t *testing.T) {
	got := sanitizeForTerminal("hello\u00A0world")
	if got != "hello world" {
		t.Errorf("nbsp: got %q, want %q", got, "hello world")
	}
}

func TestSanitizeForTerminal_Bullet(t *testing.T) {
	got := sanitizeForTerminal("\u2022 item")
	if got != "* item" {
		t.Errorf("bullet: got %q, want %q", got, "* item")
	}
}

func TestSanitizeForTerminal_Guillemets(t *testing.T) {
	got := sanitizeForTerminal("\u00ABquote\u00BB")
	if got != "<<quote>>" {
		t.Errorf("guillemets: got %q, want %q", got, "<<quote>>")
	}
}

func TestSanitizeForTerminal_PlainASCII(t *testing.T) {
	got := sanitizeForTerminal("hello world")
	if got != "hello world" {
		t.Errorf("plain ASCII should pass through: got %q", got)
	}
}

// --- hexValue ---

func TestHexValue_Digits(t *testing.T) {
	for i := byte('0'); i <= '9'; i++ {
		got := hexValue(i)
		want := i - '0'
		if got != want {
			t.Errorf("hexValue(%q) = %d, want %d", i, got, want)
		}
	}
}

func TestHexValue_Lowercase(t *testing.T) {
	cases := map[byte]byte{'a': 10, 'b': 11, 'f': 15}
	for c, want := range cases {
		got := hexValue(c)
		if got != want {
			t.Errorf("hexValue(%q) = %d, want %d", c, got, want)
		}
	}
}

func TestHexValue_Uppercase(t *testing.T) {
	cases := map[byte]byte{'A': 10, 'B': 11, 'F': 15}
	for c, want := range cases {
		got := hexValue(c)
		if got != want {
			t.Errorf("hexValue(%q) = %d, want %d", c, got, want)
		}
	}
}

func TestHexValue_Invalid(t *testing.T) {
	got := hexValue('G')
	if got != 0 {
		t.Errorf("hexValue('G') = %d, want 0", got)
	}
}

// --- decodeHexString ---

func TestDecodeHexString_UTF16BE(t *testing.T) {
	// "Hello" in UTF-16BE hex (with FEFF BOM stripped by caller pattern)
	// H=0048 e=0065 l=006C l=006C o=006F
	got := decodeHexString("00480065006C006C006F")
	if got != "Hello" {
		t.Errorf("UTF-16BE: got %q, want %q", got, "Hello")
	}
}

func TestDecodeHexString_WithBOM(t *testing.T) {
	// FEFF prefix should be stripped
	got := decodeHexString("FEFF00480069")
	if got != "Hi" {
		t.Errorf("with BOM: got %q, want %q", got, "Hi")
	}
}

func TestDecodeHexString_OddLength(t *testing.T) {
	got := decodeHexString("ABC")
	if got != "" {
		t.Errorf("odd length: got %q, want empty", got)
	}
}

func TestDecodeHexString_Empty(t *testing.T) {
	got := decodeHexString("")
	if got != "" {
		t.Errorf("empty: got %q, want empty", got)
	}
}

// --- extractField ---

func TestExtractField_Parentheses(t *testing.T) {
	text := `/Title (My Great Book)`
	got := extractField(text, "Title")
	if got != "My Great Book" {
		t.Errorf("extractField parentheses: got %q, want %q", got, "My Great Book")
	}
}

func TestExtractField_HexFormat(t *testing.T) {
	// "Hi" in UTF-16BE: 0048=H, 0069=i
	text := `/Title <00480069>`
	got := extractField(text, "Title")
	if got != "Hi" {
		t.Errorf("extractField hex: got %q, want %q", got, "Hi")
	}
}

func TestExtractField_NotFound(t *testing.T) {
	text := `/Author (Someone)`
	got := extractField(text, "Title")
	if got != "" {
		t.Errorf("extractField not found: got %q, want empty", got)
	}
}

// --- decodePDFString additional tests ---

func TestDecodePDFString_Escapes(t *testing.T) {
	got := decodePDFString(`Hello\nWorld`)
	if got != "Hello\nWorld" {
		t.Errorf("escape \\n: got %q", got)
	}
}

func TestDecodePDFString_ParenEscape(t *testing.T) {
	got := decodePDFString(`\(parens\)`)
	if got != "(parens)" {
		t.Errorf("escape parens: got %q, want %q", got, "(parens)")
	}
}

func TestDecodePDFString_PlainASCII(t *testing.T) {
	got := decodePDFString("Hello World")
	if got != "Hello World" {
		t.Errorf("plain ASCII: got %q, want %q", got, "Hello World")
	}
}

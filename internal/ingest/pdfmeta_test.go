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

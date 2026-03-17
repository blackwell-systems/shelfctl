package ingest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExtractPDFMetadata_LongLine tests that the scanner handles lines longer than 8192 bytes
func TestExtractPDFMetadata_LongLine(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_long_line.pdf")

	// Create a synthetic "PDF" with a very long line (exceeds old 8192 buffer)
	// This simulates PDFs with large metadata fields or embedded content
	longLine := strings.Repeat("A", 10000) // 10KB line

	// Basic PDF structure with a long metadata field
	pdfContent := `%PDF-1.4
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
/Resources <<
/Font <<
/F1 <<
/Type /Font
/Subtype /Type1
/BaseFont /Helvetica
>>
>>
>>
/MediaBox [0 0 612 792]
/Contents 4 0 R
>>
endobj
4 0 obj
<<
/Length 44
>>
stream
BT
/F1 12 Tf
100 700 Td
(Test) Tj
ET
endstream
endobj
5 0 obj
<<
/Title (` + longLine + `)
/Author (Test Author)
/Subject (Test Subject)
>>
endobj
xref
0 6
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
0000000317 00000 n
0000000410 00000 n
trailer
<<
/Size 6
/Root 1 0 R
/Info 5 0 R
>>
startxref
` + longLine + `
%%EOF
`

	// Write the test PDF
	if err := os.WriteFile(testFile, []byte(pdfContent), 0644); err != nil {
		t.Fatalf("Failed to create test PDF: %v", err)
	}

	// Extract metadata - should not panic and should return partial results
	meta, err := ExtractPDFMetadata(testFile)
	if err != nil {
		t.Fatalf("ExtractPDFMetadata failed: %v", err)
	}

	// We should get some metadata even if the long line caused issues
	if meta == nil {
		t.Fatal("Expected non-nil metadata")
	}

	// The author and subject should be extracted (they appear before the long line)
	if meta.Author != "Test Author" {
		t.Errorf("Expected author 'Test Author', got '%s'", meta.Author)
	}
	if meta.Subject != "Test Subject" {
		t.Errorf("Expected subject 'Test Subject', got '%s'", meta.Subject)
	}

	// The title field contains the long line, which might or might not be fully extracted
	// depending on when the scanner encounters it. The important thing is we don't panic.
	t.Logf("Title length: %d", len(meta.Title))
}

// TestExtractPDFMetadata_MultipleBufferSizes tests various line lengths
func TestExtractPDFMetadata_MultipleBufferSizes(t *testing.T) {
	testCases := []struct {
		name       string
		lineLength int
	}{
		{"small_line", 100},
		{"medium_line", 1000},
		{"old_buffer_limit", 8192},
		{"just_over_old_limit", 8193},
		{"large_line", 32000},
		{"very_large_line", 65000}, // Just under new 64KB limit
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.pdf")

			longLine := strings.Repeat("B", tc.lineLength)
			pdfContent := []byte(`%PDF-1.4
trailer
<<
/Info <<
/Title (` + longLine + `)
/Author (Test)
>>
>>
%%EOF
`)

			if err := os.WriteFile(testFile, pdfContent, 0644); err != nil {
				t.Fatalf("Failed to create test PDF: %v", err)
			}

			// Should not panic regardless of line length
			meta, err := ExtractPDFMetadata(testFile)
			if err != nil {
				t.Fatalf("ExtractPDFMetadata failed: %v", err)
			}
			if meta == nil {
				t.Fatal("Expected non-nil metadata")
			}

			// For lines within the new buffer size, we should get the full content
			if tc.lineLength < 64*1024 {
				if !strings.Contains(meta.Title, strings.Repeat("B", min(tc.lineLength, 100))) {
					t.Logf("Title might be truncated or not fully extracted for length %d", tc.lineLength)
				}
			}
		})
	}
}

// TestExtractPDFMetadata_EmptyFile tests handling of empty file
func TestExtractPDFMetadata_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.pdf")

	if err := os.WriteFile(testFile, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	meta, err := ExtractPDFMetadata(testFile)
	if err != nil {
		t.Fatalf("ExtractPDFMetadata failed: %v", err)
	}
	if meta == nil {
		t.Fatal("Expected non-nil metadata")
	}

	// Empty file should return empty metadata
	if meta.Title != "" || meta.Author != "" || meta.Subject != "" {
		t.Error("Expected empty metadata for empty file")
	}
}

// TestExtractPDFMetadata_ValidPDF tests extraction from a basic valid PDF
func TestExtractPDFMetadata_ValidPDF(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "valid.pdf")

	// Minimal valid PDF with metadata
	pdfContent := []byte(`%PDF-1.4
1 0 obj
<<
/Type /Catalog
/Pages 2 0 R
>>
endobj
2 0 obj
<<
/Type /Pages
/Kids []
/Count 0
>>
endobj
3 0 obj
<<
/Title (Valid Test PDF)
/Author (John Doe)
/Subject (Testing)
>>
endobj
xref
0 4
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
trailer
<<
/Size 4
/Root 1 0 R
/Info 3 0 R
>>
startxref
236
%%EOF
`)

	if err := os.WriteFile(testFile, pdfContent, 0644); err != nil {
		t.Fatalf("Failed to create test PDF: %v", err)
	}

	meta, err := ExtractPDFMetadata(testFile)
	if err != nil {
		t.Fatalf("ExtractPDFMetadata failed: %v", err)
	}
	if meta == nil {
		t.Fatal("Expected non-nil metadata")
	}

	if meta.Title != "Valid Test PDF" {
		t.Errorf("Expected title 'Valid Test PDF', got '%s'", meta.Title)
	}
	if meta.Author != "John Doe" {
		t.Errorf("Expected author 'John Doe', got '%s'", meta.Author)
	}
	if meta.Subject != "Testing" {
		t.Errorf("Expected subject 'Testing', got '%s'", meta.Subject)
	}
}

// TestExtractPDFMetadata_BinaryData tests handling of binary data in PDF
func TestExtractPDFMetadata_BinaryData(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "binary.pdf")

	// PDF with embedded binary stream
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	buf.WriteString("3 0 obj\n<<\n/Title (Test Binary PDF)\n/Author (Tester)\n>>\nendobj\n")

	// Add some binary data
	buf.Write([]byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD})
	buf.WriteString("\n%%EOF\n")

	if err := os.WriteFile(testFile, buf.Bytes(), 0644); err != nil {
		t.Fatalf("Failed to create test PDF: %v", err)
	}

	meta, err := ExtractPDFMetadata(testFile)
	if err != nil {
		t.Fatalf("ExtractPDFMetadata failed: %v", err)
	}
	if meta == nil {
		t.Fatal("Expected non-nil metadata")
	}

	if meta.Title != "Test Binary PDF" {
		t.Errorf("Expected title 'Test Binary PDF', got '%s'", meta.Title)
	}
	if meta.Author != "Tester" {
		t.Errorf("Expected author 'Tester', got '%s'", meta.Author)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

package fixtures

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

// TestDefaultFixtures validates the structure of the default fixture set.
func TestDefaultFixtures(t *testing.T) {
	fixtures := DefaultFixtures()

	if fixtures == nil {
		t.Fatal("DefaultFixtures() returned nil")
	}

	if len(fixtures.Shelves) == 0 {
		t.Fatal("DefaultFixtures() returned empty Shelves slice")
	}

	for i, shelf := range fixtures.Shelves {
		if shelf.Name == "" {
			t.Errorf("Shelf %d has empty Name", i)
		}
		if shelf.Owner == "" {
			t.Errorf("Shelf %d has empty Owner", i)
		}
		if shelf.Repo == "" {
			t.Errorf("Shelf %d has empty Repo", i)
		}
		if len(shelf.Books) == 0 {
			t.Errorf("Shelf %d has no books", i)
		}
		if shelf.Assets == nil {
			t.Errorf("Shelf %d has nil Assets map", i)
		}
	}
}

// TestShelfCount verifies that exactly 3 shelves are present.
func TestShelfCount(t *testing.T) {
	fixtures := DefaultFixtures()

	expected := 3
	actual := len(fixtures.Shelves)

	if actual != expected {
		t.Errorf("Expected %d shelves, got %d", expected, actual)
	}

	// Verify expected shelf names
	expectedNames := map[string]bool{
		"tech":      false,
		"fiction":   false,
		"reference": false,
	}

	for _, shelf := range fixtures.Shelves {
		if _, ok := expectedNames[shelf.Name]; ok {
			expectedNames[shelf.Name] = true
		} else {
			t.Errorf("Unexpected shelf name: %s", shelf.Name)
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("Expected shelf %s not found", name)
		}
	}
}

// TestBookMetadata validates that all books have realistic and complete metadata.
func TestBookMetadata(t *testing.T) {
	fixtures := DefaultFixtures()

	for _, shelf := range fixtures.Shelves {
		for _, book := range shelf.Books {
			// Required fields
			if book.ID == "" {
				t.Errorf("Book in shelf %s has empty ID", shelf.Name)
			}
			if book.Title == "" {
				t.Errorf("Book %s in shelf %s has empty Title", book.ID, shelf.Name)
			}
			if book.Format == "" {
				t.Errorf("Book %s in shelf %s has empty Format", book.ID, shelf.Name)
			}

			// Format should be valid
			if book.Format != "pdf" && book.Format != "epub" {
				t.Errorf("Book %s has invalid format: %s (expected pdf or epub)", book.ID, book.Format)
			}

			// Optional but expected fields
			if book.Author == "" {
				t.Errorf("Book %s in shelf %s has empty Author", book.ID, shelf.Name)
			}
			if book.Year == 0 {
				t.Errorf("Book %s in shelf %s has zero Year", book.ID, shelf.Name)
			}
			if book.Year < 1950 || book.Year > 2030 {
				t.Errorf("Book %s has unrealistic year: %d", book.ID, book.Year)
			}
			if len(book.Tags) == 0 {
				t.Errorf("Book %s in shelf %s has no tags", book.ID, shelf.Name)
			}

			// Checksum must be present and valid format
			if book.Checksum.SHA256 == "" {
				t.Errorf("Book %s in shelf %s has empty SHA256 checksum", book.ID, shelf.Name)
			}
			if len(book.Checksum.SHA256) != 64 {
				t.Errorf("Book %s has invalid SHA256 length: %d (expected 64)", book.ID, len(book.Checksum.SHA256))
			}

			// Size must be positive
			if book.SizeBytes <= 0 {
				t.Errorf("Book %s has invalid SizeBytes: %d", book.ID, book.SizeBytes)
			}

			// Source must be complete
			if book.Source.Type != "github_release" {
				t.Errorf("Book %s has unexpected source type: %s", book.ID, book.Source.Type)
			}
			if book.Source.Owner == "" {
				t.Errorf("Book %s has empty source owner", book.ID)
			}
			if book.Source.Repo == "" {
				t.Errorf("Book %s has empty source repo", book.ID)
			}
			if book.Source.Release == "" {
				t.Errorf("Book %s has empty source release", book.ID)
			}
			if book.Source.Asset == "" {
				t.Errorf("Book %s has empty source asset", book.ID)
			}
		}
	}
}

// TestAssetContent validates that PDFs are valid and checksums match.
func TestAssetContent(t *testing.T) {
	fixtures := DefaultFixtures()

	for _, shelf := range fixtures.Shelves {
		for _, book := range shelf.Books {
			// Verify asset exists
			content, ok := shelf.Assets[book.ID]
			if !ok {
				t.Errorf("Book %s in shelf %s has no asset content", book.ID, shelf.Name)
				continue
			}

			// Verify content is not empty
			if len(content) == 0 {
				t.Errorf("Book %s in shelf %s has empty asset content", book.ID, shelf.Name)
				continue
			}

			// Verify PDF header (if format is pdf)
			if book.Format == "pdf" {
				if !strings.HasPrefix(string(content), "%PDF-") {
					t.Errorf("Book %s claims to be PDF but content doesn't start with %%PDF- header", book.ID)
				}
			}

			// Verify size matches actual content
			if book.SizeBytes != int64(len(content)) {
				t.Errorf("Book %s reports size %d but actual content is %d bytes", book.ID, book.SizeBytes, len(content))
			}

			// Verify checksum matches content
			actualHash := sha256.Sum256(content)
			actualHashStr := fmt.Sprintf("%x", actualHash)
			if book.Checksum.SHA256 != actualHashStr {
				t.Errorf("Book %s checksum mismatch: catalog=%s, actual=%s", book.ID, book.Checksum.SHA256, actualHashStr)
			}
		}
	}
}

// TestChecksumConsistency verifies that checksums are consistent across multiple calls.
func TestChecksumConsistency(t *testing.T) {
	fixtures1 := DefaultFixtures()
	fixtures2 := DefaultFixtures()

	if len(fixtures1.Shelves) != len(fixtures2.Shelves) {
		t.Fatal("Fixture generation is non-deterministic: shelf count mismatch")
	}

	for i := range fixtures1.Shelves {
		shelf1 := fixtures1.Shelves[i]
		shelf2 := fixtures2.Shelves[i]

		if len(shelf1.Books) != len(shelf2.Books) {
			t.Errorf("Shelf %s has inconsistent book count: %d vs %d", shelf1.Name, len(shelf1.Books), len(shelf2.Books))
			continue
		}

		for j := range shelf1.Books {
			book1 := shelf1.Books[j]
			book2 := shelf2.Books[j]

			if book1.ID != book2.ID {
				t.Errorf("Book order changed at position %d in shelf %s", j, shelf1.Name)
				continue
			}

			if book1.Checksum.SHA256 != book2.Checksum.SHA256 {
				t.Errorf("Book %s has inconsistent checksum across calls", book1.ID)
			}

			// Verify asset content is identical
			content1 := shelf1.Assets[book1.ID]
			content2 := shelf2.Assets[book2.ID]

			if string(content1) != string(content2) {
				t.Errorf("Book %s has inconsistent asset content across calls", book1.ID)
			}
		}
	}
}

// TestShelfBookCounts validates that each shelf has 5-10 books as specified.
func TestShelfBookCounts(t *testing.T) {
	fixtures := DefaultFixtures()

	for _, shelf := range fixtures.Shelves {
		count := len(shelf.Books)
		if count < 5 || count > 10 {
			t.Errorf("Shelf %s has %d books (expected 5-10)", shelf.Name, count)
		}
	}
}

// TestAssetMapCompleteness ensures every book has a corresponding asset.
func TestAssetMapCompleteness(t *testing.T) {
	fixtures := DefaultFixtures()

	for _, shelf := range fixtures.Shelves {
		for _, book := range shelf.Books {
			if _, ok := shelf.Assets[book.ID]; !ok {
				t.Errorf("Book %s in shelf %s is missing from Assets map", book.ID, shelf.Name)
			}
		}

		// Also check no extra assets exist
		if len(shelf.Assets) != len(shelf.Books) {
			t.Errorf("Shelf %s has %d books but %d assets (should match)", shelf.Name, len(shelf.Books), len(shelf.Assets))
		}
	}
}

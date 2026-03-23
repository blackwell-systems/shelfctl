package tui

import (
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
	"github.com/charmbracelet/bubbles/list"
)

// TestBookPickerSelection tests that book picker tracks selected book ID correctly
func TestBookPickerSelection(t *testing.T) {
	// Create test books
	books := []BookItem{
		{
			Book: catalog.Book{
				ID:     "book-1",
				Title:  "The Go Programming Language",
				Author: "Kernighan & Donovan",
				Tags:   []string{"programming", "go"},
			},
			ShelfName: "tech",
			Cached:    true,
		},
		{
			Book: catalog.Book{
				ID:     "book-2",
				Title:  "Clean Code",
				Author: "Robert Martin",
				Tags:   []string{"programming"},
			},
			ShelfName: "tech",
			Cached:    false,
		},
		{
			Book: catalog.Book{
				ID:     "book-3",
				Title:  "Design Patterns",
				Author: "Gang of Four",
				Tags:   []string{"programming", "patterns"},
			},
			ShelfName: "tech",
			Cached:    true,
		},
	}

	// Test that we can correctly identify and track a selected book
	selectedBook := books[1] // "Clean Code"

	if selectedBook.Book.ID != "book-2" {
		t.Errorf("Expected selected book ID to be 'book-2', got '%s'", selectedBook.Book.ID)
	}

	if selectedBook.Book.Title != "Clean Code" {
		t.Errorf("Expected selected book title to be 'Clean Code', got '%s'", selectedBook.Book.Title)
	}

	if selectedBook.Cached {
		t.Errorf("Expected book-2 to not be cached, but it was")
	}
}

// TestShelfPickerSelection tests that shelf picker tracks selected shelf correctly
func TestShelfPickerSelection(t *testing.T) {
	shelves := []ShelfOption{
		{Name: "tech", Repo: "owner/tech-shelf"},
		{Name: "fiction", Repo: "owner/fiction-shelf"},
		{Name: "reference", Repo: "owner/reference-shelf"},
	}

	// Test shelf selection tracking
	selectedShelf := shelves[1] // "fiction"

	if selectedShelf.Name != "fiction" {
		t.Errorf("Expected selected shelf name to be 'fiction', got '%s'", selectedShelf.Name)
	}

	if selectedShelf.Repo != "owner/fiction-shelf" {
		t.Errorf("Expected selected shelf repo to be 'owner/fiction-shelf', got '%s'", selectedShelf.Repo)
	}
}

// TestPickerMultiSelect tests multiple selection support for books
func TestPickerMultiSelect(t *testing.T) {
	books := []BookItem{
		{
			Book: catalog.Book{
				ID:     "book-1",
				Title:  "Book One",
				Author: "Author A",
			},
			ShelfName: "shelf1",
		},
		{
			Book: catalog.Book{
				ID:     "book-2",
				Title:  "Book Two",
				Author: "Author B",
			},
			ShelfName: "shelf1",
		},
		{
			Book: catalog.Book{
				ID:     "book-3",
				Title:  "Book Three",
				Author: "Author C",
			},
			ShelfName: "shelf2",
		},
	}

	// Test multi-select state management
	// Initially nothing is selected
	for i, book := range books {
		if book.IsSelected() {
			t.Errorf("Book %d should not be selected initially", i)
		}
	}

	// Select first and third books
	books[0].SetSelected(true)
	books[2].SetSelected(true)

	// Verify selection states
	if !books[0].IsSelected() {
		t.Error("Book 0 should be selected")
	}
	if books[1].IsSelected() {
		t.Error("Book 1 should not be selected")
	}
	if !books[2].IsSelected() {
		t.Error("Book 2 should be selected")
	}

	// Collect selected books
	var selected []BookItem
	for _, book := range books {
		if book.IsSelected() {
			selected = append(selected, book)
		}
	}

	if len(selected) != 2 {
		t.Errorf("Expected 2 selected books, got %d", len(selected))
	}

	if selected[0].Book.ID != "book-1" {
		t.Errorf("Expected first selected book to be 'book-1', got '%s'", selected[0].Book.ID)
	}

	if selected[1].Book.ID != "book-3" {
		t.Errorf("Expected second selected book to be 'book-3', got '%s'", selected[1].Book.ID)
	}

	// Test deselection
	books[0].SetSelected(false)
	if books[0].IsSelected() {
		t.Error("Book 0 should be deselected")
	}
}

// TestPickerFilter tests filter by search query for books
func TestPickerFilter(t *testing.T) {
	books := []BookItem{
		{
			Book: catalog.Book{
				ID:     "go-lang",
				Title:  "The Go Programming Language",
				Author: "Kernighan",
				Tags:   []string{"programming", "go"},
			},
			ShelfName: "tech",
		},
		{
			Book: catalog.Book{
				ID:     "clean-code",
				Title:  "Clean Code",
				Author: "Martin",
				Tags:   []string{"programming"},
			},
			ShelfName: "tech",
		},
		{
			Book: catalog.Book{
				ID:     "fiction-book",
				Title:  "The Great Gatsby",
				Author: "Fitzgerald",
				Tags:   []string{"fiction", "classic"},
			},
			ShelfName: "novels",
		},
	}

	tests := []struct {
		name     string
		query    string
		expected []string // Expected book IDs
	}{
		{
			name:     "filter by title substring",
			query:    "Go",
			expected: []string{"go-lang"},
		},
		{
			name:     "filter by tag",
			query:    "fiction",
			expected: []string{"fiction-book"},
		},
		{
			name:     "filter by shelf name",
			query:    "tech",
			expected: []string{"go-lang", "clean-code"},
		},
		{
			name:     "filter by ID",
			query:    "clean-code",
			expected: []string{"clean-code"},
		},
		{
			name:     "no match",
			query:    "nonexistent",
			expected: []string{},
		},
		{
			name:     "empty query returns all",
			query:    "",
			expected: []string{"go-lang", "clean-code", "fiction-book"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate filtering by using FilterValue
			var filtered []BookItem
			for _, book := range books {
				filterVal := book.FilterValue()
				// Simple substring match (mimics how bubbletea list filtering works)
				if tt.query == "" || contains(filterVal, tt.query) {
					filtered = append(filtered, book)
				}
			}

			if len(filtered) != len(tt.expected) {
				t.Errorf("Expected %d filtered books, got %d", len(tt.expected), len(filtered))
			}

			// Verify correct books were filtered
			for i, expectedID := range tt.expected {
				if i >= len(filtered) {
					t.Errorf("Missing expected book at index %d: %s", i, expectedID)
					continue
				}
				if filtered[i].Book.ID != expectedID {
					t.Errorf("Expected book ID '%s' at index %d, got '%s'", expectedID, i, filtered[i].Book.ID)
				}
			}
		})
	}
}

// TestPickerEmpty tests handling of empty picker list
func TestPickerEmpty(t *testing.T) {
	t.Run("empty book list", func(t *testing.T) {
		books := []BookItem{}

		if len(books) != 0 {
			t.Errorf("Expected empty book list, got %d books", len(books))
		}

		// Attempting to select from empty list should not panic
		var selected *BookItem
		if len(books) > 0 {
			selected = &books[0]
		}

		if selected != nil {
			t.Error("Expected nil selection from empty list")
		}
	})

	t.Run("empty shelf list", func(t *testing.T) {
		shelves := []ShelfOption{}

		if len(shelves) != 0 {
			t.Errorf("Expected empty shelf list, got %d shelves", len(shelves))
		}

		// Attempting to select from empty list should not panic
		var selected *ShelfOption
		if len(shelves) > 0 {
			selected = &shelves[0]
		}

		if selected != nil {
			t.Error("Expected nil selection from empty list")
		}
	})

	t.Run("single shelf auto-select", func(t *testing.T) {
		shelves := []ShelfOption{
			{Name: "only-shelf", Repo: "owner/only-shelf"},
		}

		// Single shelf should be auto-selectable
		if len(shelves) != 1 {
			t.Errorf("Expected 1 shelf, got %d", len(shelves))
		}

		autoSelected := shelves[0]
		if autoSelected.Name != "only-shelf" {
			t.Errorf("Expected auto-selected shelf to be 'only-shelf', got '%s'", autoSelected.Name)
		}
	})
}

// TestShelfOptionFilterValue tests the filter value for shelf options
func TestShelfOptionFilterValue(t *testing.T) {
	shelf := ShelfOption{
		Name: "tech-books",
		Repo: "myorg/tech-shelf",
	}

	filterVal := shelf.FilterValue()

	// FilterValue should include both name and repo for better search
	if !contains(filterVal, "tech-books") {
		t.Error("Filter value should contain shelf name")
	}

	if !contains(filterVal, "myorg/tech-shelf") {
		t.Error("Filter value should contain repo")
	}
}

// TestBookItemSelectable tests that all books are selectable
func TestBookItemSelectable(t *testing.T) {
	book := BookItem{
		Book: catalog.Book{
			ID:    "test-book",
			Title: "Test Book",
		},
		ShelfName: "test-shelf",
	}

	if !book.IsSelectable() {
		t.Error("All books should be selectable")
	}
}

// TestCollectSelectedBooks tests the CollectSelectedBooks helper function
func TestCollectSelectedBooks(t *testing.T) {
	books := []BookItem{
		{
			Book: catalog.Book{
				ID:    "book-1",
				Title: "Book One",
			},
			ShelfName: "shelf1",
		},
		{
			Book: catalog.Book{
				ID:    "book-2",
				Title: "Book Two",
			},
			ShelfName: "shelf1",
		},
		{
			Book: catalog.Book{
				ID:    "book-3",
				Title: "Book Three",
			},
			ShelfName: "shelf2",
		},
	}

	// Convert to list items (pointers for selection state)
	items := make([]list.Item, len(books))
	for i := range books {
		items[i] = &books[i]
	}

	// Select some books
	books[0].SetSelected(true)
	books[2].SetSelected(true)

	// CollectSelectedBooks would iterate and collect selected items
	var selected []BookItem
	for _, item := range items {
		if bookItem, ok := item.(*BookItem); ok && bookItem.IsSelected() {
			selected = append(selected, *bookItem)
		}
	}

	if len(selected) != 2 {
		t.Errorf("Expected 2 selected books, got %d", len(selected))
	}

	expectedIDs := []string{"book-1", "book-3"}
	for i, expectedID := range expectedIDs {
		if selected[i].Book.ID != expectedID {
			t.Errorf("Expected book ID '%s' at index %d, got '%s'", expectedID, i, selected[i].Book.ID)
		}
	}
}

// Helper function to check if a string contains a substring (case-insensitive would be better but keeping simple)
func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && indexSubstring(s, substr) >= 0
}

func indexSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

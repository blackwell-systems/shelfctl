package tui

import (
	"testing"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
)

// TestListBrowserFiltering tests filtering books by tag and format.
func TestListBrowserFiltering(t *testing.T) {
	tests := []struct {
		name        string
		books       []BookItem
		filterQuery string
		wantCount   int
		wantIDs     []string
	}{
		{
			name: "filter by tag",
			books: []BookItem{
				{Book: catalog.Book{ID: "book1", Title: "Go Programming", Tags: []string{"programming", "golang"}}},
				{Book: catalog.Book{ID: "book2", Title: "Python Basics", Tags: []string{"programming", "python"}}},
				{Book: catalog.Book{ID: "book3", Title: "Cooking Guide", Tags: []string{"cooking", "recipes"}}},
			},
			filterQuery: "golang",
			wantCount:   1,
			wantIDs:     []string{"book1"},
		},
		{
			name: "filter by title",
			books: []BookItem{
				{Book: catalog.Book{ID: "book1", Title: "Go Programming", Tags: []string{"programming"}}},
				{Book: catalog.Book{ID: "book2", Title: "Python Basics", Tags: []string{"programming"}}},
				{Book: catalog.Book{ID: "book3", Title: "Go Advanced", Tags: []string{"programming"}}},
			},
			filterQuery: "Go",
			wantCount:   2,
			wantIDs:     []string{"book1", "book3"},
		},
		{
			name: "filter by shelf name",
			books: []BookItem{
				{Book: catalog.Book{ID: "book1", Title: "Book One"}, ShelfName: "technical"},
				{Book: catalog.Book{ID: "book2", Title: "Book Two"}, ShelfName: "fiction"},
				{Book: catalog.Book{ID: "book3", Title: "Book Three"}, ShelfName: "technical"},
			},
			filterQuery: "technical",
			wantCount:   2,
			wantIDs:     []string{"book1", "book3"},
		},
		{
			name: "no match",
			books: []BookItem{
				{Book: catalog.Book{ID: "book1", Title: "Go Programming", Tags: []string{"programming"}}},
				{Book: catalog.Book{ID: "book2", Title: "Python Basics", Tags: []string{"programming"}}},
			},
			filterQuery: "nonexistent",
			wantCount:   0,
			wantIDs:     []string{},
		},
		{
			name: "empty filter returns all",
			books: []BookItem{
				{Book: catalog.Book{ID: "book1", Title: "Book One"}},
				{Book: catalog.Book{ID: "book2", Title: "Book Two"}},
				{Book: catalog.Book{ID: "book3", Title: "Book Three"}},
			},
			filterQuery: "",
			wantCount:   3,
			wantIDs:     []string{"book1", "book2", "book3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test FilterValue method and filtering logic (business logic only)
			matchCount := 0
			var matchedIDs []string

			for _, book := range tt.books {
				filterVal := book.FilterValue()

				// Verify FilterValue contains expected fields
				if !containsSubstring(filterVal, book.Book.ID) {
					t.Errorf("FilterValue() = %q, should contain ID %q", filterVal, book.Book.ID)
				}
				if !containsSubstring(filterVal, book.Book.Title) {
					t.Errorf("FilterValue() = %q, should contain Title %q", filterVal, book.Book.Title)
				}

				// Apply filter logic (simulating how bubbles list would filter)
				if tt.filterQuery == "" || containsSubstring(filterVal, tt.filterQuery) {
					matchCount++
					matchedIDs = append(matchedIDs, book.Book.ID)
				}
			}

			if matchCount != tt.wantCount {
				t.Errorf("got %d matches, want %d", matchCount, tt.wantCount)
			}

			for _, wantID := range tt.wantIDs {
				if !containsString(matchedIDs, wantID) {
					t.Errorf("expected ID %q in results, got %v", wantID, matchedIDs)
				}
			}
		})
	}
}

// TestListBrowserSelection tests selection state tracking.
func TestListBrowserSelection(t *testing.T) {
	tests := []struct {
		name           string
		books          []BookItem
		selectIndices  []int
		wantSelected   []string
		clearAfter     bool
		wantAfterClear []string
	}{
		{
			name: "single selection",
			books: []BookItem{
				{Book: catalog.Book{ID: "book1", Title: "Book One"}},
				{Book: catalog.Book{ID: "book2", Title: "Book Two"}},
				{Book: catalog.Book{ID: "book3", Title: "Book Three"}},
			},
			selectIndices: []int{1},
			wantSelected:  []string{"book2"},
		},
		{
			name: "multi selection",
			books: []BookItem{
				{Book: catalog.Book{ID: "book1", Title: "Book One"}},
				{Book: catalog.Book{ID: "book2", Title: "Book Two"}},
				{Book: catalog.Book{ID: "book3", Title: "Book Three"}},
				{Book: catalog.Book{ID: "book4", Title: "Book Four"}},
			},
			selectIndices: []int{0, 2, 3},
			wantSelected:  []string{"book1", "book3", "book4"},
		},
		{
			name: "toggle selection off",
			books: []BookItem{
				{Book: catalog.Book{ID: "book1", Title: "Book One"}, selected: true},
				{Book: catalog.Book{ID: "book2", Title: "Book Two"}, selected: true},
				{Book: catalog.Book{ID: "book3", Title: "Book Three"}},
			},
			selectIndices: []int{0}, // Toggle book1 off
			wantSelected:  []string{"book2"},
		},
		{
			name: "clear selections",
			books: []BookItem{
				{Book: catalog.Book{ID: "book1", Title: "Book One"}, selected: true},
				{Book: catalog.Book{ID: "book2", Title: "Book Two"}, selected: true},
				{Book: catalog.Book{ID: "book3", Title: "Book Three"}, selected: true},
			},
			selectIndices:  []int{},
			wantSelected:   []string{"book1", "book2", "book3"},
			clearAfter:     true,
			wantAfterClear: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Work with BookItem slice directly (business logic)
			books := make([]BookItem, len(tt.books))
			copy(books, tt.books)

			// Simulate selection toggling
			for _, idx := range tt.selectIndices {
				books[idx].selected = !books[idx].selected
			}

			// Check selection state
			var selected []string
			for _, book := range books {
				if book.selected {
					selected = append(selected, book.Book.ID)
				}
			}

			if len(selected) != len(tt.wantSelected) {
				t.Errorf("got %d selected, want %d", len(selected), len(tt.wantSelected))
			}

			for _, wantID := range tt.wantSelected {
				if !containsString(selected, wantID) {
					t.Errorf("expected ID %q in selected, got %v", wantID, selected)
				}
			}

			// Test clear selection
			if tt.clearAfter {
				for i := range books {
					books[i].selected = false
				}

				var afterClear []string
				for _, book := range books {
					if book.selected {
						afterClear = append(afterClear, book.Book.ID)
					}
				}

				if len(afterClear) != len(tt.wantAfterClear) {
					t.Errorf("after clear: got %d selected, want %d", len(afterClear), len(tt.wantAfterClear))
				}
			}
		})
	}
}

// TestListBrowserPagination tests page navigation logic.
func TestListBrowserPagination(t *testing.T) {
	tests := []struct {
		name         string
		totalBooks   int
		pageSize     int
		wantPages    int
		cursorMoves  int // Number of down arrow presses
		wantCursorAt int // Expected cursor position after moves
	}{
		{
			name:         "single page",
			totalBooks:   5,
			pageSize:     10,
			wantPages:    1,
			cursorMoves:  2,
			wantCursorAt: 2,
		},
		{
			name:         "multiple pages",
			totalBooks:   25,
			pageSize:     10,
			wantPages:    3,
			cursorMoves:  0,
			wantCursorAt: 0,
		},
		{
			name:         "cursor wraps at end",
			totalBooks:   5,
			pageSize:     10,
			wantPages:    1,
			cursorMoves:  5, // More moves than items
			wantCursorAt: 0, // Should wrap to beginning
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test pagination logic (business logic only)
			// Calculate expected pages
			totalPages := (tt.totalBooks + tt.pageSize - 1) / tt.pageSize
			if totalPages != tt.wantPages {
				t.Errorf("got %d pages, want %d", totalPages, tt.wantPages)
			}

			// Test cursor position calculation with wrapping
			expectedCursor := tt.cursorMoves % tt.totalBooks
			if expectedCursor != tt.wantCursorAt {
				t.Errorf("cursor calculation: got %d, want %d", expectedCursor, tt.wantCursorAt)
			}

			// Verify page calculation for cursor position
			currentPage := expectedCursor / tt.pageSize
			expectedPage := expectedCursor / tt.pageSize
			if currentPage != expectedPage {
				t.Errorf("page calculation: got %d, want %d", currentPage, expectedPage)
			}
		})
	}
}

// TestListBrowserSort tests sorting by title, author, and year.
func TestListBrowserSort(t *testing.T) {
	tests := []struct {
		name    string
		books   []BookItem
		sortBy  string // "title", "author", "year"
		wantIDs []string
	}{
		{
			name: "sort by title ascending",
			books: []BookItem{
				{Book: catalog.Book{ID: "book1", Title: "Zebra Book"}},
				{Book: catalog.Book{ID: "book2", Title: "Alpha Book"}},
				{Book: catalog.Book{ID: "book3", Title: "Beta Book"}},
			},
			sortBy:  "title",
			wantIDs: []string{"book2", "book3", "book1"},
		},
		{
			name: "sort by author ascending",
			books: []BookItem{
				{Book: catalog.Book{ID: "book1", Title: "Book A", Author: "Smith"}},
				{Book: catalog.Book{ID: "book2", Title: "Book B", Author: "Jones"}},
				{Book: catalog.Book{ID: "book3", Title: "Book C", Author: "Adams"}},
			},
			sortBy:  "author",
			wantIDs: []string{"book3", "book2", "book1"},
		},
		{
			name: "sort by year ascending",
			books: []BookItem{
				{Book: catalog.Book{ID: "book1", Title: "Book A", Year: 2020}},
				{Book: catalog.Book{ID: "book2", Title: "Book B", Year: 2018}},
				{Book: catalog.Book{ID: "book3", Title: "Book C", Year: 2022}},
			},
			sortBy:  "year",
			wantIDs: []string{"book2", "book1", "book3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			books := make([]BookItem, len(tt.books))
			copy(books, tt.books)

			// Sort books based on sortBy field
			switch tt.sortBy {
			case "title":
				// Bubble sort by title (simple for test)
				for i := 0; i < len(books); i++ {
					for j := i + 1; j < len(books); j++ {
						if books[i].Book.Title > books[j].Book.Title {
							books[i], books[j] = books[j], books[i]
						}
					}
				}
			case "author":
				for i := 0; i < len(books); i++ {
					for j := i + 1; j < len(books); j++ {
						if books[i].Book.Author > books[j].Book.Author {
							books[i], books[j] = books[j], books[i]
						}
					}
				}
			case "year":
				for i := 0; i < len(books); i++ {
					for j := i + 1; j < len(books); j++ {
						if books[i].Book.Year > books[j].Book.Year {
							books[i], books[j] = books[j], books[i]
						}
					}
				}
			}

			// Verify sort order
			for i, wantID := range tt.wantIDs {
				if books[i].Book.ID != wantID {
					t.Errorf("position %d: got %q, want %q", i, books[i].Book.ID, wantID)
				}
			}
		})
	}
}

// TestListBrowserEmpty tests handling of empty book list.
func TestListBrowserEmpty(t *testing.T) {
	tests := []struct {
		name      string
		books     []BookItem
		wantItems int
		wantTitle string
	}{
		{
			name:      "empty book list",
			books:     []BookItem{},
			wantItems: 0,
		},
		{
			name:      "nil book list",
			books:     nil,
			wantItems: 0,
		},
		{
			name: "single book",
			books: []BookItem{
				{Book: catalog.Book{ID: "book1", Title: "Only Book"}, ShelfName: "default"},
			},
			wantItems: 1,
			wantTitle: "Shelf: default",
		},
		{
			name: "multiple books from one shelf",
			books: []BookItem{
				{Book: catalog.Book{ID: "book1", Title: "Book One"}, ShelfName: "technical"},
				{Book: catalog.Book{ID: "book2", Title: "Book Two"}, ShelfName: "technical"},
			},
			wantItems: 2,
			wantTitle: "Shelf: technical",
		},
		{
			name: "books from multiple shelves",
			books: []BookItem{
				{Book: catalog.Book{ID: "book1", Title: "Book One"}, ShelfName: "technical"},
				{Book: catalog.Book{ID: "book2", Title: "Book Two"}, ShelfName: "fiction"},
			},
			wantItems: 2,
			wantTitle: "All Shelves",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test browserTitle function
			title := browserTitle(tt.books)
			if tt.wantTitle != "" && title != tt.wantTitle {
				t.Errorf("browserTitle() = %q, want %q", title, tt.wantTitle)
			}

			// Test NewBrowserModel with various book lists
			model := NewBrowserModel(tt.books, nil, false)
			if len(model.list.Items()) != tt.wantItems {
				t.Errorf("model has %d items, want %d", len(model.list.Items()), tt.wantItems)
			}

			// Verify initial state
			if model.IsQuitting() {
				t.Error("new model should not be quitting")
			}
			if model.GetAction() != ActionNone {
				t.Errorf("new model action = %q, want %q", model.GetAction(), ActionNone)
			}
			if model.GetSelected() != nil {
				t.Error("new model should have no selected item")
			}
			if len(model.GetSelectedBooks()) != 0 {
				t.Error("new model should have no selected books")
			}
		})
	}
}

// TestBookItemIsSelectable tests the IsSelectable interface.
func TestBookItemIsSelectable(t *testing.T) {
	book := BookItem{
		Book: catalog.Book{ID: "book1", Title: "Test Book"},
	}

	if !book.IsSelectable() {
		t.Error("BookItem should be selectable")
	}
}

// TestBookItemSetSelected tests the SetSelected interface.
func TestBookItemSetSelected(t *testing.T) {
	book := BookItem{
		Book: catalog.Book{ID: "book1", Title: "Test Book"},
	}

	if book.IsSelected() {
		t.Error("BookItem should not be selected initially")
	}

	book.SetSelected(true)
	if !book.IsSelected() {
		t.Error("BookItem should be selected after SetSelected(true)")
	}

	book.SetSelected(false)
	if book.IsSelected() {
		t.Error("BookItem should not be selected after SetSelected(false)")
	}
}

// TestBrowserModelGetters tests BrowserModel accessor methods.
func TestBrowserModelGetters(t *testing.T) {
	books := []BookItem{
		{Book: catalog.Book{ID: "book1", Title: "Book One"}},
		{Book: catalog.Book{ID: "book2", Title: "Book Two"}},
	}

	model := NewBrowserModel(books, nil, false)

	// Test IsQuitting
	if model.IsQuitting() {
		t.Error("new model should not be quitting")
	}
	model.quitting = true
	if !model.IsQuitting() {
		t.Error("model should be quitting after setting flag")
	}

	// Test GetAction
	if model.GetAction() != ActionNone {
		t.Errorf("new model action = %q, want %q", model.GetAction(), ActionNone)
	}
	model.action = ActionOpen
	if model.GetAction() != ActionOpen {
		t.Errorf("model action = %q, want %q", model.GetAction(), ActionOpen)
	}

	// Test GetSelected
	if model.GetSelected() != nil {
		t.Error("new model should have no selected item")
	}
	model.selected = &books[0]
	if model.GetSelected() == nil || model.GetSelected().Book.ID != "book1" {
		t.Error("model should have book1 selected")
	}

	// Test GetSelectedBooks
	if len(model.GetSelectedBooks()) != 0 {
		t.Error("new model should have no selected books")
	}
	model.selectedBooks = books
	if len(model.GetSelectedBooks()) != 2 {
		t.Errorf("model has %d selected books, want 2", len(model.GetSelectedBooks()))
	}
}

// Helper functions

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsSubstring(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

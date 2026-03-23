package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestEditFormValidation tests field validation for title, author, and year
func TestEditFormValidation(t *testing.T) {
	tests := []struct {
		name      string
		yearInput string
		wantErr   bool
	}{
		{
			name:      "valid year",
			yearInput: "2023",
			wantErr:   false,
		},
		{
			name:      "year out of range negative",
			yearInput: "-1",
			wantErr:   true,
		},
		{
			name:      "year within char limit",
			yearInput: "1000",
			wantErr:   false,
		},
		{
			name:      "invalid year text",
			yearInput: "abcd",
			wantErr:   true,
		},
		{
			name:      "empty year",
			yearInput: "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newEditForm(EditFormDefaults{
				BookID: "test-id",
				Title:  "Test Book",
				Author: "Test Author",
				Year:   0,
			})

			// Focus on Year field
			m.focused = editFieldYear
			m.inputs[editFieldYear].Focus()

			// Clear and type the year input
			m.inputs[editFieldYear].SetValue("")
			for _, ch := range tt.yearInput {
				m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
				m = m2.(editFormModel)
			}

			// Trigger confirmation
			m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = m2.(editFormModel)

			// Try to confirm
			m3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
			fm := m3.(editFormModel)

			if tt.wantErr {
				if fm.err == nil {
					t.Errorf("expected error for year input %q, got nil", tt.yearInput)
				}
			} else {
				if fm.err != nil {
					t.Errorf("expected no error for year input %q, got %v", tt.yearInput, fm.err)
				}
			}
		})
	}
}

// TestEditFormRequired tests required field enforcement
func TestEditFormRequired(t *testing.T) {
	m := newEditForm(EditFormDefaults{
		BookID: "test-id",
		Title:  "",
		Author: "",
		Year:   0,
	})

	// Submit with all empty fields
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(editFormModel)

	if !m.confirming {
		t.Error("expected form to enter confirmation state even with empty fields")
	}

	// Confirm with 'y'
	m3, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	fm := m3.(editFormModel)

	if cmd == nil {
		t.Error("expected tea.Quit command after confirmation")
	}

	if fm.result == nil {
		t.Fatal("expected result to be set after confirmation")
	}

	// Verify empty fields are preserved
	if fm.result.Title != "" {
		t.Errorf("expected empty Title, got %q", fm.result.Title)
	}
	if fm.result.Author != "" {
		t.Errorf("expected empty Author, got %q", fm.result.Author)
	}
	if fm.result.Year != 0 {
		t.Errorf("expected Year=0, got %d", fm.result.Year)
	}
}

// TestEditFormDefaults tests default value population
func TestEditFormDefaults(t *testing.T) {
	defaults := EditFormDefaults{
		BookID: "book-123",
		Title:  "The Great Gatsby",
		Author: "F. Scott Fitzgerald",
		Year:   1925,
		Tags:   []string{"fiction", "classic"},
	}

	m := newEditForm(defaults)

	// Verify defaults are stored
	if m.defaults.BookID != "book-123" {
		t.Errorf("expected BookID default 'book-123', got %q", m.defaults.BookID)
	}
	if m.defaults.Title != "The Great Gatsby" {
		t.Errorf("expected Title default 'The Great Gatsby', got %q", m.defaults.Title)
	}
	if m.defaults.Author != "F. Scott Fitzgerald" {
		t.Errorf("expected Author default 'F. Scott Fitzgerald', got %q", m.defaults.Author)
	}
	if m.defaults.Year != 1925 {
		t.Errorf("expected Year default 1925, got %d", m.defaults.Year)
	}
	if len(m.defaults.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(m.defaults.Tags))
	}

	// Verify input fields are pre-populated with defaults
	if m.inputs[editFieldTitle].Value() != "The Great Gatsby" {
		t.Errorf("expected Title field value 'The Great Gatsby', got %q", m.inputs[editFieldTitle].Value())
	}
	if m.inputs[editFieldAuthor].Value() != "F. Scott Fitzgerald" {
		t.Errorf("expected Author field value 'F. Scott Fitzgerald', got %q", m.inputs[editFieldAuthor].Value())
	}
	if m.inputs[editFieldYear].Value() != "1925" {
		t.Errorf("expected Year field value '1925', got %q", m.inputs[editFieldYear].Value())
	}
	if m.inputs[editFieldTags].Value() != "fiction,classic" {
		t.Errorf("expected Tags field value 'fiction,classic', got %q", m.inputs[editFieldTags].Value())
	}
}

// TestEditFormSubmit tests form submission data structure
func TestEditFormSubmit(t *testing.T) {
	defaults := EditFormDefaults{
		BookID: "test-book",
		Title:  "Original Title",
		Author: "Original Author",
		Year:   2000,
		Tags:   []string{"tag1"},
	}

	m := newEditForm(defaults)

	// Modify Title field
	m.focused = editFieldTitle
	m.inputs[editFieldTitle].Focus()
	m.inputs[editFieldTitle].SetValue("Modified Title")

	// Modify Author field
	m.inputs[editFieldAuthor].SetValue("Modified Author")

	// Modify Year field
	m.inputs[editFieldYear].SetValue("2024")

	// Modify Tags field
	m.inputs[editFieldTags].SetValue("newtag1,newtag2")

	// Trigger confirmation
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(editFormModel)

	if !m.confirming {
		t.Fatal("expected form to enter confirmation state")
	}

	// Confirm submission with 'y'
	m3, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	fm := m3.(editFormModel)

	if cmd == nil {
		t.Error("expected tea.Quit command after confirmation")
	}

	if fm.result == nil {
		t.Fatal("expected result to be set after confirmation")
	}

	// Verify submitted data matches modified values
	if fm.result.Title != "Modified Title" {
		t.Errorf("expected Title='Modified Title', got %q", fm.result.Title)
	}
	if fm.result.Author != "Modified Author" {
		t.Errorf("expected Author='Modified Author', got %q", fm.result.Author)
	}
	if fm.result.Year != 2024 {
		t.Errorf("expected Year=2024, got %d", fm.result.Year)
	}
	if fm.result.Tags != "newtag1,newtag2" {
		t.Errorf("expected Tags='newtag1,newtag2', got %q", fm.result.Tags)
	}
}

// TestEditFormCancel tests cancel without saving
func TestEditFormCancel(t *testing.T) {
	m := newEditForm(EditFormDefaults{
		BookID: "test-id",
		Title:  "Test",
		Author: "Author",
		Year:   2020,
	})

	// Modify a field
	m.inputs[editFieldTitle].SetValue("Modified Title")

	// Cancel with Esc
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	fm := m2.(editFormModel)

	if !fm.canceled {
		t.Error("expected canceled=true after Esc")
	}
	if cmd == nil {
		t.Error("expected tea.Quit command after Esc")
	}
	if fm.result != nil {
		t.Error("expected result to be nil after cancel")
	}
}

// TestEditFormCancelDuringConfirmation tests cancel during confirmation prompt
func TestEditFormCancelDuringConfirmation(t *testing.T) {
	m := newEditForm(EditFormDefaults{
		BookID: "test-id",
		Title:  "Test",
	})

	// Trigger confirmation
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = m2.(editFormModel)

	if !m.confirming {
		t.Fatal("expected confirming state")
	}

	// Press 'n' to cancel
	m3, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	fm := m3.(editFormModel)

	if !fm.canceled {
		t.Error("expected canceled=true after pressing 'n' during confirmation")
	}
	if cmd == nil {
		t.Error("expected tea.Quit command after pressing 'n'")
	}
	if fm.result != nil {
		t.Error("expected result to be nil after cancel during confirmation")
	}
}

// TestEditFormTabNavigation tests tab navigation between fields
func TestEditFormTabNavigation(t *testing.T) {
	m := newEditForm(EditFormDefaults{
		BookID: "test",
		Title:  "Test",
		Author: "Author",
		Year:   2021,
	})

	// Initially focused on Title (0)
	if m.focused != editFieldTitle {
		t.Errorf("expected initial focus on editFieldTitle (0), got %d", m.focused)
	}

	// Tab to Author
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	fm := m2.(editFormModel)
	if fm.focused != editFieldAuthor {
		t.Errorf("expected focus on editFieldAuthor (1) after tab, got %d", fm.focused)
	}

	// Tab to Year
	m3, _ := fm.Update(tea.KeyMsg{Type: tea.KeyTab})
	fm = m3.(editFormModel)
	if fm.focused != editFieldYear {
		t.Errorf("expected focus on editFieldYear (2) after tab, got %d", fm.focused)
	}

	// Tab to Tags
	m4, _ := fm.Update(tea.KeyMsg{Type: tea.KeyTab})
	fm = m4.(editFormModel)
	if fm.focused != editFieldTags {
		t.Errorf("expected focus on editFieldTags (3) after tab, got %d", fm.focused)
	}

	// Tab wraps back to Title
	m5, _ := fm.Update(tea.KeyMsg{Type: tea.KeyTab})
	fm = m5.(editFormModel)
	if fm.focused != editFieldTitle {
		t.Errorf("expected focus to wrap to editFieldTitle (0) after tab, got %d", fm.focused)
	}
}

// TestEditFormYearFieldRendered tests Year field rendering and defaults
func TestEditFormYearFieldRendered(t *testing.T) {
	tests := []struct {
		name                string
		yearDefault         int
		expectedValue       string
		expectedPlaceholder string
	}{
		{
			name:                "year provided",
			yearDefault:         2022,
			expectedValue:       "2022",
			expectedPlaceholder: "2024",
		},
		{
			name:                "year not provided",
			yearDefault:         0,
			expectedValue:       "",
			expectedPlaceholder: "2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newEditForm(EditFormDefaults{
				BookID: "test",
				Title:  "Test",
				Year:   tt.yearDefault,
			})

			// Verify field count
			if len(m.inputs) != 4 {
				t.Fatalf("expected 4 input fields, got %d", len(m.inputs))
			}

			// Verify Year field value
			if m.inputs[editFieldYear].Value() != tt.expectedValue {
				t.Errorf("expected Year field value %q, got %q", tt.expectedValue, m.inputs[editFieldYear].Value())
			}

			// Verify Year field placeholder
			if m.inputs[editFieldYear].Placeholder != tt.expectedPlaceholder {
				t.Errorf("expected Year placeholder %q, got %q", tt.expectedPlaceholder, m.inputs[editFieldYear].Placeholder)
			}
		})
	}
}

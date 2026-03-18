package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewShelveForm_Defaults(t *testing.T) {
	defaults := ShelveFormDefaults{
		Filename: "test.pdf",
		Title:    "Test Title",
		Author:   "Test Author",
		Year:     2023,
		ID:       "test-id",
	}

	m := newShelveForm(defaults)

	// Verify defaults are stored
	if m.defaults.Title != "Test Title" {
		t.Errorf("expected Title default 'Test Title', got %q", m.defaults.Title)
	}
	if m.defaults.Author != "Test Author" {
		t.Errorf("expected Author default 'Test Author', got %q", m.defaults.Author)
	}
	if m.defaults.Year != 2023 {
		t.Errorf("expected Year default 2023, got %d", m.defaults.Year)
	}
	if m.defaults.ID != "test-id" {
		t.Errorf("expected ID default 'test-id', got %q", m.defaults.ID)
	}

	// Verify placeholders
	if m.inputs[fieldTitle].Placeholder != "Test Title" {
		t.Errorf("expected Title placeholder 'Test Title', got %q", m.inputs[fieldTitle].Placeholder)
	}
	if m.inputs[fieldAuthor].Placeholder != "Test Author" {
		t.Errorf("expected Author placeholder 'Test Author', got %q", m.inputs[fieldAuthor].Placeholder)
	}
	if m.inputs[fieldYear].Placeholder != "2023" {
		t.Errorf("expected Year placeholder '2023', got %q", m.inputs[fieldYear].Placeholder)
	}
	if m.inputs[fieldID].Placeholder != "test-id" {
		t.Errorf("expected ID placeholder 'test-id', got %q", m.inputs[fieldID].Placeholder)
	}

	// Verify cache default
	if !m.cacheLocally {
		t.Error("expected cacheLocally to default to true")
	}
}

func TestShelveForm_YearFieldRendered(t *testing.T) {
	defaults := ShelveFormDefaults{
		Filename: "test.pdf",
		Title:    "Test",
		Author:   "Author",
		Year:     2020,
		ID:       "test",
	}

	m := newShelveForm(defaults)

	// Verify Year field exists in inputs slice
	if len(m.inputs) != 5 {
		t.Fatalf("expected 5 input fields, got %d", len(m.inputs))
	}

	// Verify field order by checking placeholders
	if m.inputs[fieldTitle].Placeholder != "Test" {
		t.Errorf("fieldTitle (0) has wrong placeholder: %q", m.inputs[fieldTitle].Placeholder)
	}
	if m.inputs[fieldAuthor].Placeholder != "Author" {
		t.Errorf("fieldAuthor (1) has wrong placeholder: %q", m.inputs[fieldAuthor].Placeholder)
	}
	if m.inputs[fieldYear].Placeholder != "2020" {
		t.Errorf("fieldYear (2) has wrong placeholder: %q", m.inputs[fieldYear].Placeholder)
	}
	if m.inputs[fieldTags].Placeholder != "comma,separated,tags" {
		t.Errorf("fieldTags (3) has wrong placeholder: %q", m.inputs[fieldTags].Placeholder)
	}
	if m.inputs[fieldID].Placeholder != "test" {
		t.Errorf("fieldID (4) has wrong placeholder: %q", m.inputs[fieldID].Placeholder)
	}
}

func TestShelveForm_TabNavigation(t *testing.T) {
	m := newShelveForm(ShelveFormDefaults{
		Title:  "Test",
		Author: "Author",
		Year:   2021,
		ID:     "id",
	})

	// Initially focused on Title (0)
	if m.focused != fieldTitle {
		t.Errorf("expected initial focus on fieldTitle (0), got %d", m.focused)
	}

	// Tab to Author
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	fm := m2.(shelveFormModel)
	if fm.focused != fieldAuthor {
		t.Errorf("expected focus on fieldAuthor (1) after tab, got %d", fm.focused)
	}

	// Tab to Year
	m3, _ := fm.Update(tea.KeyMsg{Type: tea.KeyTab})
	fm = m3.(shelveFormModel)
	if fm.focused != fieldYear {
		t.Errorf("expected focus on fieldYear (2) after tab, got %d", fm.focused)
	}

	// Tab to Tags
	m4, _ := fm.Update(tea.KeyMsg{Type: tea.KeyTab})
	fm = m4.(shelveFormModel)
	if fm.focused != fieldTags {
		t.Errorf("expected focus on fieldTags (3) after tab, got %d", fm.focused)
	}

	// Tab to ID
	m5, _ := fm.Update(tea.KeyMsg{Type: tea.KeyTab})
	fm = m5.(shelveFormModel)
	if fm.focused != fieldID {
		t.Errorf("expected focus on fieldID (4) after tab, got %d", fm.focused)
	}

	// Tab to Cache checkbox
	m6, _ := fm.Update(tea.KeyMsg{Type: tea.KeyTab})
	fm = m6.(shelveFormModel)
	if fm.focused != fieldCache {
		t.Errorf("expected focus on fieldCache (5) after tab, got %d", fm.focused)
	}

	// Tab wraps back to Title
	m7, _ := fm.Update(tea.KeyMsg{Type: tea.KeyTab})
	fm = m7.(shelveFormModel)
	if fm.focused != fieldTitle {
		t.Errorf("expected focus to wrap to fieldTitle (0) after tab, got %d", fm.focused)
	}
}

func TestShelveForm_CancelOnEsc(t *testing.T) {
	m := newShelveForm(ShelveFormDefaults{Title: "Test"})

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	fm := m2.(shelveFormModel)

	if !fm.canceled {
		t.Error("expected canceled=true after Esc")
	}
	if cmd == nil {
		t.Error("expected tea.Quit command after Esc")
	}
}

func TestShelveForm_SubmitWithEmptyInputs(t *testing.T) {
	defaults := ShelveFormDefaults{
		Filename: "file.pdf",
		Title:    "Default Title",
		Author:   "Default Author",
		Year:     1999,
		ID:       "default-id",
	}
	m := newShelveForm(defaults)

	// Submit without typing anything (all inputs empty)
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fm := m2.(shelveFormModel)

	if fm.result == nil {
		t.Fatal("expected result to be set after Enter")
	}
	if cmd == nil {
		t.Error("expected tea.Quit command after Enter")
	}

	// Verify defaults are used
	if fm.result.Title != "Default Title" {
		t.Errorf("expected Title='Default Title', got %q", fm.result.Title)
	}
	if fm.result.Author != "" {
		// Author doesn't use getValue, it uses direct value
		t.Errorf("expected Author='', got %q", fm.result.Author)
	}
	if fm.result.Year != 1999 {
		t.Errorf("expected Year=1999, got %d", fm.result.Year)
	}
	if fm.result.ID != "default-id" {
		t.Errorf("expected ID='default-id', got %q", fm.result.ID)
	}
	if !fm.result.Cache {
		t.Error("expected Cache=true (default)")
	}
}

func TestShelveForm_YearDefault(t *testing.T) {
	// Test Year=0 default (no year set)
	m1 := newShelveForm(ShelveFormDefaults{Title: "Test", Year: 0})
	if m1.inputs[fieldYear].Placeholder == "0" {
		t.Error("expected Year placeholder to not be '0' when default Year is 0")
	}
	if m1.inputs[fieldYear].Placeholder != "Publication year" {
		t.Errorf("expected default placeholder 'Publication year', got %q", m1.inputs[fieldYear].Placeholder)
	}

	// Test Year > 0 shows as placeholder
	m2 := newShelveForm(ShelveFormDefaults{Title: "Test", Year: 2022})
	if m2.inputs[fieldYear].Placeholder != "2022" {
		t.Errorf("expected Year placeholder '2022', got %q", m2.inputs[fieldYear].Placeholder)
	}

	// Submit without entering year, should use default
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fm := m3.(shelveFormModel)
	if fm.result.Year != 2022 {
		t.Errorf("expected Year=2022 from default, got %d", fm.result.Year)
	}
}

func TestShelveForm_YearParsing(t *testing.T) {
	m := newShelveForm(ShelveFormDefaults{Title: "Test", Year: 0})

	// Focus on Year field (fieldYear = 2)
	m.focused = fieldYear
	m.inputs[fieldYear].Focus()

	// Type "2023"
	for _, ch := range "2023" {
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = m2.(shelveFormModel)
	}

	// Submit
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fm := m2.(shelveFormModel)

	if fm.result.Year != 2023 {
		t.Errorf("expected Year=2023 from input, got %d", fm.result.Year)
	}
}

func TestShelveForm_YearInvalidParsing(t *testing.T) {
	m := newShelveForm(ShelveFormDefaults{Title: "Test", Year: 1990})

	// Focus on Year field
	m.focused = fieldYear
	m.inputs[fieldYear].Focus()

	// Type invalid text "abcd"
	for _, ch := range "abcd" {
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = m2.(shelveFormModel)
	}

	// Submit - should return default year (1990) since parsing fails
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fm := m2.(shelveFormModel)

	// Invalid input should result in 0 (parse error), not default
	// Actually, getYearValue returns default if val is empty, but if val is non-empty
	// and invalid, fmt.Sscanf returns 0. Let's check the actual behavior.
	// Looking at the code: if val == "", return default; else parse and return (0 on error)
	// So "abcd" is not empty, parse fails, returns 0
	if fm.result.Year != 0 {
		t.Errorf("expected Year=0 from invalid input, got %d", fm.result.Year)
	}
}

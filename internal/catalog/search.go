package catalog

import "strings"

// Filter applies all non-empty criteria and returns matching books.
type Filter struct {
	Shelf  string // shelf name — handled by caller, not here
	Tag    string
	Search string // matches title, author, or any tag
	Format string
}

// Apply returns the subset of books matching all non-empty filter fields.
func (f Filter) Apply(books []Book) []Book {
	var out []Book
	for _, b := range books {
		if f.Tag != "" && !hasTag(b, f.Tag) {
			continue
		}
		if f.Format != "" && !strings.EqualFold(b.Format, f.Format) {
			continue
		}
		if f.Search != "" && !matchesSearch(b, f.Search) {
			continue
		}
		out = append(out, b)
	}
	return out
}

// ByID returns the first book with the given ID, or nil.
func ByID(books []Book, id string) *Book {
	for i := range books {
		if books[i].ID == id {
			return &books[i]
		}
	}
	return nil
}

func hasTag(b Book, tag string) bool {
	tag = strings.ToLower(tag)
	for _, t := range b.Tags {
		if strings.ToLower(t) == tag {
			return true
		}
	}
	return false
}

func matchesSearch(b Book, q string) bool {
	q = strings.ToLower(q)
	if strings.Contains(strings.ToLower(b.Title), q) {
		return true
	}
	if strings.Contains(strings.ToLower(b.Author), q) {
		return true
	}
	for _, t := range b.Tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	return false
}

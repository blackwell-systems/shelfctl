package catalog

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Marshal encodes a book list to YAML bytes.
func Marshal(books []Book) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(books); err != nil {
		return nil, fmt.Errorf("encoding catalog: %w", err)
	}
	return buf.Bytes(), nil
}

// Save writes the book list to a file on disk.
func Save(path string, books []Book) error {
	data, err := Marshal(books)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// Append adds a book to the list and returns the updated slice.
// If a book with the same ID already exists it is replaced.
func Append(books []Book, b Book) []Book {
	for i, existing := range books {
		if existing.ID == b.ID {
			books[i] = b
			return books
		}
	}
	return append(books, b)
}

// Remove removes a book by ID. Returns the updated slice and whether a book
// was actually removed.
func Remove(books []Book, id string) ([]Book, bool) {
	for i, b := range books {
		if b.ID == id {
			return append(books[:i], books[i+1:]...), true
		}
	}
	return books, false
}

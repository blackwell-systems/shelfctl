package catalog

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads a catalog.yml file from disk.
func Load(path string) ([]Book, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Book{}, nil
		}
		return nil, fmt.Errorf("reading catalog: %w", err)
	}
	return Parse(data)
}

// Parse decodes YAML bytes into a book list.
func Parse(data []byte) ([]Book, error) {
	if len(data) == 0 {
		return []Book{}, nil
	}
	var books []Book
	if err := yaml.Unmarshal(data, &books); err != nil {
		return nil, fmt.Errorf("parsing catalog YAML: %w", err)
	}
	if books == nil {
		return []Book{}, nil
	}
	return books, nil
}

package catalog

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

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

package fixtures

import (
	"crypto/sha256"
	"fmt"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
)

// FixtureSet contains pre-configured test data for multiple shelves.
type FixtureSet struct {
	Shelves []ShelfFixture
}

// ShelfFixture represents a complete shelf with books and asset content.
type ShelfFixture struct {
	Name   string
	Owner  string
	Repo   string
	Books  []catalog.Book
	Assets map[string][]byte // bookID -> PDF content
}

// DefaultFixtures returns a pre-configured fixture set with 3 shelves.
func DefaultFixtures() *FixtureSet {
	return &FixtureSet{
		Shelves: []ShelfFixture{
			createTechShelf(),
			createFictionShelf(),
			createReferenceShelf(),
		},
	}
}

// createTechShelf returns a shelf with technical books.
func createTechShelf() ShelfFixture {
	assets := make(map[string][]byte)

	// Create minimal valid PDF content for each book
	books := []catalog.Book{
		{
			ID:     "go-patterns",
			Title:  "Go Design Patterns",
			Author: "Mario Castro Contreras",
			Year:   2017,
			Tags:   []string{"golang", "patterns", "design"},
			Format: "pdf",
		},
		{
			ID:     "rust-programming",
			Title:  "The Rust Programming Language",
			Author: "Steve Klabnik and Carol Nichols",
			Year:   2023,
			Tags:   []string{"rust", "systems", "programming"},
			Format: "pdf",
		},
		{
			ID:     "distributed-systems",
			Title:  "Designing Data-Intensive Applications",
			Author: "Martin Kleppmann",
			Year:   2017,
			Tags:   []string{"distributed", "databases", "architecture"},
			Format: "pdf",
		},
		{
			ID:     "kubernetes-up",
			Title:  "Kubernetes: Up and Running",
			Author: "Brendan Burns and Joe Beda",
			Year:   2022,
			Tags:   []string{"kubernetes", "devops", "containers"},
			Format: "pdf",
		},
		{
			ID:     "site-reliability",
			Title:  "Site Reliability Engineering",
			Author: "Betsy Beyer",
			Year:   2016,
			Tags:   []string{"sre", "operations", "google"},
			Format: "pdf",
		},
		{
			ID:     "clean-code",
			Title:  "Clean Code: A Handbook of Agile Software Craftsmanship",
			Author: "Robert C. Martin",
			Year:   2008,
			Tags:   []string{"code-quality", "best-practices", "refactoring"},
			Format: "epub",
		},
		{
			ID:     "pragmatic-programmer",
			Title:  "The Pragmatic Programmer",
			Author: "David Thomas and Andrew Hunt",
			Year:   2019,
			Tags:   []string{"career", "programming", "best-practices"},
			Format: "pdf",
		},
	}

	// Generate assets and checksums for each book
	for i := range books {
		content := generateMinimalPDF(books[i].Title)
		assets[books[i].ID] = content
		books[i].Checksum = catalog.Checksum{
			SHA256: computeSHA256(content),
		}
		books[i].SizeBytes = int64(len(content))
		books[i].Source = catalog.Source{
			Type:    "github_release",
			Owner:   "tech-library",
			Repo:    "tech-books",
			Release: "v2023.1",
			Asset:   books[i].ID + "." + books[i].Format,
		}
	}

	return ShelfFixture{
		Name:   "tech",
		Owner:  "tech-library",
		Repo:   "tech-books",
		Books:  books,
		Assets: assets,
	}
}

// createFictionShelf returns a shelf with fiction books.
func createFictionShelf() ShelfFixture {
	assets := make(map[string][]byte)

	books := []catalog.Book{
		{
			ID:     "project-hail-mary",
			Title:  "Project Hail Mary",
			Author: "Andy Weir",
			Year:   2021,
			Tags:   []string{"science-fiction", "space", "adventure"},
			Format: "epub",
		},
		{
			ID:     "neuromancer",
			Title:  "Neuromancer",
			Author: "William Gibson",
			Year:   1984,
			Tags:   []string{"cyberpunk", "science-fiction", "classic"},
			Format: "pdf",
		},
		{
			ID:     "foundation",
			Title:  "Foundation",
			Author: "Isaac Asimov",
			Year:   1951,
			Tags:   []string{"science-fiction", "space-opera", "classic"},
			Format: "epub",
		},
		{
			ID:     "dune",
			Title:  "Dune",
			Author: "Frank Herbert",
			Year:   1965,
			Tags:   []string{"science-fiction", "epic", "classic"},
			Format: "pdf",
		},
		{
			ID:     "left-hand-darkness",
			Title:  "The Left Hand of Darkness",
			Author: "Ursula K. Le Guin",
			Year:   1969,
			Tags:   []string{"science-fiction", "anthropology", "classic"},
			Format: "epub",
		},
		{
			ID:     "enders-game",
			Title:  "Ender's Game",
			Author: "Orson Scott Card",
			Year:   1985,
			Tags:   []string{"science-fiction", "military", "young-adult"},
			Format: "pdf",
		},
		{
			ID:     "snow-crash",
			Title:  "Snow Crash",
			Author: "Neal Stephenson",
			Year:   1992,
			Tags:   []string{"cyberpunk", "science-fiction", "metaverse"},
			Format: "epub",
		},
		{
			ID:     "three-body-problem",
			Title:  "The Three-Body Problem",
			Author: "Liu Cixin",
			Year:   2008,
			Tags:   []string{"science-fiction", "hard-sf", "chinese"},
			Format: "pdf",
		},
	}

	for i := range books {
		content := generateMinimalPDF(books[i].Title)
		assets[books[i].ID] = content
		books[i].Checksum = catalog.Checksum{
			SHA256: computeSHA256(content),
		}
		books[i].SizeBytes = int64(len(content))
		books[i].Source = catalog.Source{
			Type:    "github_release",
			Owner:   "fiction-archive",
			Repo:    "scifi-classics",
			Release: "v2024.2",
			Asset:   books[i].ID + "." + books[i].Format,
		}
	}

	return ShelfFixture{
		Name:   "fiction",
		Owner:  "tech-library",
		Repo:   "scifi-classics",
		Books:  books,
		Assets: assets,
	}
}

// createReferenceShelf returns a shelf with reference books.
func createReferenceShelf() ShelfFixture {
	assets := make(map[string][]byte)

	books := []catalog.Book{
		{
			ID:     "c-programming-language",
			Title:  "The C Programming Language",
			Author: "Brian W. Kernighan and Dennis M. Ritchie",
			Year:   1988,
			Tags:   []string{"c", "programming", "classic", "reference"},
			Format: "pdf",
		},
		{
			ID:     "structure-interpretation",
			Title:  "Structure and Interpretation of Computer Programs",
			Author: "Harold Abelson and Gerald Jay Sussman",
			Year:   1996,
			Tags:   []string{"scheme", "programming", "computer-science"},
			Format: "pdf",
		},
		{
			ID:     "art-computer-programming",
			Title:  "The Art of Computer Programming Vol 1",
			Author: "Donald E. Knuth",
			Year:   1997,
			Tags:   []string{"algorithms", "computer-science", "classic"},
			Format: "pdf",
		},
		{
			ID:     "design-patterns",
			Title:  "Design Patterns: Elements of Reusable Object-Oriented Software",
			Author: "Erich Gamma et al.",
			Year:   1994,
			Tags:   []string{"patterns", "oop", "design", "gang-of-four"},
			Format: "pdf",
		},
		{
			ID:     "unix-programming",
			Title:  "The Art of Unix Programming",
			Author: "Eric S. Raymond",
			Year:   2003,
			Tags:   []string{"unix", "philosophy", "programming"},
			Format: "pdf",
		},
		{
			ID:     "compilers-dragon-book",
			Title:  "Compilers: Principles, Techniques, and Tools",
			Author: "Alfred V. Aho et al.",
			Year:   2006,
			Tags:   []string{"compilers", "parsing", "computer-science"},
			Format: "pdf",
		},
		{
			ID:     "introduction-algorithms",
			Title:  "Introduction to Algorithms",
			Author: "Thomas H. Cormen et al.",
			Year:   2009,
			Tags:   []string{"algorithms", "data-structures", "reference"},
			Format: "pdf",
		},
		{
			ID:     "concrete-mathematics",
			Title:  "Concrete Mathematics",
			Author: "Ronald L. Graham et al.",
			Year:   1994,
			Tags:   []string{"mathematics", "computer-science", "reference"},
			Format: "pdf",
		},
		{
			ID:     "database-systems",
			Title:  "Database System Concepts",
			Author: "Abraham Silberschatz et al.",
			Year:   2019,
			Tags:   []string{"databases", "sql", "reference"},
			Format: "pdf",
		},
		{
			ID:     "operating-systems",
			Title:  "Operating System Concepts",
			Author: "Abraham Silberschatz et al.",
			Year:   2018,
			Tags:   []string{"operating-systems", "os", "reference"},
			Format: "pdf",
		},
	}

	for i := range books {
		content := generateMinimalPDF(books[i].Title)
		assets[books[i].ID] = content
		books[i].Checksum = catalog.Checksum{
			SHA256: computeSHA256(content),
		}
		books[i].SizeBytes = int64(len(content))
		books[i].Source = catalog.Source{
			Type:    "github_release",
			Owner:   "cs-reference",
			Repo:    "classic-texts",
			Release: "v2023.12",
			Asset:   books[i].ID + "." + books[i].Format,
		}
	}

	return ShelfFixture{
		Name:   "reference",
		Owner:  "tech-library",
		Repo:   "classic-texts",
		Books:  books,
		Assets: assets,
	}
}

// generateMinimalPDF creates a minimal valid PDF with the given title.
// This is a 1KB placeholder suitable for testing.
func generateMinimalPDF(title string) []byte {
	// Minimal PDF structure with a valid header
	pdf := fmt.Sprintf(`%%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>
endobj
4 0 obj
<< /Length 44 >>
stream
BT
/F1 12 Tf
100 700 Td
(%s) Tj
ET
endstream
endobj
xref
0 5
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
0000000214 00000 n
trailer
<< /Size 5 /Root 1 0 R >>
startxref
308
%%%%EOF
`, title)
	return []byte(pdf)
}

// computeSHA256 calculates the SHA256 checksum of the given content.
func computeSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

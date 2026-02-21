package cache

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/catalog"
)

// IndexBook represents a book for HTML index generation
type IndexBook struct {
	Book      catalog.Book
	ShelfName string
	Repo      string
	FilePath  string // Path to cached file
	CoverPath string // Path to cover (catalog or extracted)
	HasCover  bool
}

// GenerateHTMLIndex creates an index.html in the cache directory with all cached books.
func (m *Manager) GenerateHTMLIndex(books []IndexBook) error {
	indexPath := filepath.Join(m.baseDir, "index.html")

	html := generateHTML(books)

	if err := os.WriteFile(indexPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("writing index.html: %w", err)
	}

	return nil
}

func generateHTML(books []IndexBook) string {
	var s strings.Builder

	s.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>shelfctl Library</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: #1a1a1a;
            color: #e0e0e0;
            padding: 20px;
            line-height: 1.6;
        }
        header {
            max-width: 1200px;
            margin: 0 auto 30px;
        }
        h1 {
            font-size: 2rem;
            margin-bottom: 10px;
            color: #fff;
        }
        .subtitle {
            color: #888;
            font-size: 0.9rem;
        }
        .search-box {
            max-width: 1200px;
            margin: 0 auto 30px;
        }
        #search {
            width: 100%;
            padding: 12px 20px;
            font-size: 1rem;
            background: #2a2a2a;
            border: 1px solid #444;
            border-radius: 8px;
            color: #e0e0e0;
        }
        #search:focus {
            outline: none;
            border-color: #4a9eff;
        }
        .shelf-section {
            max-width: 1200px;
            margin: 0 auto 40px;
        }
        .shelf-title {
            font-size: 1.3rem;
            margin-bottom: 15px;
            color: #4a9eff;
            border-bottom: 2px solid #333;
            padding-bottom: 8px;
        }
        .book-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
            gap: 20px;
        }
        .book-card {
            background: #2a2a2a;
            border: 1px solid #333;
            border-radius: 8px;
            padding: 15px;
            cursor: pointer;
            transition: all 0.2s;
            text-decoration: none;
            color: inherit;
            display: block;
        }
        .book-card:hover {
            transform: translateY(-2px);
            border-color: #4a9eff;
            box-shadow: 0 4px 12px rgba(74, 158, 255, 0.3);
        }
        .book-cover {
            width: 100%;
            height: 200px;
            background: #1a1a1a;
            border-radius: 4px;
            margin-bottom: 12px;
            display: flex;
            align-items: center;
            justify-content: center;
            overflow: hidden;
        }
        .book-cover img {
            max-width: 100%;
            max-height: 100%;
            object-fit: contain;
        }
        .book-cover.no-cover {
            font-size: 3rem;
        }
        .book-id {
            font-size: 0.85rem;
            color: #888;
            margin-bottom: 5px;
            font-family: monospace;
        }
        .book-title {
            font-size: 1rem;
            font-weight: 600;
            margin-bottom: 5px;
            color: #fff;
        }
        .book-author {
            font-size: 0.9rem;
            color: #aaa;
            margin-bottom: 8px;
        }
        .book-tags {
            display: flex;
            flex-wrap: wrap;
            gap: 5px;
        }
        .tag {
            background: #333;
            color: #4a9eff;
            padding: 3px 8px;
            border-radius: 4px;
            font-size: 0.8rem;
        }
        .no-results {
            text-align: center;
            color: #888;
            padding: 40px;
            font-size: 1.1rem;
        }
        @media (max-width: 768px) {
            .book-grid {
                grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
            }
        }
    </style>
</head>
<body>
    <header>
        <h1>📚 shelfctl Library</h1>
        <div class="subtitle">` + fmt.Sprintf("%d books", len(books)) + `</div>
    </header>

    <div class="search-box">
        <input type="text" id="search" placeholder="Search books by title, author, or tags...">
    </div>

    <div id="library">
`)

	// Group books by shelf
	shelfBooks := make(map[string][]IndexBook)
	for _, book := range books {
		shelfBooks[book.ShelfName] = append(shelfBooks[book.ShelfName], book)
	}

	// Render each shelf section
	for shelfName, shelfBookList := range shelfBooks {
		s.WriteString(fmt.Sprintf(`
        <div class="shelf-section" data-shelf="%s">
            <h2 class="shelf-title">%s (%d)</h2>
            <div class="book-grid">
`, html.EscapeString(shelfName), html.EscapeString(shelfName), len(shelfBookList)))

		for _, book := range shelfBookList {
			renderBookCard(&s, book)
		}

		s.WriteString(`
            </div>
        </div>
`)
	}

	s.WriteString(`
    </div>

    <div id="no-results" class="no-results" style="display:none;">
        No books match your search.
    </div>

    <script>
        const search = document.getElementById('search');
        const library = document.getElementById('library');
        const noResults = document.getElementById('no-results');

        search.addEventListener('input', (e) => {
            const query = e.target.value.toLowerCase();
            const cards = document.querySelectorAll('.book-card');
            let visibleCount = 0;

            cards.forEach(card => {
                const text = card.textContent.toLowerCase();
                if (text.includes(query)) {
                    card.style.display = 'block';
                    visibleCount++;
                } else {
                    card.style.display = 'none';
                }
            });

            // Show/hide sections based on visible cards
            document.querySelectorAll('.shelf-section').forEach(section => {
                const visibleCards = section.querySelectorAll('.book-card[style*="display: block"], .book-card:not([style*="display: none"])');
                section.style.display = visibleCards.length > 0 ? 'block' : 'none';
            });

            // Show "no results" message
            if (visibleCount === 0 && query !== '') {
                library.style.display = 'none';
                noResults.style.display = 'block';
            } else {
                library.style.display = 'block';
                noResults.style.display = 'none';
            }
        });
    </script>
</body>
</html>
`)

	return s.String()
}

func renderBookCard(s *strings.Builder, book IndexBook) {
	// Convert tags to lowercase for search
	tags := strings.Join(book.Book.Tags, ", ")

	fmt.Fprintf(s, `
                <a href="file://%s" class="book-card" data-id="%s" data-tags="%s">
                    <div class="book-cover%s">
`,
		html.EscapeString(book.FilePath),
		html.EscapeString(book.Book.ID),
		html.EscapeString(tags),
		func() string {
			if !book.HasCover {
				return " no-cover"
			}
			return ""
		}(),
	)

	if book.HasCover {
		// Make cover path relative to index.html location
		relCoverPath, _ := filepath.Rel(filepath.Dir(filepath.Join(book.FilePath, "..")), book.CoverPath)
		fmt.Fprintf(s, `<img src="%s" alt="Cover">`,
			html.EscapeString(relCoverPath))
	} else {
		s.WriteString("📚")
	}

	s.WriteString(`
                    </div>
                    <div class="book-id">` + html.EscapeString(book.Book.ID) + `</div>
                    <div class="book-title">` + html.EscapeString(book.Book.Title) + `</div>
`)

	if book.Book.Author != "" {
		s.WriteString(`                    <div class="book-author">` + html.EscapeString(book.Book.Author) + `</div>
`)
	}

	if len(book.Book.Tags) > 0 {
		s.WriteString(`                    <div class="book-tags">
`)
		for _, tag := range book.Book.Tags {
			fmt.Fprintf(s, `                        <span class="tag">%s</span>
`, html.EscapeString(tag))
		}
		s.WriteString(`                    </div>
`)
	}

	s.WriteString(`                </a>
`)
}

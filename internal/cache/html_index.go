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

	// Collect all unique tags
	tagSet := make(map[string]int)
	for _, book := range books {
		for _, tag := range book.Book.Tags {
			tagSet[tag]++
		}
	}

	s.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>shelfctl Library</title>
    <style>
        :root {
            --orange: #fb6820;
            --orange-dark: #d45610;
            --teal: #1b8487;
            --teal-light: #2ecfd4;
            --teal-dim: #0d3536;
            --teal-card: #1c2829;
            --teal-border: #1e3a3c;
            --teal-nav-border: #1b4e50;
        }
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: #1a1a1a;
            color: #e0e0e0;
            line-height: 1.6;
            padding-top: 0;
        }
        .sticky-nav {
            position: sticky;
            top: 0;
            z-index: 1000;
            background: #1a1a1a;
            padding: 20px 20px 10px;
            border-bottom: 2px solid var(--teal-nav-border);
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.5);
        }
        header {
            max-width: 1200px;
            margin: 0 auto 20px;
        }
        h1 {
            font-size: 2rem;
            margin-bottom: 10px;
        }
        h1 .brand-shelf {
            color: var(--orange);
        }
        h1 .brand-ctl {
            color: var(--teal-light);
        }
        h1 .brand-suffix {
            color: #fff;
        }
        .subtitle {
            color: #888;
            font-size: 0.9rem;
        }
        .controls {
            max-width: 1200px;
            margin: 0 auto 15px;
            display: flex;
            gap: 15px;
            align-items: center;
        }
        .search-box {
            flex: 1;
        }
        .sort-box {
            min-width: 200px;
        }
        #sort-by {
            width: 100%;
            padding: 12px 15px;
            font-size: 0.95rem;
            background: #2a2a2a;
            border: 1px solid #444;
            border-radius: 8px;
            color: #e0e0e0;
            cursor: pointer;
        }
        #sort-by:focus {
            outline: none;
            border-color: var(--teal-light);
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
            border-color: var(--teal-light);
        }
        .tag-filters {
            max-width: 1200px;
            margin: 0 auto 20px;
        }
        .content-wrapper {
            padding: 20px;
        }
        .tag-filters-title {
            font-size: 0.9rem;
            color: #888;
            margin-bottom: 10px;
        }
        .tag-cloud {
            display: flex;
            flex-wrap: wrap;
            gap: 10px;
        }
        .tag-filter {
            background: #2a2a2a;
            border: 2px solid #444;
            color: #e0e0e0;
            padding: 8px 16px;
            border-radius: 6px;
            cursor: pointer;
            font-size: 0.9rem;
            transition: all 0.2s;
            user-select: none;
        }
        .tag-filter:hover {
            border-color: var(--orange);
            background: #333;
        }
        .tag-filter.active {
            background: var(--orange);
            border-color: var(--orange);
            color: #fff;
            font-weight: 600;
        }
        .tag-filter.active:hover {
            background: var(--orange-dark);
            border-color: var(--orange-dark);
        }
        .tag-count {
            color: #888;
            font-size: 0.85rem;
            margin-left: 6px;
        }
        .tag-filter.active .tag-count {
            color: rgba(255,255,255,0.8);
        }
        .clear-filters {
            background: #3a2020;
            border: 2px solid #6b2c2c;
            color: #e07070;
            padding: 8px 16px;
            border-radius: 6px;
            cursor: pointer;
            font-size: 0.9rem;
            transition: all 0.2s;
            user-select: none;
            display: none;
        }
        .clear-filters:hover {
            background: #4a2020;
            border-color: #8b3c3c;
            color: #f08080;
        }
        .clear-filters.visible {
            display: inline-block;
        }
        .filter-count {
            color: var(--teal-light);
            font-size: 0.85rem;
            margin-left: 10px;
        }
        .shelf-section {
            max-width: 1200px;
            margin: 0 auto 40px;
        }
        .shelf-title {
            font-size: 1.3rem;
            margin-bottom: 15px;
            color: var(--orange);
            border-bottom: 2px solid var(--teal-border);
            padding-bottom: 8px;
        }
        .book-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
            gap: 20px;
        }
        .book-card {
            background: var(--teal-card);
            border: 1px solid var(--teal-border);
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
            border-color: var(--orange);
            box-shadow: 0 4px 12px rgba(251, 104, 32, 0.2);
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
            background: var(--teal-dim);
            color: var(--teal-light);
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
            .controls {
                flex-direction: column;
            }
            .sort-box {
                width: 100%;
            }
            .book-grid {
                grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
            }
        }
    </style>
</head>
<body>
    <div class="sticky-nav">
        <header>
            <h1>📚 <span class="brand-shelf">shelf</span><span class="brand-ctl">ctl</span> <span class="brand-suffix">Library</span></h1>
            <div class="subtitle">` + fmt.Sprintf("%d books", len(books)) + `</div>
        </header>

        <div class="controls">
            <div class="search-box">
                <input type="text" id="search" placeholder="Search books by title, author, or tags...">
            </div>
            <div class="sort-box">
                <select id="sort-by">
                    <option value="recent">Recently Added</option>
                    <option value="title">Title (A-Z)</option>
                    <option value="author">Author (A-Z)</option>
                    <option value="year-desc">Year (Newest First)</option>
                    <option value="year-asc">Year (Oldest First)</option>
                </select>
            </div>
        </div>
`)

	// Only show tag filters if we have tags
	if len(tagSet) > 0 {
		s.WriteString(`
        <div class="tag-filters">
            <div class="tag-filters-title">
                Filter by tag:
                <button class="clear-filters" id="clear-filters">Clear filters</button>
                <span class="filter-count" id="filter-count"></span>
            </div>
            <div class="tag-cloud" id="tag-cloud">
`)

		// Render tag filter buttons (sorted alphabetically)
		var allTags []string
		for tag := range tagSet {
			allTags = append(allTags, tag)
		}
		// Sort tags
		sortedTags := allTags
		for i := 0; i < len(sortedTags)-1; i++ {
			for j := i + 1; j < len(sortedTags); j++ {
				if sortedTags[i] > sortedTags[j] {
					sortedTags[i], sortedTags[j] = sortedTags[j], sortedTags[i]
				}
			}
		}

		for _, tag := range sortedTags {
			count := tagSet[tag]
			fmt.Fprintf(&s, `                <button class="tag-filter" data-tag="%s">%s <span class="tag-count">%d</span></button>
`, html.EscapeString(tag), html.EscapeString(tag), count)
		}

		s.WriteString(`            </div>
        </div>
`)
	}

	s.WriteString(`
    </div>

    <div class="content-wrapper">
        <div id="library">
`)

	// Group books by shelf
	shelfBooks := make(map[string][]IndexBook)
	for _, book := range books {
		shelfBooks[book.ShelfName] = append(shelfBooks[book.ShelfName], book)
	}

	// Render each shelf section
	bookIndex := 0
	for shelfName, shelfBookList := range shelfBooks {
		fmt.Fprintf(&s, `
            <div class="shelf-section" data-shelf="%s">
                <h2 class="shelf-title">%s (%d)</h2>
                <div class="book-grid">
`, html.EscapeString(shelfName), html.EscapeString(shelfName), len(shelfBookList))

		for _, book := range shelfBookList {
			renderBookCard(&s, book, bookIndex)
			bookIndex++
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
    </div>

    <script>
        const search = document.getElementById('search');
        const library = document.getElementById('library');
        const noResults = document.getElementById('no-results');
        const tagFilters = document.querySelectorAll('.tag-filter');
        const clearFiltersBtn = document.getElementById('clear-filters');
        const filterCount = document.getElementById('filter-count');
        let activeTags = new Set();

        // Tag filter click handler
        tagFilters.forEach(filter => {
            filter.addEventListener('click', () => {
                const tag = filter.dataset.tag;
                if (activeTags.has(tag)) {
                    activeTags.delete(tag);
                    filter.classList.remove('active');
                } else {
                    activeTags.add(tag);
                    filter.classList.add('active');
                }
                applyFilters();
            });
        });

        // Clear filters button
        if (clearFiltersBtn) {
            clearFiltersBtn.addEventListener('click', () => {
                activeTags.clear();
                tagFilters.forEach(filter => filter.classList.remove('active'));
                applyFilters();
            });
        }

        // Search input handler
        search.addEventListener('input', applyFilters);

        // Sort dropdown handler
        const sortBy = document.getElementById('sort-by');
        sortBy.addEventListener('change', sortBooks);

        function sortBooks() {
            const sortValue = sortBy.value;
            const sections = document.querySelectorAll('.shelf-section');

            sections.forEach(section => {
                const grid = section.querySelector('.book-grid');
                const cards = Array.from(grid.querySelectorAll('.book-card'));

                cards.sort((a, b) => {
                    switch(sortValue) {
                        case 'title':
                            return a.dataset.title.localeCompare(b.dataset.title);
                        case 'author':
                            const authorA = a.dataset.author || '';
                            const authorB = b.dataset.author || '';
                            return authorA.localeCompare(authorB);
                        case 'year-desc':
                            return parseInt(b.dataset.year || 0) - parseInt(a.dataset.year || 0);
                        case 'year-asc':
                            return parseInt(a.dataset.year || 0) - parseInt(b.dataset.year || 0);
                        case 'recent':
                        default:
                            return parseInt(b.dataset.index || 0) - parseInt(a.dataset.index || 0);
                    }
                });

                // Re-append cards in sorted order
                cards.forEach(card => grid.appendChild(card));
            });
        }

        function applyFilters() {
            const query = search.value.toLowerCase();
            const cards = document.querySelectorAll('.book-card');
            let visibleCount = 0;

            cards.forEach(card => {
                const text = card.textContent.toLowerCase();
                const cardTags = card.dataset.tags.toLowerCase().split(', ').filter(t => t);

                // Check text search
                const matchesSearch = query === '' || text.includes(query);

                // Check tag filters (must have ALL active tags)
                let matchesTags = true;
                if (activeTags.size > 0) {
                    matchesTags = Array.from(activeTags).every(tag =>
                        cardTags.includes(tag.toLowerCase())
                    );
                }

                if (matchesSearch && matchesTags) {
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

            // Show/hide clear button
            if (clearFiltersBtn) {
                if (activeTags.size > 0) {
                    clearFiltersBtn.classList.add('visible');
                } else {
                    clearFiltersBtn.classList.remove('visible');
                }
            }

            // Update filter count
            if (filterCount) {
                if (activeTags.size > 0 || query !== '') {
                    filterCount.textContent = visibleCount + ' books';
                } else {
                    filterCount.textContent = '';
                }
            }

            // Show "no results" message
            if (visibleCount === 0 && (query !== '' || activeTags.size > 0)) {
                library.style.display = 'none';
                noResults.style.display = 'block';
            } else {
                library.style.display = 'block';
                noResults.style.display = 'none';
            }
        }
    </script>
</body>
</html>
`)

	return s.String()
}

func renderBookCard(s *strings.Builder, book IndexBook, index int) {
	// Convert tags to lowercase for search
	tags := strings.Join(book.Book.Tags, ", ")

	fmt.Fprintf(s, `
                <a href="file://%s" class="book-card" data-id="%s" data-tags="%s" data-title="%s" data-author="%s" data-year="%d" data-index="%d">
                    <div class="book-cover%s">
`,
		html.EscapeString(book.FilePath),
		html.EscapeString(book.Book.ID),
		html.EscapeString(tags),
		html.EscapeString(book.Book.Title),
		html.EscapeString(book.Book.Author),
		book.Book.Year,
		index,
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

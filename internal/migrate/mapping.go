package migrate

import (
	"strings"

	"github.com/blackwell-systems/shelfctl/internal/config"
)

// RouteSource matches oldPath against a MigrationSource's mapping and
// returns the target shelf name, or "" if no mapping matches.
func RouteSource(oldPath string, src config.MigrationSource) string {
	// Longest prefix match wins.
	best := ""
	bestShelf := ""
	for prefix, shelf := range src.Mapping {
		if strings.HasPrefix(oldPath, prefix) && len(prefix) > len(best) {
			best = prefix
			bestShelf = shelf
		}
	}
	return bestShelf
}

// FindRoute scans all migration sources and returns the first matching
// (source, shelf) pair, or ("", "", "") if nothing matches.
func FindRoute(oldPath string, sources []config.MigrationSource) (config.MigrationSource, string, bool) {
	for _, src := range sources {
		if shelf := RouteSource(oldPath, src); shelf != "" {
			return src, shelf, true
		}
	}
	return config.MigrationSource{}, "", false
}

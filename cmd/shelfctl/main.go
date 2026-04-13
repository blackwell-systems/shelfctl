// shelfctl is a CLI/TUI for managing a personal PDF/EPUB library backed by
// GitHub Release assets. Books are stored as Release assets (not git commits),
// keeping repositories lightweight and enabling on-demand per-file downloads.
// A catalog.yml in each shelf repo holds searchable metadata.
package main

import "github.com/blackwell-systems/shelfctl/internal/app"

// version is set by goreleaser via ldflags.
var version = "dev"

func main() {
	app.SetVersion(version)
	app.Execute()
}

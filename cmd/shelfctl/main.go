package main

import "github.com/blackwell-systems/shelfctl/internal/app"

// version is set by goreleaser via ldflags.
var version = "dev"

func main() {
	app.SetVersion(version)
	app.Execute()
}

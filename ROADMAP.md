# Roadmap

## CLI

| Feature | Status | Description |
|---------|--------|-------------|
| **`shelfctl doctor`** | Done (v0.4.9) | Checks config, token, API connectivity, scopes, shelf access, cache integrity |
| **Token scope detection** | Done (v0.4.8) | 403 on private repos now shows actionable "check token scope" message |
| **Shell completions** | Planned | Bash/zsh/fish/PowerShell via Cobra's built-in generators. `shelfctl completion <shell>` |
| **Cross-shelf search** | Planned | `shelfctl search --all` to search across all configured shelves |

## TUI

| Feature | Status | Description |
|---------|--------|-------------|
| **Error behavior docs** | Done | Error Handling section in docs/guides/hub.md |
| **Auto-sync toggle** | Done (v0.4.7) | Toggle auto-sync from the hub menu |
| **Settings screen** | Planned | Read-only config viewer + editor for simple fields (auto-sync, debounce, cache dir, asset naming). Structural config stays CLI-only |
| **Cross-shelf search** | Planned | Search across all shelves from the TUI search view |

## Documentation

| Feature | Status | Description |
|---------|--------|-------------|
| **catalog.yml editing guide** | Done | "Editing catalog.yml Directly" in docs/reference/architecture.md |
| **Demo GIF** | Planned | Asciinema or GIF recording of the TUI for README |

## Distribution

| Feature | Status | Description |
|---------|--------|-------------|
| **APT/RPM packages** | Done (v0.4.9) | `.deb` and `.rpm` via GoReleaser nfpms, published as release assets |
| **AUR package** | Planned | Arch Linux AUR package. Low maintenance once set up |
| **Nix flake** | Planned (low priority) | `nix profile install` support |

## Features

| Feature | Status | Description |
|---------|--------|-------------|
| **Calibre import (Phase 1)** | Planned | Import books from Calibre's `metadata.db` into shelfctl shelves. Design: [docs/planning/calibre-integration.md](docs/planning/calibre-integration.md) |
| **OPDS server** | Planned | `shelfctl serve` — lightweight OPDS feed for e-reader apps (KOReader, Moon+ Reader) |
| **Reading lists** | Planned | User-defined ordered collections spanning shelves, with progress tracking |
| **Deduplication** | Planned (low priority) | Detect duplicate books across shelves via file hash comparison |
| **Book annotations** | Planned (low priority) | Attach notes/highlights to books, stored in catalog.yml metadata |

# Roadmap

## CLI

| Feature | Status | Description |
|---------|--------|-------------|
| **`shelfctl doctor`** | Planned | Check prerequisites (Git, poppler), token validity, shelf connectivity — single command for "is my setup working?" |
| **`--private` help text** | Done | Already shows `(default: true, use --private=false for public)` |
| **Token scope detection** | Planned | Detect 403 on private repo operations and suggest checking token scope (`repo` vs `public_repo`) |

## TUI

| Feature | Status | Description |
|---------|--------|-------------|
| **Error behavior docs** | Done | Added Error Handling section to docs/guides/hub.md covering all failure modes |

## Documentation

| Feature | Status | Description |
|---------|--------|-------------|
| **catalog.yml editing guide** | Done | Added "Editing catalog.yml Directly" section to docs/reference/architecture.md |
| **Demo GIF** | Planned | Asciinema or GIF recording of the TUI for README — highest-leverage visibility improvement |

## Distribution

| Feature | Status | Description |
|---------|--------|-------------|
| **AUR package** | Planned | Arch Linux AUR package. Requires Arch access for account creation CAPTCHA. Low maintenance once set up |
| **Nix flake** | Planned (low priority) | `nix profile install` support. Moderate effort |
| **Chocolatey** | Planned (low priority) | Windows enterprise package manager. Low priority given winget + Scoop coverage |

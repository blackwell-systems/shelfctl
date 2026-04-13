# Roadmap

## CLI

| Feature | Status | Description |
|---------|--------|-------------|
| **`shelfctl doctor`** | Planned | Check prerequisites (Git, poppler), token validity, shelf connectivity — single command for "is my setup working?" |
| **`--private` help text** | Planned | Update `init --private` help to show `(default: true, use --private=false for public)` |
| **Token scope detection** | Planned | Detect 403 on private repo operations and suggest checking token scope (`repo` vs `public_repo`) |

## TUI

| Feature | Status | Description |
|---------|--------|-------------|
| **Error behavior docs** | Planned | Document what happens in the TUI when downloads fail, GitHub times out, or connectivity drops — add section to docs/guides/hub.md |

## Documentation

| Feature | Status | Description |
|---------|--------|-------------|
| **catalog.yml editing guide** | Planned | Document that advanced users can edit catalog.yml directly — add to docs/reference/architecture.md |
| **Demo GIF** | Planned | Asciinema or GIF recording of the TUI for README — highest-leverage visibility improvement |

## Distribution

| Feature | Status | Description |
|---------|--------|-------------|
| **AUR package** | Planned | Arch Linux AUR package. Requires Arch access for account creation CAPTCHA. Low maintenance once set up |
| **Nix flake** | Planned (low priority) | `nix profile install` support. Moderate effort |
| **Chocolatey** | Planned (low priority) | Windows enterprise package manager. Low priority given winget + Scoop coverage |

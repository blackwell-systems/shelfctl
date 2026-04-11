# Distribution Strategy

This document covers all active and planned distribution channels for shelfctl.

---

## Active Channels

### Homebrew Tap (macOS/Linux)
**Status:** Active — manual update required after each release

Users install via:
```bash
brew install blackwell-systems/tap/shelfctl
```

Formula lives in [blackwell-systems/homebrew-tap](https://github.com/blackwell-systems/homebrew-tap).
Currently updated manually by copying new version, URLs, and SHA256 hashes after each GitHub release.

**Known gap:** The GoReleaser `brews:` config is disabled due to requiring a cross-repo PAT.
Automating this is the highest-priority distribution improvement — it eliminates a manual step
on every release.

**Automation path:** Create a `HOMEBREW_TAP_TOKEN` secret with `repo` scope on `homebrew-tap`,
then uncomment the `brews:` block in `.goreleaser.yml`.

---

### Install Script (Linux/macOS)
**Status:** Active — added in v0.4.4

```bash
curl -fsSL https://raw.githubusercontent.com/blackwell-systems/shelfctl/main/install.sh | sh
```

Installs the latest release to `/usr/local/bin`. Supports overrides:
- `INSTALL_DIR=~/bin` — custom install location
- `VERSION=v0.4.4` — pin a specific version

Script at [`install.sh`](../install.sh). Features:
- OS and architecture auto-detection (Darwin/Linux × amd64/arm64)
- SHA256 checksum verification against `checksums.txt`
- Graceful `sudo` fallback if install dir is not writable

---

### Go Install
**Status:** Active — available since initial release

```bash
go install github.com/blackwell-systems/shelfctl/cmd/shelfctl@latest
```

Works for any Go user. Documented in release headers and README.

---

### GitHub Releases (direct binary download)
**Status:** Active — automated via GoReleaser on every tag push

Binaries published for all six targets:

| Platform | Architecture | Format |
|----------|-------------|--------|
| macOS    | arm64       | .tar.gz |
| macOS    | x86_64      | .tar.gz |
| Linux    | arm64       | .tar.gz |
| Linux    | x86_64      | .tar.gz |
| Windows  | arm64       | .zip |
| Windows  | x86_64      | .zip |

SHA256 checksums published as `checksums.txt` alongside binaries.

---

### Windows Package Manager (winget)
**Status:** Submission in progress — v0.4.4 manifests prepared

Manifests live in [`winget/`](../winget/). Once accepted, users install via:
```bash
winget install BlackwellSystems.shelfctl
```

Package identifier: `BlackwellSystems.shelfctl`

Manifest schema: 1.12.0. Validated on every release via `validate-winget` CI job
(`.github/workflows/release.yml`) running `winget validate` on a `windows-latest` runner.

**Future:** Automate PR submission using the `winget-releaser` GitHub Action with a
`WINGET_TOKEN` PAT (same PAT setup as Homebrew automation).

---

## Proposed Channels

### Homebrew Automation (highest priority)
Eliminate the manual formula update on every release. Requires a one-time PAT setup.
See "Known gap" under Homebrew Tap above.

### winget Automation
After initial manual submission is accepted, automate future updates using
[`vedantmgoyal9/winget-releaser`](https://github.com/vedantmgoyal9/winget-releaser)
in the release workflow. Requires a `WINGET_TOKEN` PAT with `public_repo` scope.

### AUR (Arch Linux)
A community-maintained AUR package would serve Arch Linux users. Low effort to create
(`PKGBUILD` pointing at GitHub release tarball), but requires an AUR account and ongoing
maintenance. Low priority until Arch user demand is confirmed.

### Nix Flake
A `flake.nix` would make shelfctl installable via `nix profile install`. Moderate effort.
Low priority unless Nix users request it.

---

## Release Workflow Summary

Each release currently involves:

1. Tag pushed to `main` → GoReleaser builds all 6 targets and publishes GitHub Release
2. `validate-winget` CI job validates `winget/` manifests on Windows runner
3. **Manual:** Homebrew formula updated in `homebrew-tap` repo
4. **Manual:** winget PR submitted to `microsoft/winget-pkgs` (until automated)
5. Binary rebuilt locally with `make install` to stamp the new version

When Homebrew and winget automation are in place, steps 3 and 4 become automatic,
reducing the release process to tag + push.

---

## Manifest / Formula Reference

| Channel | Location | Updated |
|---------|----------|---------|
| Homebrew formula | `github.com/blackwell-systems/homebrew-tap/Formula/shelfctl.rb` | Manual |
| winget manifests | `winget/` in this repo | Per release |
| Install script | `install.sh` in this repo | Rarely (URL convention stable) |
| GoReleaser config | `.goreleaser.yml` | Rarely |

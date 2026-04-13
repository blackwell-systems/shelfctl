# Distribution Strategy

This document covers all active and planned distribution channels for shelfctl.

---

## Active Channels

### Homebrew Tap (macOS/Linux)
**Status:** Active — automated via GoReleaser on every release

Users install via:
```bash
brew install blackwell-systems/tap/shelfctl
```

Formula lives in [blackwell-systems/homebrew-tap](https://github.com/blackwell-systems/homebrew-tap).
GoReleaser automatically pushes an updated `Formula/shelfctl.rb` on every tag push
using the `HOMEBREW_TAP_TOKEN` secret (classic PAT with `repo` scope).

---

### Scoop Bucket (Windows)
**Status:** Active — automated via GoReleaser on every release

```powershell
scoop bucket add blackwell-systems https://github.com/blackwell-systems/scoop-bucket
scoop install shelfctl
```

Bucket lives at [blackwell-systems/scoop-bucket](https://github.com/blackwell-systems/scoop-bucket).
GoReleaser automatically pushes an updated `bucket/shelfctl.json` manifest on every tag push
using the same `HOMEBREW_TAP_TOKEN` secret.

---

### Windows Package Manager (winget)
**Status:** Active — automated via `winget-releaser` GitHub Action on every release

```powershell
winget install BlackwellSystems.shelfctl
```

Package identifier: `BlackwellSystems.shelfctl`. Manifest schema: 1.12.0.

On every release, the `winget-releaser` action automatically generates manifests from
the published release assets and submits a PR to `microsoft/winget-pkgs` using the
`HOMEBREW_TAP_TOKEN` secret. Local manifests in [`winget/`](../winget/) are validated
on every push via the `validate-winget` CI job.

**Pending PRs:**
- [microsoft/winget-pkgs#358438](https://github.com/microsoft/winget-pkgs/pull/358438) — Add BlackwellSystems.shelfctl v0.4.5 (opened 2026-04-11, validation passed, awaiting merge)

---

### Install Script (Linux/macOS)
**Status:** Active — added in v0.4.4

```bash
curl -fsSL https://raw.githubusercontent.com/blackwell-systems/shelfctl/main/install.sh | sh
```

Installs the latest release to `/usr/local/bin`. Supports overrides:
- `INSTALL_DIR=~/bin` — custom install location
- `VERSION=v0.4.5` — pin a specific version

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

## Proposed Channels

### AUR (Arch Linux)
A community-maintained AUR package would serve Arch Linux users. Low effort to create
(`PKGBUILD` pointing at GitHub release tarball), but requires an AUR account and ongoing
maintenance. Low priority until Arch user demand is confirmed.

### Nix Flake
A `flake.nix` would make shelfctl installable via `nix profile install`. Moderate effort.
Low priority unless Nix users request it.

### Chocolatey (Windows)
An older but widely-used Windows package manager, particularly in enterprise environments.
Low priority given winget and Scoop coverage.

---

## Release Workflow Summary

Each release involves only:

1. Tag pushed to `main` → GoReleaser builds all 6 targets and publishes GitHub Release
2. GoReleaser automatically pushes updated formula to `homebrew-tap`
3. GoReleaser automatically pushes updated manifest to `scoop-bucket`
4. `winget-releaser` action automatically submits PR to `microsoft/winget-pkgs`
5. `validate-winget` CI job validates `winget/` manifests on Windows runner
6. Binary rebuilt locally with `make install` to stamp the new version

The release process is fully automated — tag + push is all that's needed.

---

## Manifest / Formula Reference

| Channel | Location | Updated |
|---------|----------|---------|
| Homebrew formula | `github.com/blackwell-systems/homebrew-tap/Formula/shelfctl.rb` | Automated |
| Scoop manifest | `github.com/blackwell-systems/scoop-bucket/bucket/shelfctl.json` | Automated |
| winget manifests | `winget/` in this repo | Automated via winget-releaser |
| Install script | `install.sh` in this repo | Rarely (URL convention stable) |
| GoReleaser config | `.goreleaser.yml` | Rarely |

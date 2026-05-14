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

### PyPI (pip)
**Status:** Active — automated via GitHub Actions on every release

```bash
pip install shelfctl
```

Platform-specific wheels for macOS (arm64, x86_64), Linux (arm64, x86_64), and Windows (arm64, x86_64).
Each wheel contains the pre-built Go binary with a thin Python entry point that exec's it. No Python
runtime needed at execution time.

The `pypi` job in the release workflow downloads GoReleaser archives from the GitHub Release,
wraps each into a platform wheel via `python/_build_wheels.py`, and publishes to PyPI using
the `PYPI_API_TOKEN` secret.

> **Note:** This is a binary distribution. The Python package exists only as a delivery
> mechanism for the Go binary, similar to how ruff and pyright distribute via pip.

---

### Go Install / pkg.go.dev
**Status:** Active — available since initial release

```bash
go install github.com/blackwell-systems/shelfctl/cmd/shelfctl@latest
```

Works for any Go user. Documented in release headers and README. The module is also
automatically indexed at [pkg.go.dev/github.com/blackwell-systems/shelfctl](https://pkg.go.dev/github.com/blackwell-systems/shelfctl)
via the Go module proxy — no action needed on release.

---

### APT / RPM Packages (Debian, Ubuntu, Fedora, RHEL)
**Status:** Active — automated via GoReleaser nFPM on every release

`.deb` and `.rpm` packages are published as GitHub Release assets alongside the tarballs.

```bash
# Debian/Ubuntu
curl -LO https://github.com/blackwell-systems/shelfctl/releases/latest/download/shelfctl_0.X.Y_linux_amd64.deb
sudo dpkg -i shelfctl_0.X.Y_linux_amd64.deb

# Fedora/RHEL
curl -LO https://github.com/blackwell-systems/shelfctl/releases/latest/download/shelfctl_0.X.Y_linux_amd64.rpm
sudo rpm -i shelfctl_0.X.Y_linux_amd64.rpm
```

GoReleaser's [`nfpms`](https://goreleaser.com/customization/nfpm/) section builds the packages
using nFPM — no `dpkg-deb` or `rpmbuild` required on the CI runner. The binary is installed to
`/usr/bin/shelfctl`; docs and license go to `/usr/share/doc/shelfctl/`.

> **Note:** These are standalone packages downloaded from GitHub Releases, not a hosted APT/RPM
> repository. Users download and install manually rather than using `apt install shelfctl`.
> Hosting a proper repo (via Packagecloud, Cloudsmith, etc.) is a possible future enhancement.

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

Each archive includes `README.md`, `LICENSE`, and the user-facing docs (`docs/guides/`,
`docs/reference/`). The `.goreleaser.yml` `archives.files` glob controls what's bundled.

**GoReleaser release notes** are auto-generated from commit messages since the previous
tag, with `docs:`, `test:`, `ci:`, `chore:`, and `typo` commits filtered out. This is
separate from `CHANGELOG.md` — the GitHub Release shows the commit-derived notes; the
release footer links to `CHANGELOG.md` for the full human-written entry.

**`prerelease: auto`** — tags containing a `-` (e.g. `v0.5.0-beta.1`) are automatically
marked as pre-releases on GitHub. Use a clean `vMAJOR.MINOR.PATCH` tag for a stable release.

**`mode: replace`** — re-pushing an existing tag (e.g. to fix a bad binary) replaces the
GitHub Release and re-publishes updated Homebrew/Scoop manifests. winget will submit a
new PR for the same version.

---

## Proposed Channels

### AUR (Arch Linux)
A community-maintained AUR package would serve Arch Linux users. Low effort to create
(`PKGBUILD` pointing at GitHub release tarball), but requires an AUR account and ongoing
maintenance. Low priority until Arch user demand is confirmed.

### Nix Flake
A `flake.nix` would make shelfctl installable via `nix profile install`. Moderate effort.
Low priority unless Nix users request it.


---

## Release Workflow Summary

Each release involves only:

1. Update `CHANGELOG.md` with the release entry, commit to `main`
2. Push a version tag: `git tag v0.X.Y && git push origin v0.X.Y`
3. GoReleaser builds all 6 targets, `.deb`/`.rpm` packages, and publishes the GitHub Release
4. GoReleaser pushes updated formula to `homebrew-tap`
5. GoReleaser pushes updated manifest to `scoop-bucket`
6. `winget-releaser` action submits PR to `microsoft/winget-pkgs` (requires Microsoft reviewer approval — typically takes a few days)
7. `validate-winget` CI job validates `winget/` manifests on Windows runner
8. `pypi` job builds platform wheels from release archives and publishes to PyPI

Steps 3–8 are fully automated. The only manual steps are updating the changelog and pushing the tag.

> **Local install (optional):** Run `make install` after releasing to update the binary on your own machine. This is a developer convenience — it has no effect on the published release.

---

## Manifest / Formula Reference

| Channel | Location | Updated |
|---------|----------|---------|
| Homebrew formula | `github.com/blackwell-systems/homebrew-tap/Formula/shelfctl.rb` | Automated |
| Scoop manifest | `github.com/blackwell-systems/scoop-bucket/bucket/shelfctl.json` | Automated |
| winget manifests | `winget/` in this repo | Automated via winget-releaser |
| Install script | `install.sh` in this repo | Rarely (URL convention stable) |
| nFPM (deb/rpm) | `.goreleaser.yml` `nfpms` section | Automated |
| PyPI wheels | `python/_build_wheels.py` | Automated |
| GoReleaser config | `.goreleaser.yml` | Rarely |

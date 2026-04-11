#!/bin/sh
# shelfctl installer
# Usage: curl -fsSL https://raw.githubusercontent.com/blackwell-systems/shelfctl/main/install.sh | sh
#
# Environment overrides:
#   INSTALL_DIR   destination directory (default: /usr/local/bin)
#   VERSION       specific version to install, e.g. v0.4.4 (default: latest)

set -e

REPO="blackwell-systems/shelfctl"
BINARY="shelfctl"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# ── Detect OS ────────────────────────────────────────────────────────────────

OS=$(uname -s)
case "$OS" in
  Linux*)  OS="Linux"  ;;
  Darwin*) OS="Darwin" ;;
  *)
    echo "Error: unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# ── Detect architecture ───────────────────────────────────────────────────────

ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64)   ARCH="x86_64" ;;
  arm64|aarch64)  ARCH="arm64"  ;;
  *)
    echo "Error: unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# ── Resolve version ───────────────────────────────────────────────────────────

if [ -z "$VERSION" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
fi

if [ -z "$VERSION" ]; then
  echo "Error: failed to fetch latest version from GitHub API" >&2
  exit 1
fi

VERSION_NUM="${VERSION#v}"
FILENAME="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

echo "Installing ${BINARY} ${VERSION} (${OS}/${ARCH}) → ${INSTALL_DIR}/${BINARY}"

# ── Download ──────────────────────────────────────────────────────────────────

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT INT TERM

curl -fsSL "${BASE_URL}/${FILENAME}"      -o "${TMP}/${FILENAME}"
curl -fsSL "${BASE_URL}/checksums.txt"    -o "${TMP}/checksums.txt"

# ── Verify checksum ───────────────────────────────────────────────────────────

cd "$TMP"
CHECKSUM_LINE=$(grep "$FILENAME" checksums.txt || true)

if [ -z "$CHECKSUM_LINE" ]; then
  echo "Warning: checksum entry not found for ${FILENAME}, skipping verification" >&2
elif command -v sha256sum > /dev/null 2>&1; then
  echo "$CHECKSUM_LINE" | sha256sum --check --status || {
    echo "Error: checksum verification failed" >&2
    exit 1
  }
elif command -v shasum > /dev/null 2>&1; then
  echo "$CHECKSUM_LINE" | shasum -a 256 --check --status || {
    echo "Error: checksum verification failed" >&2
    exit 1
  }
else
  echo "Warning: neither sha256sum nor shasum found, skipping checksum verification" >&2
fi

# ── Extract and install ───────────────────────────────────────────────────────

tar -xzf "$FILENAME" "$BINARY"

if [ -w "$INSTALL_DIR" ]; then
  mv "$BINARY" "${INSTALL_DIR}/${BINARY}"
else
  echo "  (${INSTALL_DIR} is not writable, trying sudo)"
  sudo mv "$BINARY" "${INSTALL_DIR}/${BINARY}"
fi

chmod +x "${INSTALL_DIR}/${BINARY}"

echo ""
echo "✓ ${BINARY} ${VERSION} installed to ${INSTALL_DIR}/${BINARY}"
echo "  Run '${BINARY} version' to verify"

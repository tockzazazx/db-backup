#!/usr/bin/env bash
# Install the latest boxdb release to /usr/local/bin.
# Usage: wget -qO- https://github.com/tockzazazx/db-backup/releases/latest/download/install.sh | bash
set -euo pipefail

REPO="tockzazazx/db-backup"

ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH=amd64 ;;
  aarch64 | arm64) ARCH=arm64 ;;
  *)
    echo "unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

URL="https://github.com/$REPO/releases/latest/download/boxdb_linux_$ARCH"
TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT

echo "Downloading $URL"
if command -v wget > /dev/null; then
  wget -qO "$TMP" "$URL"
else
  curl -fsSL -o "$TMP" "$URL"
fi

SUDO=""
if [ "$(id -u)" -ne 0 ]; then
  SUDO="sudo"
fi
$SUDO install -m 755 "$TMP" /usr/local/bin/boxdb

echo "Installed: $(boxdb --version)"

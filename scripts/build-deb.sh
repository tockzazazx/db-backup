#!/usr/bin/env bash
# Build a .deb package from a prebuilt linux binary in bin/.
# Usage: ./scripts/build-deb.sh <amd64|arm64> <version>
set -euo pipefail

ARCH=${1:?usage: build-deb.sh <amd64|arm64> <version>}
VERSION=${2:?usage: build-deb.sh <amd64|arm64> <version>}
DEBVER=${VERSION#v}

ROOT="dist/deb/$ARCH"
rm -rf "$ROOT"
mkdir -p "$ROOT/DEBIAN" "$ROOT/usr/bin"

install -m 755 "bin/boxdb_linux_$ARCH" "$ROOT/usr/bin/boxdb"

cat > "$ROOT/DEBIAN/control" <<EOF
Package: boxdb
Version: $DEBVER
Section: utils
Priority: optional
Architecture: $ARCH
Maintainer: Punnawish <tock.nuwrii@gmail.com>
Homepage: https://github.com/tockzazazx/db-backup
Description: Database backup CLI tool
 boxdb backs up databases to local files.
EOF

mkdir -p dist
dpkg-deb --build --root-owner-group "$ROOT" "dist/boxdb_${DEBVER}_${ARCH}.deb"

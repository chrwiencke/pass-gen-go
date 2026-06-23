#!/usr/bin/env bash
set -euo pipefail

# fyne.io/systray requires CGO, so macOS and Windows builds should be built
# on their native OS or with a configured C cross-compiler.
./scripts/build-macos.sh

ARCH="${GOARCH:-amd64}"
if command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
  mkdir -p "dist/windows-${ARCH}"
  CGO_ENABLED=1 GOOS=windows GOARCH="${ARCH}" CC=x86_64-w64-mingw32-gcc \
    go build -trimpath -ldflags="-H=windowsgui -s -w" \
    -o "dist/windows-${ARCH}/gopass.exe" ./cmd/gopass
  echo "Built dist/windows-${ARCH}/gopass.exe"
else
  echo "Skipped Windows build: fyne.io/systray requires CGO and x86_64-w64-mingw32-gcc was not found."
  echo "Use GitHub Actions or run scripts/build-windows.ps1 on Windows with MSYS2/MinGW-w64 installed."
fi

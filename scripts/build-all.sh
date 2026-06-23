#!/usr/bin/env bash
set -euo pipefail

# fyne.io/systray requires CGO, so macOS and Windows builds should be built
# on their native OS or with a configured C cross-compiler.
./scripts/build-macos.sh

ARCH="${GOARCH:-amd64}"
VERSION="${VERSION:-1.0.0}"
if command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
  mkdir -p "dist/windows-${ARCH}"
  ASSET_NAME="gopass-windows-${ARCH}.exe"
  ASSET_PATH="dist/windows-${ARCH}/${ASSET_NAME}"
  CGO_ENABLED=1 GOOS=windows GOARCH="${ARCH}" CC=x86_64-w64-mingw32-gcc \
    go build -trimpath -ldflags="-H=windowsgui -s -w -X main.version=${VERSION}" \
    -o "${ASSET_PATH}" ./cmd/gopass
  cp "${ASSET_PATH}" "dist/windows-${ARCH}/gopass.exe"
  checksum="$(shasum -a 256 "${ASSET_PATH}" | awk '{print $1}')"
  printf "%s  %s\n" "${checksum}" "${ASSET_NAME}" > "${ASSET_PATH}.sha256"
  echo "Built dist/windows-${ARCH}/gopass.exe"
  echo "Release asset: ${ASSET_PATH}"
else
  echo "Skipped Windows build: fyne.io/systray requires CGO and x86_64-w64-mingw32-gcc was not found."
  echo "Use GitHub Actions or run scripts/build-windows.ps1 on Windows with MSYS2/MinGW-w64 installed."
fi

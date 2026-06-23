#!/usr/bin/env bash
set -euo pipefail

# Build the current Mac architecture by default.
./scripts/build-macos.sh

# Build a Windows exe. Override with GOARCH=arm64 ./scripts/build-all.sh if needed.
ARCH="${GOARCH:-amd64}"
mkdir -p "dist/windows-${ARCH}"
CGO_ENABLED=0 GOOS=windows GOARCH="${ARCH}" \
  go build -trimpath -ldflags="-H=windowsgui -s -w" \
  -o "dist/windows-${ARCH}/gopass.exe" ./cmd/gopass

echo "Built dist/windows-${ARCH}/gopass.exe"

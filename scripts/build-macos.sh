#!/usr/bin/env bash
set -euo pipefail

APP_NAME="GoPass"
BIN_NAME="GoPass"
ARCH="${GOARCH:-$(go env GOARCH)}"
VERSION="${VERSION:-0.1.0}"
PLIST_VERSION="${VERSION%%-*}"
BUNDLE_VERSION="${GITHUB_RUN_NUMBER:-1}"

DIST_DIR="dist/macos-${ARCH}"
APP_DIR="${DIST_DIR}/${APP_NAME}.app"

rm -rf "${DIST_DIR}"
mkdir -p "${APP_DIR}/Contents/MacOS" "${APP_DIR}/Contents/Resources"

CGO_ENABLED=1 GOOS=darwin GOARCH="${ARCH}" \
  go build -trimpath \
  -ldflags="-s -w -X main.version=${VERSION}" \
  -o "${APP_DIR}/Contents/MacOS/${BIN_NAME}" ./cmd/gopass

cat > "${APP_DIR}/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleExecutable</key>
  <string>${BIN_NAME}</string>
  <key>CFBundleIdentifier</key>
  <string>local.gopass.tray</string>
  <key>CFBundleName</key>
  <string>${APP_NAME}</string>
  <key>CFBundleDisplayName</key>
  <string>${APP_NAME}</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>${PLIST_VERSION}</string>
  <key>CFBundleVersion</key>
  <string>${BUNDLE_VERSION}</string>
  <key>LSMinimumSystemVersion</key>
  <string>12.0</string>
  <key>LSUIElement</key>
  <string>1</string>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
PLIST

chmod +x "${APP_DIR}/Contents/MacOS/${BIN_NAME}"
echo "Built ${APP_DIR}"
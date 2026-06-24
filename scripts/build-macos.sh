#!/usr/bin/env bash
set -euo pipefail

APP_NAME="GoPass"
BIN_NAME="gopass"
ARCH="${GOARCH:-$(go env GOARCH)}"
VERSION="${VERSION:-1.0.0}"
CODESIGN_IDENTITY="${MACOS_CODESIGN_IDENTITY:--}"
DIST_DIR="dist/macos-${ARCH}"
APP_DIR="${DIST_DIR}/${APP_NAME}.app"
ASSET_NAME="${BIN_NAME}-darwin-${ARCH}"
ASSET_PATH="${DIST_DIR}/${ASSET_NAME}"
LD_FLAGS="-s -w -X main.version=${VERSION}"

rm -rf "${DIST_DIR}"
mkdir -p "${APP_DIR}/Contents/MacOS" "${APP_DIR}/Contents/Resources"

CGO_ENABLED=1 GOOS=darwin GOARCH="${ARCH}" \
  go build -trimpath -ldflags="${LD_FLAGS}" \
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
  <string>${VERSION}</string>
  <key>CFBundleVersion</key>
  <string>${VERSION}</string>
  <key>LSMinimumSystemVersion</key>
  <string>12.0</string>
  <key>LSUIElement</key>
  <true/>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
PLIST

chmod +x "${APP_DIR}/Contents/MacOS/${BIN_NAME}"

codesign_args=(--force --deep --sign "${CODESIGN_IDENTITY}")
if [[ "${CODESIGN_IDENTITY}" == "-" ]]; then
  # macOS on Apple Silicon requires executable code inside app bundles to have
  # a valid code signature. This ad-hoc signature does not remove Gatekeeper's
  # unidentified-developer warning, but it prevents the misleading "app is
  # damaged" error for unsigned CI builds.
  codesign_args+=(--timestamp=none)
else
  codesign_args+=(--options runtime --timestamp)
fi

codesign "${codesign_args[@]}" "${APP_DIR}"
codesign --verify --deep --strict --verbose=2 "${APP_DIR}"

# Copy the signed executable after codesign has embedded its Mach-O signature.
# Self-updates use this asset directly, so copying before signing would force
# the updater to repair the bundle with a new local ad-hoc signature.
cp "${APP_DIR}/Contents/MacOS/${BIN_NAME}" "${ASSET_PATH}"
checksum="$(shasum -a 256 "${ASSET_PATH}" | awk '{print $1}')"
printf "%s  %s\n" "${checksum}" "${ASSET_NAME}" > "${ASSET_PATH}.sha256"

echo "Built and signed ${APP_DIR} with identity ${CODESIGN_IDENTITY}"
echo "Release asset: ${ASSET_PATH}"

#!/usr/bin/env bash
set -euo pipefail

APP_NAME="GoPass"
ARCH="${GOARCH:-$(go env GOARCH)}"
VERSION="${VERSION:-1.0.0}"
DIST_DIR="dist/macos-${ARCH}"
APP_DIR="${DIST_DIR}/${APP_NAME}.app"
STAGING_DIR="${DIST_DIR}/dmg-root"
DMG_NAME="${APP_NAME}-macos-${ARCH}.dmg"
DMG_PATH="dist/${DMG_NAME}"

if ! command -v hdiutil >/dev/null 2>&1; then
  echo "hdiutil was not found. DMG packaging must run on macOS." >&2
  exit 1
fi

if [[ ! -d "${APP_DIR}" ]]; then
  VERSION="${VERSION}" ./scripts/build-macos.sh
fi

rm -rf "${STAGING_DIR}" "${DMG_PATH}" "${DMG_PATH}.sha256"
mkdir -p "${STAGING_DIR}"

ditto "${APP_DIR}" "${STAGING_DIR}/${APP_NAME}.app"
ln -s /Applications "${STAGING_DIR}/Applications"

hdiutil create \
  -volname "${APP_NAME}" \
  -srcfolder "${STAGING_DIR}" \
  -ov \
  -format UDZO \
  "${DMG_PATH}"
rm -rf "${STAGING_DIR}"

checksum="$(shasum -a 256 "${DMG_PATH}" | awk '{print $1}')"
printf "%s  %s\n" "${checksum}" "${DMG_NAME}" > "${DMG_PATH}.sha256"

echo "Built ${DMG_PATH}"

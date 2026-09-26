#!/bin/bash
# Install the packaged desktop app without leaving the build artifact registered
# as a second JameClaw Desktop application.

set -euo pipefail

APP_NAME="JameClaw Desktop.app"
SOURCE_APP="$(cd "$(dirname "$0")/.." && pwd)/build/${APP_NAME}"
INSTALL_APP="/Applications/${APP_NAME}"
LSREGISTER="/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"

if [ "$(uname)" != "Darwin" ]; then
    echo "Error: macOS installation is only available on Darwin."
    exit 1
fi

if [ ! -d "$SOURCE_APP" ]; then
    echo "Error: packaged app not found: $SOURCE_APP"
    echo "Run: make build-macos-app"
    exit 1
fi

codesign --verify --deep --strict --verbose=2 "$SOURCE_APP"

# The build output is intentionally not an installed application. Remove any
# prior registration before copying so it cannot compete with /Applications.
if [ -x "$LSREGISTER" ]; then
    "$LSREGISTER" -u "$SOURCE_APP" || true
fi

ditto "$SOURCE_APP" "$INSTALL_APP"
codesign --verify --deep --strict --verbose=2 "$INSTALL_APP"

if [ -x "$LSREGISTER" ]; then
    "$LSREGISTER" -u "$SOURCE_APP" || true
    "$LSREGISTER" -f -R -trusted "$INSTALL_APP"
fi

echo "Installed and registered: $INSTALL_APP"

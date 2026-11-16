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

# Helpers moved from Contents/MacOS to Contents/Helpers. ditto intentionally
# merges bundles, so remove only the two legacy paths before copying or they
# remain as unsigned additions to the new sealed bundle.
rm -f "$INSTALL_APP/Contents/MacOS/jameclaw" "$INSTALL_APP/Contents/MacOS/jameclaw-launcher"
ditto "$SOURCE_APP" "$INSTALL_APP"
# ditto merges into an existing install without updating the bundle's own
# mtime, so Finder and the Dock keep showing a cached icon. Bump it.
touch "$INSTALL_APP" "$INSTALL_APP/Contents/Info.plist"
codesign --verify --deep --strict --verbose=2 "$INSTALL_APP"

if [ -x "$LSREGISTER" ]; then
    "$LSREGISTER" -u "$SOURCE_APP" || true
    "$LSREGISTER" -f -R -trusted "$INSTALL_APP"
fi

# Unregistering alone is not enough: Finder and Spotlight re-register any
# .app they see, so a leftover build copy shows up as a second JameClaw
# Desktop. Remove the build copy and older duplicate bundles once installed.
BUILD_DIR="$(dirname "$SOURCE_APP")"
for stale in "$SOURCE_APP" "${BUILD_DIR}/JameClaw Desktop.bundle" "${BUILD_DIR}/Jame.app"; do
    if [ -e "$stale" ]; then
        [ -x "$LSREGISTER" ] && { "$LSREGISTER" -u "$stale" || true; }
        rm -rf "$stale"
    fi
done

# Dock can retain installed copies in its recent-apps list. Remove JameClaw
# recent tiles during installation; the running app supplies its one real Dock
# icon, while unrelated recent applications remain untouched.
DOCK_PLIST="$HOME/Library/Preferences/com.apple.dock.plist"
if [ -f "$DOCK_PLIST" ]; then
    RECENT_COUNT="$(plutil -extract recent-apps raw -o - "$DOCK_PLIST" 2>/dev/null || echo 0)"
    DOCK_CHANGED=0
    for ((INDEX=RECENT_COUNT-1; INDEX>=0; INDEX--)); do
        IDENTIFIER="$(/usr/libexec/PlistBuddy -c "Print :recent-apps:${INDEX}:tile-data:bundle-identifier" "$DOCK_PLIST" 2>/dev/null || true)"
        if [ "$IDENTIFIER" = "com.jameclaw.launcher" ]; then
            /usr/libexec/PlistBuddy -c "Delete :recent-apps:${INDEX}" "$DOCK_PLIST"
            DOCK_CHANGED=1
        fi
    done
    if [ "$DOCK_CHANGED" -eq 1 ]; then
        killall cfprefsd 2>/dev/null || true
        killall Dock 2>/dev/null || true
    fi
fi

echo "Installed and registered: $INSTALL_APP"

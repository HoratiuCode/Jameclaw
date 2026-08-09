#!/bin/bash
set -euo pipefail

TARGET_APP=${1:-"./build/Jame.app"}
SOURCE="./macos/JameClawHome/JameClawHome.swift"
PLIST="./macos/JameClawHome/Info.plist"
MACOS_DIR="$TARGET_APP/Contents/MacOS"
RESOURCES_DIR="$TARGET_APP/Contents/Resources"

rm -rf "$TARGET_APP"
mkdir -p "$MACOS_DIR"
mkdir -p "$RESOURCES_DIR"
cp "$PLIST" "$TARGET_APP/Contents/Info.plist"
cp "./scripts/icon.icns" "$RESOURCES_DIR/icon.icns"
cp "./macos/JameClawHome/creation-of-adam.jpg" "$RESOURCES_DIR/creation-of-adam.jpg"
swiftc -parse-as-library "$SOURCE" -o "$MACOS_DIR/Jame" -framework SwiftUI -framework AppKit -framework UserNotifications

# swiftc leaves only a linker signature on the executable. Sign the complete
# app bundle after its plist and resources exist so Launch Services can reopen
# Jame reliably after a rebuild.
codesign --force --deep --sign - "$TARGET_APP"

#!/bin/bash
set -euo pipefail

TARGET_APP=${1:-"./build/JameClaw Settings.app"}
SOURCE="./macos/JameClawSettings/JameClawSettings.swift"
PLIST="./macos/JameClawSettings/Info.plist"
MACOS_DIR="$TARGET_APP/Contents/MacOS"
RESOURCES_DIR="$TARGET_APP/Contents/Resources"

rm -rf "$TARGET_APP"
mkdir -p "$MACOS_DIR"
mkdir -p "$RESOURCES_DIR"
cp "$PLIST" "$TARGET_APP/Contents/Info.plist"
cp "./scripts/icon.icns" "$RESOURCES_DIR/icon.icns"
swiftc -parse-as-library "$SOURCE" -o "$MACOS_DIR/JameClaw Settings" -framework SwiftUI -framework AppKit

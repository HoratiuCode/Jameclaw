#!/bin/bash
set -euo pipefail

TARGET_APP=${1:-"./build/Jame.app"}
SOURCE="./macos/JameClawHome/JameClawHome.swift"
PLIST="./macos/JameClawHome/Info.plist"
MACOS_DIR="$TARGET_APP/Contents/MacOS"

rm -rf "$TARGET_APP"
mkdir -p "$MACOS_DIR"
cp "$PLIST" "$TARGET_APP/Contents/Info.plist"
swiftc -parse-as-library "$SOURCE" -o "$MACOS_DIR/Jame" -framework SwiftUI -framework AppKit

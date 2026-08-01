#!/bin/bash
# Build macOS .app bundle for JameClaw Desktop

set -e

EXECUTABLE=$1

if [ -z "$EXECUTABLE" ]; then
    echo "Usage: $0 <executable>"
    exit 1
fi

echo "executable: $EXECUTABLE"

APP_NAME="JameClaw Desktop"
APP_PATH="./build/${APP_NAME}.app"
APP_CONTENTS="${APP_PATH}/Contents"
APP_MACOS="${APP_CONTENTS}/MacOS"
APP_RESOURCES="${APP_CONTENTS}/Resources"
APP_EXECUTABLE="jameclaw-launcher"
ICON_SOURCE="./scripts/icon.icns"

# Clean up existing .app
if [ -d "$APP_PATH" ]; then
    echo "Removing existing ${APP_PATH}"
    rm -rf "$APP_PATH"
fi

# Create directory structure
echo "Creating .app bundle structure..."
mkdir -p "$APP_MACOS"
mkdir -p "$APP_RESOURCES"

# Copy executable
echo "Copying executable..."
if [ -f "./web/build/${APP_EXECUTABLE}" ]; then
    cp "./web/build/${APP_EXECUTABLE}" "${APP_MACOS}/"
else
    echo "Error: ./web/build/${APP_EXECUTABLE} not found. Please build the web backend first."
    echo "Run: make build in web dir"
    exit 1
fi
if [ -f "./build/jameclaw" ]; then
    cp "./build/jameclaw" "${APP_MACOS}/"
else
    echo "Error: ./build/jameclaw not found. Please build the main file first."
    echo "Run: make build"
    exit 1
fi
chmod +x "${APP_MACOS}/"*

# Create Info.plist
echo "Creating Info.plist..."
cat > "${APP_CONTENTS}/Info.plist" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>jameclaw-launcher</string>
    <key>CFBundleIdentifier</key>
    <string>com.jameclaw.launcher</string>
    <key>CFBundleName</key>
    <string>JameClaw Desktop</string>
    <key>CFBundleDisplayName</key>
    <string>JameClaw Desktop</string>
    <key>CFBundleIconFile</key>
    <string>icon</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0</string>
    <key>CFBundleVersion</key>
    <string>1</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSSupportsAutomaticGraphicsSwitching</key>
    <true/>
    <key>NSUserNotificationUsageDescription</key>
    <string>JameClaw uses notifications to show alerts for agent actions, automation results, and gateway status changes.</string>
    <key>NSScreenCaptureDescription</key>
    <string>JameClaw can capture the screen when you ask an agent to use screenshots as context.</string>
    <key>NSCameraUsageDescription</key>
    <string>JameClaw can access the camera when you ask an agent to capture photos or video context.</string>
    <key>NSMicrophoneUsageDescription</key>
    <string>JameClaw uses the microphone for voice messages and voice-driven agent workflows.</string>
    <key>NSSpeechRecognitionUsageDescription</key>
    <string>JameClaw can use speech recognition when you enable voice-driven workflows.</string>
    <key>NSAppleEventsUsageDescription</key>
    <string>JameClaw uses Automation permission when you ask an agent to control local apps or run macOS workflows.</string>
    <key>NSLocalNetworkUsageDescription</key>
    <string>JameClaw uses the local network to connect the Web Console, gateway, extensions, and local model services.</string>
    <key>NSRemindersUsageDescription</key>
    <string>JameClaw can access Reminders when you ask an agent to work with reminders.</string>
    <key>NSAppTransportSecurity</key>
    <dict>
        <key>NSAllowsArbitraryLoadsInWebContent</key>
        <true/>
    </dict>
</dict>
</plist>
EOF

#sips -z 128 128 "$ICON_SOURCE" --out "${ICONSET_PATH}/icon_128x128.png" > /dev/null 2>&1
#
## Create icns file
#iconutil -c icns "$ICONSET_PATH" -o "$ICON_OUTPUT" 2>/dev/null || {
#    echo "Warning: iconutil failed"
#}

cp $ICON_SOURCE "${APP_RESOURCES}/icon.icns"
bash ./scripts/build-macos-settings-app.sh "${APP_RESOURCES}/JameClaw Settings.app"
bash ./scripts/build-macos-home-app.sh "${APP_RESOURCES}/Jame.app"

# The Go executables carry linker-generated ad-hoc signatures. Embedding the
# Swift child apps after copying those executables changes the resource seal,
# which can make Launch Services report a misleading kLSNoExecutableErr.
# Re-sign the completed nested bundle so the runnable .app opens reliably.
codesign --force --deep --sign - "$APP_PATH"

echo ""
echo "=========================================="
echo "Successfully created: ${APP_PATH}"
echo "=========================================="
echo ""
echo "To launch JameClaw:"
echo "  1. Double-click ${APP_NAME}.app in Finder"
echo "  2. Or use: open ${APP_PATH}"
echo ""
echo "Note: The app will run in the menu bar (systray) without a terminal window."
echo ""

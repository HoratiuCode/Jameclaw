# JameClaw Desktop 908M

Date: 2026-08-09

## Purpose

`908M` records the current uncommitted JameClaw Desktop improvements made in
this checkout. It is a local change record, not a Git release or a pushed
version.

## Desktop reliability and single-app behavior

- The top-level `JameClaw Desktop.app` owns its Go launcher helper through a
  dedicated `NativeLaunchCoordinator`.
- An unexpected launcher exit is recovered automatically with bounded
  exponential backoff (up to three attempts). A 30-second stable run resets
  that retry budget.
- An intentional app quit cancels pending restarts and terminates the helper
  owned by the desktop app.
- The helper runs as a macOS `UIElement`, so the visible app remains the only
  foreground Dock identity.
- The native boot screen now displays the actual connection/recovery state
  while the private localhost gateway starts.

## Chat recovery

- Added an eight-minute response watchdog for provider requests that never
  reach a completion event.
- When the watchdog fires, Chat exits its permanently-thinking state and shows
  the existing `Retry message` and `Continue` recovery actions.
- Switching to a new or resumed chat cancels the old response watchdog.

## Provider model discovery

- Catalog-backed providers can request their live `/models` list through
  `POST /api/models/catalog/discover`.
- The desktop and Web Console can display and select remotely discovered model
  IDs instead of requiring a manually typed model name.
- The stored provider credential remains server-side; discovery does not
  return the API key to the client.
- Nous Research discovery was checked against the saved local credential and
  returned 354 models.

## Greeting routing

- Removed the native hard-coded greeting shortcut that returned:
  `Hello — I’m ready. You can ask me to search files, work in your workspace, or handle a larger task.`
- Greetings such as `hello`, `hi`, and `salut` now use the selected API
  provider and normal chat WebSocket flow, like every other chat request.

## Key files

- `macos/JameClawHome/NativeAppInfrastructure.swift`
- `macos/JameClawHome/JameClawHome.swift`
- `macos/JameClawHome/Info.plist`
- `web/backend/desktop_menu_darwin.m`
- `web/backend/api/models.go`
- `web/backend/api/models_test.go`
- `web/frontend/src/api/models.ts`
- `web/frontend/src/components/models/add-provider-model-sheet.tsx`
- `scripts/build-macos-app.sh`

## Validation recorded

- `swiftc -parse-as-library -typecheck macos/JameClawHome/JameClawHome.swift macos/JameClawHome/NativeAppInfrastructure.swift -framework SwiftUI -framework AppKit -framework UserNotifications` passed.
- `./scripts/build-macos-app.sh jameclaw-darwin-arm64` rebuilt the real top-level desktop bundle.
- `codesign --verify --deep --strict --verbose=2 'build/JameClaw Desktop.app'` passed.
- Clean app launch produced one foreground `JameClaw Desktop` process and one
  owned launcher helper marked `UIElement`.
- In a controlled test, terminating the helper caused the desktop app to start
  a replacement helper and restore the localhost service on port `18800`.
- The authenticated local gateway status endpoint reported `gateway_status` as
  `running`.

## Before committing

```bash
git status --short
git diff --check
./scripts/build-macos-app.sh jameclaw-darwin-arm64
codesign --verify --deep --strict --verbose=2 'build/JameClaw Desktop.app'
```

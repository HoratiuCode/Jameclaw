# JameClaw 108D — Unpushed Git Work

This document records the current local Git state and work that has not been pushed from this checkout as of 2026-08-01.

## Version and branch state

- Working version: `108D`
- Branch: `main`
- Remote: `origin`
- Local HEAD: `71fa048d982ce7f03137163495dde391e779ad8f` (`108D`)
- Remote `origin/main`: `71fa048d982ce7f03137163495dde391e779ad8f` (`108D`)
- Ahead/behind: `0 / 0`

## Unpushed commits

There are currently **zero unpushed Git commits**. The existing `108D` commit is present on both local `main` and `origin/main`.

The work below is **uncommitted local work**. It is not part of Git history and is not pushed until it is reviewed, committed, and pushed.

## Uncommitted 108D design-theme work

The native Desktop Settings window and Web Console Design section now support complete interface themes in addition to individual light/dark and appearance controls.

- Adds seven full-design presets: Jame Dark, Jame Light, Nordic Frost, Sepia Reading, Cyberpunk Neon, Forest Focus, and Sunset Studio.
- Applying a preset updates the color palette, global accent, font family, text size, interface spacing, and corner style together.
- Adds Compact, Comfortable, and Spacious density options that scale spacing across the Web Console.
- Adds Square, Soft, and Rounded corner systems that update the shared radius tokens used by controls, cards, menus, and panels.
- Applies the selected accent to primary actions, focus rings, sidebar selection, charts, chat, and live agent activity.
- Persists density and corner preferences in local storage alongside the existing theme, font, text-size, and accent settings.
- Automatically displays `Custom design` when individual settings no longer match a complete preset.
- Adds the same visible `Full design theme` selector to the native macOS Settings → Design section.
- Native presets apply theme, accent, chat spacing, message surface, text size, window transparency, Team Grid glow, and the default chat background together.
- The native accent now also updates the main window tint, background glow, Settings button, and sidebar navigation.
- Shared native accent tokens now update every supporting page, including Sessions, Memory, Capabilities, Agent, Automations, Artifacts, Chat, and Settings.

## Uncommitted 108D native startup and window work

- Builds the native SwiftUI window directly into the top-level `JameClaw Desktop.app` and signs one application bundle.
- Runs the launcher and gateway as ordinary helper executables in `Contents/MacOS`; there are no nested application bundles.
- Prevents a closed SwiftUI WindowGroup from being restored as a windowless background process.
- Retains and reveals the real main window for Finder, Dock, menu-bar, and notification reopen requests while it is open.
- Terminates the native UI after its last window closes, so reopening starts a fresh visible app while its helper lifecycle remains controlled by the top-level bundle.
- Adds a guarded AppKit fallback window when SwiftUI does not mount its normal window during a cold start.
- Packages `JameClaw Desktop.app` itself as the regular foreground application, preventing a second app named `Jame` from appearing in the Dock or application switcher.
- Bootstraps the AppKit host independently from SwiftUI scene restoration, so an empty restored `WindowGroup` cannot suppress the desktop window.
- Adds a visibility watchdog that repairs the rare SwiftUI launch race where a loaded main window is hidden after the first reveal; minimized windows are left alone.
- Persists the user's preferred window frame independently of SwiftUI's unreliable page-dependent autosave.
- Keeps the same window dimensions when navigating between Chat, Sessions, and every other page.
- Makes Chat's canvas and composer compressible so their natural fitting height cannot enlarge the desktop window.
- Changes native voice input to transcribe into the editable chat box first; recordings are never sent automatically, existing draft text is preserved, and only the normal Send button submits the reviewed transcription.
- Transforms the native Send button into a disabled `...` indicator with a pulsing accent glow while Jame is thinking, then restores the normal Send state when the response finishes.
- Uses one response-in-progress state for the visible `Thinking... 💭` label, the glowing `...` Send control, and active execution plans; removes the execution-plan card as soon as the final response arrives or the task completes.
- Hoists the native chat runtime above page navigation so its WebSocket, messages, execution plan, and thinking state continue while the user opens Sessions, Agent, Settings, or another page.
- Adds a pulsing accent outline, glow, and `...` activity mark around the Sessions navigation row whenever Jame is still working and Chat is not the visible page; the glow stops automatically with the response.

## Uncommitted 108D Team Grid work

- Displays a dedicated model/provider badge on every main, spawned, and independent team-agent card.
- Resolves inherited models to the configured global default, so Hermes and other agents no longer show only their persona while hiding the model they actually use.
- Treats Hermes as its own external runtime: Team Grid reads the non-secret provider/model fields from `~/.hermes/config.yaml`, currently displaying `Nous Research · tencent/hy3:free`, and never mislabels a missing Hermes model as inherited Codex CLI.
- Shows the same resolved runtime model in the selected-agent inspector.
- Shows each agent's highest-priority active delegated task directly on its Team Grid card.
- Adds a right-click delegation menu to every agent card with `Delegate existing task` and `Create task for …` actions.
- Reassigning an existing task uses the real team-operations API, updates `owner_agent_id`, and moves the contract to Planned.
- Creating a task from an agent card opens the real task editor with that agent already selected as owner.
- Adds delegated-task history and status to the selected-agent inspector.
- Adds a live `What JameClaw can modify` area backed by the real workspace restriction, enabled create/edit/append tools, and every configured `allow_write_paths` entry.
- Adds a dedicated Artifacts box showing the real workspace artifact folder, artifact count, kind, and latest saved projects, including an actionable empty state.
- Adds persistent Loop nodes from the Team Grid toolbar or any agent card. Every later flow node automatically depends on the latest Loop and cannot start until the Loop result has been reviewed and verified.
- Adds persistent `− / +` Team Grid zoom controls from 60% to 140%; the flow canvas scales as one unit while the selected-agent inspector remains readable.
- Widens the Team Grid sheet to a 1,060-point minimum and a 1,240-point preferred width, giving the agent flow substantially more room beside the inspector.
- Adds a direct `⌥⌘G` command for opening Team Grid from anywhere in the native app.
- Requires `What does this agent manage?` when adding either an independent team agent or a spawned subagent, saves the answer in that agent's own `memory_notes`, and exposes the durable ownership boundary in the Team Grid inspector.
- Gives JameClaw its own identity icon and every other team agent a stable distinct icon, using the known runtime icon where available and a deterministic icon for custom agents.

## Uncommitted 108D research-provider work

- Adds a dedicated Research Provider section beside the existing AI Provider settings in the native app.
- Supports Tavily, Brave Search, Perplexity, and keyless DuckDuckGo without replacing the primary chat model.
- Stores research API keys only through JameClaw's private security configuration and returns connection status without returning secrets.
- Makes the selected provider the exclusive active research backend, disables native-model search preference for that connection, and restarts the gateway after connect or disconnect.

## Uncommitted 108D AI-provider management work

- Keeps `Change primary provider` and `Add new AI provider` actions visible even when one or more AI providers are already configured.
- Lets the user select another configured provider and explicitly apply it as primary with a gateway restart.
- Reuses the full native provider catalogue to connect a completely new provider, save its credentials privately, set it as primary immediately, and restart the gateway.
- Replaces the invalid Nous `default` placeholder with Nous Portal's supported `anthropic/claude-sonnet-4.6` route while preserving compatibility for existing saved `nous/default` connections.
- Recognizes Nous low-credit responses as billing failures so the configured fallback provider can take over; without a working fallback, Chat now explains that the provider needs credits instead of showing a generic model/tool error.

## Uncommitted 108D single-app Dock identity

- Makes the top-level `JameClaw Desktop.app` the only regular macOS application and gives it bundle identifier `com.jameclaw.launcher` so the existing Dock shortcut continues to open the correct app.
- Removes the nested `Jame.app` and Settings application bundles; the launcher and gateway now run as internal helper executables without their own Dock identities.
- Uses `JameClaw Desktop` for the visible app and main-window title instead of presenting the old `Jame` name after launch.
- Migrates legacy native preferences from `com.jameclaw.home` into the unified top-level application.

Primary source files:

- `web/frontend/src/components/config/config-sections.tsx`
- `web/frontend/src/store/design.ts`
- `macos/JameClawHome/JameClawHome.swift`
- `pkg/extensions/catalog.go`
- `pkg/providers/openai_compat/provider.go`
- `pkg/providers/error_classifier.go`
- `pkg/agent/loop.go`
- `web/backend/api/team_operations.go`
- `web/backend/api/team_operations_test.go`
- `web/backend/api/voice_transcription.go`
- `web/backend/api/voice_transcription_test.go`
- `web/backend/api/research_providers.go`
- `web/backend/api/research_providers_test.go`

## Generated and runnable outputs

The feature has been compiled into the tracked frontend distributions and the runnable macOS bundle:

- `web/frontend/dist/`
- `web/backend/dist/`
- `web/build/jameclaw-launcher`
- `build/JameClaw Desktop.app/Contents/MacOS/Jame`
- `build/JameClaw Desktop.app/Contents/MacOS/jameclaw-launcher`
- `build/JameClaw Desktop.app/Contents/MacOS/jameclaw`

Because Vite uses content-hashed asset names, each rebuilt distribution records the previous assets as deleted and the replacement assets as new files. Review the source files above as the authoritative implementation.

## Validation recorded

- `npm run build` passed in `web/frontend`.
- `npx eslint src/store/design.ts src/components/config/config-sections.tsx` passed.
- `npm run build:backend` passed and regenerated the embedded Web Console distribution.
- `CGO_ENABLED=1 go build -tags stdjson -o build/jameclaw-launcher ./backend` passed in `web`; the linker emitted only its existing duplicate `-lobjc` warning.
- `./scripts/build-macos-app.sh jameclaw-darwin-arm64` rebuilt the runnable app bundle.
- `swiftc -parse-as-library -typecheck macos/JameClawHome/JameClawHome.swift -framework SwiftUI -framework AppKit -framework UserNotifications` passed.
- `codesign --verify --deep --strict --verbose=2 'build/JameClaw Desktop.app'` passed after ad-hoc signing the completed bundle.
- Bundle inspection found only the top-level `build/JameClaw Desktop.app` and no nested `.app` directories.
- The rebuilt app was restarted. Its authenticated local `/api/onboarding/status` response reported `"version":"108D"` and gateway PID `46426`.
- The JavaScript assets served by the running Desktop app contain `Full design theme`, `Jame Dark`, `sunset-studio`, and the persisted `design-density` setting.
- After terminating the stale native process and reopening the rebuilt unified app, macOS accessibility inspection confirmed `Full design theme` is visible under Settings → Design with the complete preset description and all individual controls below it.
- Cold-start and quit/reopen testing confirmed an on-screen standard `Jame` window at `900 × 650` through Core Graphics; the reopened process did not remain background-only.
- A measured Sessions → Chat regression test kept both pages at exactly `900 × 650`; Chat previously expanded the same window to `900 × 889`.
- Applying Nordic Frost through the actual `Full design theme` picker changed the stored design to Dark + Blue + Comfortable + Minimal while the window remained `900 × 650`.
- Visual checks confirmed the selected blue accent on Settings, Sessions, Memory, and Capabilities; the shared token implementation covers every remaining native navigation page.
- `swiftc` typechecking passed after the Team Grid model badges, active-task cards, and right-click delegation workflow were added.
- `go test ./backend/api -run 'TeamOperations|TeamTask' -count=1` passed for the real task creation, reassignment, and lifecycle API used by Team Grid.
- Native accessibility inspection of the rebuilt Team Grid confirmed the Hermes card and selected-agent inspector both display `Nous Research · tencent/hy3:free`, while Codex continues to display `Codex CLI · codex-cli` separately.
- Native visual and accessibility inspection confirmed the widened Team Grid, persistent 60% zoom state, separate Codex and Hermes identity icons, and the selected-agent `Manages` memory section.
- `go test ./backend/api -run 'TeamOperations|HandleCreateAgent|VoiceTranscription' -count=1` passed for Loop gating, create-agent memory persistence, and voice-upload validation.
- The final launcher binary contains `/api/voice/transcribe`; the final top-level native binary contains `What does this agent manage?`, `Stop and transcribe recording`, and `Voice transcription ready`.
- Native accessibility inspection confirmed the rebuilt idle control is exposed as `Send` with `Send message` help; the compiled binary also contains the `Jame is thinking` state used by the animated `...` control.
- Native accessibility inspection of the rebuilt Settings page confirmed the dedicated Research Provider section, provider selector, connection state, connect/update and disconnect actions, and explanatory separation from the primary AI provider. The current local configuration resolves DuckDuckGo as active.
- Native accessibility inspection confirmed `Change primary provider` and `Add new AI provider` remain visible with configured providers. Opening the new-provider flow exposed the provider catalogue, model selector, secure API-key field, and `Add and use as primary` action without opening the Web Console.
- A live Chat → Settings → Chat navigation check preserved the unsent draft in the root-owned chat runtime; the temporary verification draft was cleared afterward.
- A real Nous request reproduced the original 404 for model `default`; the rebuilt provider then routed the same request to `anthropic/claude-sonnet-4.6`, correctly identified the account's low-credit response, and completed through the configured Grok Build fallback with `Fallback: succeeded with grok-build/default after 2 attempts` and response `OK.`
- Targeted tests passed for the Nous catalogue preset, legacy-model normalization, Nous billing classification, fallback behavior, unavailable-model guidance, and insufficient-credit guidance.
- After rebuilding and reopening the signed bundle, macOS Computer Use listed exactly one visible JameClaw application: the top-level bundle `com.jameclaw.launcher`, displayed as `JameClaw Desktop`.
- Opening `JameClaw Desktop.app` again produced no second application entry or accessibility-window change, and the visible native process retained the same process identifier; the launcher remained live on port `18800` and the gateway on `18790` as internal helpers.
- `go test ./backend/api -run 'ResearchProvider|TeamOperations|HandleCreateAgent|VoiceTranscription' -count=1` passed, including secret persistence without API-key disclosure.
- `git diff --check` passed after the final Team Grid, agent-memory, icon, width, and voice-transcription changes.

## Before committing or pushing

Review the hashed distributions separately from the three source files, then run:

```bash
git status --short
git diff --check
git rev-list --left-right --count origin/main...HEAD
git log --branches --not --remotes --oneline
```

Do not describe this design-theme work as pushed or released until a new commit exists on the remote.

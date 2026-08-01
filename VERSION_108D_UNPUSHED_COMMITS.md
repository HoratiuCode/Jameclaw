# JameClaw 108D — Unpushed Git Work

This document records the local Git state and all work that has not been pushed from this checkout as of 2026-08-01.

## Version

- Working version: `108D`
- Branch: `main`
- Remote: `origin`
- Local HEAD: `6c950f04247d3913384eb902808766c17a9a8d76` (`update`)
- Remote `origin/main`: `6c950f04247d3913384eb902808766c17a9a8d76` (`update`)
- Remote references refreshed with `git fetch --all --prune` on 2026-08-01.

## Unpushed commits

There are currently **zero unpushed Git commits**. `main` is `0` commits ahead of and `0` commits behind `origin/main`. No commit reachable from a local branch is missing from the fetched remote-tracking branches.

The work described below is **uncommitted local work**, not a set of commits. It will become immutable Git history only after it is reviewed, committed, and pushed.

## Pending product work

### Native macOS application

- Expands the native workspace and navigation with a white, black, and orange design system, a Quick Actions palette, improved window behavior, saved theme and opacity preferences, and safer window restoration.
- Adds richer session browsing, search, archiving, renaming, resuming, and message previews.
- Adds native MCP connection setup, provider and fallback setup, task-folder organization, capabilities pages, self-improvement controls, and completion notifications.
- Adds document-access approval policy handling and associated macOS notification permission text.

Primary files:

- `macos/JameClawHome/JameClawHome.swift`
- `macos/JameClawHome/Info.plist`
- `scripts/build-macos-home-app.sh`
- `scripts/build-macos-app.sh`

### Agent runtime, memory, and self-improvement

- Improves conversational follow-up retrieval, memory recency and deduplication, task-plan feedback, turn handling, and coordinated agent execution.
- Adds persisted turn reflections and learning candidates, including approval-aware promotion into memory or reusable skills and protections for security-sensitive behavior.
- Extends tests for coordination, memory search, loop behavior, task feedback, and self-improvement.

Primary files:

- `pkg/agent/context.go`
- `pkg/agent/coordination_test.go`
- `pkg/agent/loop.go`
- `pkg/agent/loop_test.go`
- `pkg/agent/memory.go`
- `pkg/agent/memory_search_test.go`
- `pkg/agent/self_improvement.go` (untracked)
- `pkg/agent/self_improvement_test.go` (untracked)
- `pkg/agent/task_plan_feedback.go`
- `pkg/agent/task_plan_feedback_test.go`
- `pkg/agent/turn.go`

### Jame channel, configuration, gateway, and heartbeat

- Extends Jame channel events and protocol behavior used by the native and Web Console clients.
- Updates agent defaults and configuration behavior, gateway lifecycle handling, and the heartbeat/initiative workflow.
- Updates the workspace heartbeat templates and tests.

Primary files:

- `pkg/channels/interfaces.go`
- `pkg/channels/jame/jame.go`
- `pkg/channels/jame/jame_test.go`
- `pkg/channels/jame/protocol.go`
- `pkg/config/config.go`
- `pkg/config/config_test.go`
- `pkg/config/defaults.go`
- `pkg/gateway/gateway.go`
- `pkg/heartbeat/service.go`
- `pkg/heartbeat/service_test.go`
- `scripts/HEARTBEAT.md`
- `workspace/HEARTBEAT.md`

### Web backend and team operations

- Expands agent and session APIs for native session workflows, self-improvement data, activity summaries, and safer tool configuration.
- Adds persistent team goals and tasks with dependencies, file-scope conflict detection, review states, acceptance criteria, budgets, and verification evidence.
- Registers and tests the new backend routes and updates macOS desktop menu behavior.

Primary files:

- `web/backend/api/agents.go`
- `web/backend/api/agents_test.go`
- `web/backend/api/router.go`
- `web/backend/api/session.go`
- `web/backend/api/session_test.go`
- `web/backend/api/team_operations.go` (untracked)
- `web/backend/api/team_operations_test.go` (untracked)
- `web/backend/api/tools.go`
- `web/backend/api/tools_test.go`
- `web/backend/desktop_menu_darwin.m`
- `web/backend/main.go`

### Web Console and version 108D

- Extends configuration UI/model fields and English labels for the pending runtime and design settings.
- Displays `Version 108D` in the Web Console sidebar footer and uses `108D` as the backend default version when linker flags do not override it.
- Adds the local pnpm workspace build allowlist.

Primary files:

- `pkg/config/version.go`
- `web/frontend/src/components/app-sidebar.tsx`
- `web/frontend/src/components/config/config-page.tsx`
- `web/frontend/src/components/config/config-sections.tsx`
- `web/frontend/src/components/config/form-model.ts`
- `web/frontend/src/i18n/locales/en.json`
- `web/frontend/pnpm-workspace.yaml` (untracked)

### Human-readable update history

- `JAMECLAW_UPDATE_HISTORY.md` has pending entries for native MCP setup, memory retrieval, team-agent templates, the team activity map, Quick Actions, per-conversation provider controls, and chat fast paths.

## Generated, dependency, and runtime files

The checkout also contains pending non-source changes. After the `108D` source update and Web Console rebuild, `git status --porcelain=v1 -uall` reported 323 changed or untracked paths:

| Category | Paths | Notes |
| --- | ---: | --- |
| Frontend dependency tree | 144 | Installed `web/frontend/node_modules/` files and links; review separately from product source. |
| Backend distribution | 111 | Rebuilt or replaced `web/backend/dist/` assets. |
| Build outputs | 4 | macOS/Go binaries under `build/` and `web/build/`. |
| Onboarding runtime scaffold | 12 | Heartbeat, cron, state, and session files under `cmd/jameclaw/internal/onboard/workspace/`; these include runtime data and should not be staged blindly. |
| Source, documentation, metadata, and other files | 52 | The tracked and untracked implementation files described above, this version document, and `.DS_Store` metadata. |

The full tracked diff after the rebuild was 249 files with 10,838 insertions and 2,291 deletions. Excluding dependencies, generated distributions, build outputs, runtime scaffold data, and `.DS_Store`, the pending tracked source/documentation diff was 41 files with 6,768 insertions and 522 deletions. The new version document itself is untracked and is therefore not included in those tracked-diff totals.

## Verification commands

Use these commands before creating or pushing the eventual commit:

```bash
git fetch --all --prune
git rev-list --left-right --count origin/main...HEAD
git log --branches --not --remotes --oneline
git status --short
git diff --check
```

Expected unpushed-commit result at the time this document was created:

```text
0  0
```

Do not describe the current local work as pushed or released until a new commit exists on the remote.

## Validation recorded for 108D

- `npm run build:backend` passed and regenerated the embedded Web Console distribution.
- `CGO_ENABLED=1 go build -tags stdjson -o build/jameclaw-launcher ./backend` passed; the macOS linker emitted only its existing duplicate `-lobjc` warning.
- `./scripts/build-macos-app.sh jameclaw-darwin-arm64` rebuilt and ad-hoc signed the runnable `build/JameClaw Desktop.app` bundle.
- `codesign --verify --deep --strict --verbose=2 'build/JameClaw Desktop.app'` passed.
- `npx eslint src/components/app-sidebar.tsx` passed.
- `go test ./pkg/config` passed.
- `go test ./web/backend/api` passed outside the restricted sandbox because its `httptest` cases bind local loopback ports.
- After restarting the rebuilt app, the authenticated local `/api/onboarding/status` response reported `"version": "108D"`.

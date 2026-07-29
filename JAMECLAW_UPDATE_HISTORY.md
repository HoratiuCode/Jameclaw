# JameClaw Update History

This is the human-readable update ledger for this fork of JameClaw. It records what changed in the code that was committed and pushed to the `origin/main` GitHub branch.

**Tracking state**

- Repository: [`HoratiuCode/Jameclaw`](https://github.com/HoratiuCode/Jameclaw)
- Branch: `main`
- Latest recorded revision: [`69a4d6b0`](https://github.com/HoratiuCode/Jameclaw/commit/69a4d6b030ad5c2ed6132abe3846769ad6353bf5)
- Latest recorded date: 2026-07-28
- Git tags: none at the time this file was created
- History inspected: 163 commits, from 2026-03-25 through 2026-07-28

> Version names in this file use a date plus the short commit SHA because the repository does not yet publish Git tags or semantic release numbers. A version becomes immutable once it is committed.

## Current version — 2026-07-28 (`69a4d6b0`)

**Desktop teams, artifacts, memory, and conversation management.** This is the latest GitHub-pushed state inspected for this document. The commit changes 178 files (1,483 additions and 235 deletions); many of those are regenerated frontend/backend distribution assets and rebuilt macOS binaries. The source-level product changes are:

- **Artifact workspace support** — adds backend artifact APIs (`web/backend/api/artifacts.go`) plus tests, a frontend artifact API and route, an Artifacts sidebar entry, and an artifact page. Users can browse saved website artifacts, inspect/edit project files, save changes, and run an `index.html` artifact inside the app.
- **Native macOS artifact workspace** — expands `macos/JameClawHome/JameClawHome.swift` with an artifact project browser, source editor, save flow, and local WebKit preview. A bundled `creation-of-adam.jpg` is added to the desktop app resources.
- **Long-term memory and identity editor** — adds a native Memory section that reads and writes the main agent memory, user profile, people/relationships, persona, tone, discussion mode, configured memory notes, and status style.
- **Conversation controls** — adds fixed/pinned chats, a pin/unpin endpoint flow, session API path encoding for IDs that contain reserved characters, and dedicated Fixed Chats and Sessions navigation.
- **Chat and session refinements** — updates the web chat composer/page and dashboard sessions list to surface the new workspace and conversation behavior.
- **Provider resilience** — lets the macOS quick settings screen select a primary provider and an optional fallback provider, calling the model failover API when configured.
- **Agent context** — updates `pkg/agent/context.go` for the current agent behavior.
- **Build outputs** — rebuilds the macOS launcher/app and frontend/backend bundles so distributed apps include the source changes.

### Files that define the current change

| Area | Main files modified or added |
| --- | --- |
| Artifact API | `web/backend/api/artifacts.go`, `web/backend/api/artifacts_test.go`, `web/backend/api/router.go`, `web/backend/api/session.go` |
| Web Console | `web/frontend/src/api/artifacts.ts`, `web/frontend/src/api/sessions.ts`, `web/frontend/src/components/artifacts/artifacts-page.tsx`, `web/frontend/src/components/chat/chat-composer.tsx`, `web/frontend/src/components/chat/chat-page.tsx`, `web/frontend/src/components/dashboard/sessions-page.tsx`, `web/frontend/src/components/app-sidebar.tsx`, `web/frontend/src/routes/artifacts.tsx` |
| macOS desktop | `macos/JameClawHome/JameClawHome.swift`, `macos/JameClawHome/creation-of-adam.jpg`, `scripts/build-macos-home-app.sh` |
| Agent runtime | `pkg/agent/context.go` |
| Generated packages | `web/frontend/dist/`, `web/backend/dist/`, `build/JameClaw Desktop.app/`, `web/build/` |

## Historical versions and milestones

### 2026-07-25 — Desktop workflow expansion

- `a080998f` — adds agent orchestration, teams, and subagents.
- `809626db` — improves desktop navigation.
- `cb5f7fe1` — adds desktop automation support.

### 2026-07-13 to 2026-07-22 — Automation, memory, and desktop iteration

- `c65c0121`, `9f5e858e`, `ddea902f` — desktop updates and application launch work.
- `8a5cc1af`, `503879ab`, `0c1be212`, `9310dbf3` — Chrome/MCP extension design, visual polish, reasoning, and Web Console effects.
- `fd661cff` — expands sending capability.
- `7bcc5a82`, `3930e456`, `7e5d0dc4` — independence and scheduled/managed message automation.
- `42b5788b`, `67d690d0`, `619201ad`, `6509b631`, `0bf4a3cb`, `4f98acec`, `c7eaf9dc` — skills, cron output checks, Web Console memory, product guide, Memory v2, and Aspirine/memory mode.
- `9e529f0f`, `32cc1cf6`, `14466fc6`, `5552af59`, `2296dba2`, `9ddeeca9` — logs/CLI, dashboard, blueprint animation, AI automation, JameClaw screen, and CLI updates.

### 2026-07-08 to 2026-07-12 — Web Console and extension foundation

- `243ca36e`, `f6b1ceac`, `e5938a6d`, `bda2e90c` — voice repair, Instagram photo update, icons, and new automation.
- `9deabc0c`, `c377d835`, `eec168b0`, `cbe2b033`, `5e8fd6bb` — launcher browser, agent browser, opening appearance, chat mentions, and macOS improvements.
- `ca3205a2`, `6e15a3cf`, `4b4c7820ed` — Chrome extension work and updates.
- `cd4db386`, `bb93c0e9`, `317cfcbe` — agent/settings, models, and settings UI refinements.
- `b11261a5`, `121de420`, `28c80ec2`, `3fe33ed3`, `b8feaa90`, `ee7eff94` — naming/system updates, Web Console error resolution, channel structure, PDF capability, and screen capability.
- `443693cf`, `519bfca6`, `b98ff247`, `0db62a45`, `d0a46eba`, `3f1865c2` — voice/search, Web Console design, Telegram fixes, chat, WebUI fixes, and dashboard/routes/memory/gateway/skills.

### 2026-07-02 to 2026-07-07 — Launch, onboarding, and provider workflows

- `27032b74`, `68500513`, `41035dd6`, `e243d542`, `46fbf198` — early version marker, expanded capability, Web Console chat fix, launcher, and icon appearance.
- `e38d0434`, `15428050`, `5790bde6`, `b52a82d6` — Codex CLI work, models, spawned agents, and terminal chat from the navbar.
- `6407ad0e`, `16dfd38d`, `f7afc15e` — phone experience and onboarding.
- `bc1d2ecd`, `95770e7c`, `996e67e0` — system/release updates plus reasoning events and automatic verification hooks.

### 2026-04-08 to 2026-06-24 — Platform capability growth

- `8fe65f89`, `56a7bdee`, `0ba12b1a` — agent guide and versioned updates.
- `ac809107`, `5696f2d3`, `f56bacc5`, `d01f6aea`, `aeda196b`, `553fc263` — Web Console, message error handling, provider improvements, personalization, and broader error handling.
- `392501f5`, `0c083db4`, `30893a29`, `eb1a1ce4`, `6dc1ece2` — README plan, webhook v2/v2.1, dock, and agent styling.
- `e33ee70e` through `03d2f6ed` — README, skills, Chrome theme, loading/error work, about pages, UI structure, memory, and Google-related integration.
- `c88623a3` through `6f01b808` — Chrome extension UI/logic, side panel, Manifest V3, extension recovery, learn page, media, dark mode, and menus.
- `f20ae65d` through `1afbea8b` — version markers, OpenRouter model/provider integration, launcher rebuild, and error cleanup.

### 2026-03-25 to 2026-04-03 — Initial product structure

- `5d598065` — initial commit.
- `392bf83e` through `77868835` — Web Console, launcher/TUI, onboarding/uninstall, README, UI assets, links, prompts, and project cleanup.
- `7e070ab3`, `7c44943b`, `99aaee5a`, `cf6cb74a`, `67522153`, `d20c0b45` — tags/cards, security, skills, tool documentation, and webhook work.

## Complete committed-update ledger

The source of truth remains Git history. Use this command to list every committed update, newest first, with its date and stable version identifier:

```bash
git log origin/main --date=short --pretty=format:'%h | %ad | %s'
```

Use this command to see the exact files and line changes for one listed version:

```bash
git show --stat --find-renames <commit-sha>
git show <commit-sha> -- <path>
```

## Keeping this file current after every GitHub push

When a new change is ready to commit, add a new section above **Current version** before pushing. Include:

1. Date, short SHA, and a clear user-facing title.
2. What changed, grouped by feature rather than by generated bundle file.
3. The key source files changed, plus tests added or updated.
4. Any migration, compatibility, security, or configuration impact.
5. Whether frontend/backend/macOS build outputs were regenerated.

After the push, verify the record matches GitHub:

```bash
git fetch origin
git log --oneline HEAD..origin/main
git diff --stat HEAD~1..HEAD
```

If Git tags are introduced later, replace the date/SHA heading with the tag (for example, `v0.8.0`) and retain the SHA below it. Do not change historical entries merely to rename versions; add a clarification note instead.

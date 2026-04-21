---
name: session-logs
description: Search and analyze JameClaw session logs and legacy OpenClaw history using jq and rg.
metadata: {"openclaw":{"emoji":"📜","requires":{"bins":["jq","rg"]},"install":[{"id":"brew-jq","kind":"brew","formula":"jq","bins":["jq"],"label":"Install jq (brew)"},{"id":"brew-rg","kind":"brew","formula":"ripgrep","bins":["rg"],"label":"Install ripgrep (brew)"}]}}
---

# session-logs

Use this skill when you need older chat history, session archaeology, or raw
log inspection.

## Where logs live

Primary JameClaw sessions:

- `$JAMECLAW_HOME/workspace/sessions/` (default: `~/.jameclaw/workspace/sessions/`)

Files:

- `*.jsonl` - current append-only message stream
- `*.meta.json` - summary, count, skip offset, timestamps
- `*.json` - legacy session snapshots if older storage is still present

Legacy OpenClaw history, if it has not been migrated yet:

- `~/.openclaw/agents/<agentId>/sessions/`

Notes:

- `jameclaw migrate --from openclaw` migrates config and workspace files.
- It does not automatically rewrite OpenClaw session transcripts into the
  JameClaw JSONL store.
- If the user wants old OpenClaw conversations searchable in JameClaw, inspect
  them in place or copy/convert them into the JameClaw session directory.

## Quick commands

List JameClaw session files:

```bash
SESSION_DIR="${JAMECLAW_HOME:-$HOME/.jameclaw}/workspace/sessions"
for f in "$SESSION_DIR"/*.jsonl; do
  [ -e "$f" ] || continue
  printf "%s %s\n" "$(basename "$f")" "$(wc -l < "$f")"
done | sort -r
```

Show metadata for the newest session:

```bash
SESSION_DIR="${JAMECLAW_HOME:-$HOME/.jameclaw}/workspace/sessions"
latest="$(ls -1t "$SESSION_DIR"/*.jsonl 2>/dev/null | head -1)"
jq '.' "${latest%.jsonl}.meta.json"
```

Extract user messages from a JameClaw JSONL session:

```bash
jq -r 'select(.role == "user") | .content' <session>.jsonl
```

Search assistant messages:

```bash
jq -r 'select(.role == "assistant") | .content' <session>.jsonl | rg -i "keyword"
```

List tool calls:

```bash
jq -r '.tool_calls[]?.function?.name' <session>.jsonl | sort | uniq -c
```

## OpenClaw legacy format

If you are reading an un-migrated OpenClaw transcript directly, its JSONL
wrapper uses the older nested shape:

```bash
jq -r 'select(.message.role == "user") | .message.content[]? | select(.type == "text") | .text' <session>.jsonl
```

For a one-time migration into JameClaw, run the built-in migration first:

```bash
jameclaw migrate --from openclaw --force
```

Then inspect the JameClaw workspace session directory again.


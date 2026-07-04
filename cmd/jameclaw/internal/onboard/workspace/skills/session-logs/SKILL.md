---
name: session-logs
description: Search and analyze JameClaw session logs using jq and rg.
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

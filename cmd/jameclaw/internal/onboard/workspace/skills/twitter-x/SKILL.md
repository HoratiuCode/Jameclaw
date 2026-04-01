---
name: twitter-x
description: A CLI tool for making authenticated requests to the X (Twitter) API. Use this skill when you need to post tweets, reply, quote, search, read posts, manage followers, send DMs, upload media, or interact with any X API v2 endpoint.
metadata:
  {
    "openclaw":
      {
        "emoji": "🐦",
        "requires": { "bins": ["xurl"] },
        "install":
          [
            {
              "id": "brew",
              "kind": "brew",
              "formula": "xdevplatform/tap/xurl",
              "bins": ["xurl"],
              "label": "Install xurl (brew)",
            },
            {
              "id": "npm",
              "kind": "npm",
              "package": "@xdevplatform/xurl",
              "bins": ["xurl"],
              "label": "Install xurl (npm)",
            },
          ],
      },
  }
---

# twitter-x

`xurl` is the CLI used for X and Twitter API work. It supports shortcut commands and raw curl-style access to v2 endpoints. Commands return JSON to stdout.

Prerequisites

- This skill requires the `xurl` CLI utility.
- Before using any command you must be authenticated. Run `xurl auth status` to check.

Secret safety

- Never read or print `~/.xurl` in model context.
- Never ask the user to paste credentials or tokens into chat.
- The user must complete credential setup manually outside the agent session.
- Never use `--verbose` or `-v` in agent sessions.

Quick reference

- Post: `xurl post "Hello world!"`
- Reply: `xurl reply POST_ID "Nice post!"`
- Quote: `xurl quote POST_ID "My take"`
- Read a post: `xurl read POST_ID`
- Search posts: `xurl search "QUERY" -n 10`
- Who am I: `xurl whoami`
- Look up a user: `xurl user @handle`
- Home timeline: `xurl timeline -n 20`
- Mentions: `xurl mentions -n 10`
- Follow: `xurl follow @handle`
- Send DM: `xurl dm @handle "message"`
- Upload media: `xurl media upload path/to/file.mp4`

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

Installation

- Homebrew: `brew install --cask xdevplatform/tap/xurl`
- npm: `npm install -g @xdevplatform/xurl`
- Go: `go install github.com/xdevplatform/xurl@latest`

Prerequisites

- This skill requires the `xurl` CLI utility: <https://github.com/xdevplatform/xurl>.
- Before using any command you must be authenticated. Run `xurl auth status` to check.

Secret safety

- Never read, print, parse, summarize, upload, or send `~/.xurl` to the model context.
- Never ask the user to paste credentials or tokens into chat.
- The user must fill `~/.xurl` with required secrets manually on their own machine.
- Do not recommend or execute auth commands with inline secrets in agent sessions.
- Never use `--verbose` or `-v` in agent sessions.

Authentication

- App credential registration must be done manually by the user outside the agent session.
- After credentials are registered, authenticate with `xurl auth oauth2`.
- To verify whether credentials are already available, run `xurl auth status`.

Quick reference

- Post: `xurl post "Hello world!"`
- Reply: `xurl reply POST_ID "Nice post!"`
- Quote: `xurl quote POST_ID "My take"`
- Delete a post: `xurl delete POST_ID`
- Read a post: `xurl read POST_ID`
- Search posts: `xurl search "QUERY" -n 10`
- Who am I: `xurl whoami`
- Look up a user: `xurl user @handle`
- Home timeline: `xurl timeline -n 20`
- Mentions: `xurl mentions -n 10`
- Like: `xurl like POST_ID`
- Repost: `xurl repost POST_ID`
- Bookmark: `xurl bookmark POST_ID`
- Follow: `xurl follow @handle`
- Send DM: `xurl dm @handle "message"`
- Upload media: `xurl media upload path/to/file.mp4`

Examples

```bash
xurl post "Hello world!"
xurl media upload photo.jpg
xurl post "Check this out" --media-id MEDIA_ID
xurl reply 1234567890 "Great point!"
xurl reply https://x.com/user/status/1234567890 "Agreed!"
xurl quote 1234567890 "Adding my thoughts"
xurl read 1234567890
xurl search "golang"
xurl whoami
xurl user @XDevelopers
xurl timeline -n 25
xurl mentions
```

Notes

- Anywhere `POST_ID` appears, you can also paste a full post URL like `https://x.com/user/status/1234567890`.
- Leading `@` is optional in usernames.
- Prefer the shortcut commands when possible; use raw endpoint calls only when needed.

# JameClaw Chrome Extension

This extension is a thin shell around the local JameClaw web launcher.

It does two things:

- opens a compact JameClaw page from `http://localhost:18800/extension`
- captures the active tab's title, URL, selection, and a short text excerpt, then forwards that context into the compact assistant

## Load it in Chrome

1. Start the launcher and open JameClaw once in Chrome at `http://localhost:18800`
2. Open `chrome://extensions`
3. Enable **Developer mode**
4. Click **Load unpacked**
5. Select this folder: `web/chrome-extension`

## Current behavior

- `Read This`: summarize the current page with actions and notable points
- `Explain Selection`: explain the selected text in context
- `Save To Calendar`: asks JameClaw to extract event details and use tools if available
- `Next Steps`: turns the page into a short action plan

## Current limitation

The embedded page uses the same local session protection as the normal launcher. If the popup shows an auth message, open `http://localhost:18800` in Chrome first so the launcher session cookie is created.

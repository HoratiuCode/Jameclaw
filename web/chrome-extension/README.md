# JameClaw Chrome Extension

This extension is a native popup chat for the local JameClaw launcher.

It does two things:

- opens a plain chat popup inside Chrome
- captures the active tab's title, URL, selection, and a short text excerpt, then attaches that context to each message

## Load it in Chrome

1. Start the launcher and open JameClaw once in Chrome at `http://localhost:18800`
2. Open `chrome://extensions`
3. Enable **Developer mode**
4. Click **Load unpacked**
5. Select this folder: `web/chrome-extension`

## Current behavior

- the popup only shows chat
- the current page context is attached automatically in the background
- it talks to JameClaw through a local extension bootstrap endpoint and websocket proxy on `localhost:18800`

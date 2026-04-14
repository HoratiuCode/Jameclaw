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
- selected text on the page is remembered and reused when you open the extension
- the header `Pick` action refreshes the current page context and selected text
- the header `Dock` action opens a persistent JameClaw chat tab so it stays visible while you work on the website
- it talks to JameClaw through a local extension bootstrap endpoint and websocket proxy on `localhost:18800`

## Notes

- if you want JameClaw to focus on one part of a website, select that text before opening the extension
- if `Dock` is used, the extension opens a dedicated chat tab instead of a Chrome side panel

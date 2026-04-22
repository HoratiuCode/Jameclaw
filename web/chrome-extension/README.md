# JameClaw Chrome Extension

This extension is a native popup chat for the local JameClaw launcher, with an optional floating dock.

It does two things:

- opens a plain chat popup inside Chrome
- captures the active tab's title, URL, selection, and a short text excerpt, then attaches that context to each message

## Load it in Chrome

Recommended for non-technical users:

1. Open `chrome://extensions`
2. Enable **Developer mode**
3. Click **Load unpacked**
4. Select the top-level folder named `Chrome-Extension-Upload`

Developer/source folder:

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
- the header `Dock` action opens a floating JameClaw panel in the corner of the page so it stays visible while you work
- when docked, the panel is restored on the next page you open in the same tab
- it talks to JameClaw through a local extension bootstrap endpoint and websocket proxy on `localhost:18800`

## Notes

- if you want JameClaw to focus on one part of a website, select that text before opening the extension
- if `Dock` is used, the extension opens a floating corner panel instead of a separate window

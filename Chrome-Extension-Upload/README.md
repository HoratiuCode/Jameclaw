# JameClaw Chrome Extension

This extension is a native popup chat for the local JameClaw launcher, with an optional floating dock.

It does two things:

- opens a plain chat popup inside Chrome
- captures the active tab's title, URL, description, headings, selected text, and readable main content, then attaches that context to each message

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

- the popup shows chat plus a compact page-context card, so you can see what Jame will receive
- the current page context includes the main article or page region, page summary, and heading outline instead of an unstructured body dump
- selected text on the page is remembered and reused when you open the extension
- the header `Pop out` action opens the chat in a separate JameClaw window
- it talks to JameClaw through a local extension bootstrap endpoint and websocket proxy on `localhost:18800`
- when an extension-enabled Chrome tab is open, the agent can inspect the page and use structured browser actions: navigate, click CSS selectors, type into form controls, scroll, go back, and reload

## Notes

- if you want JameClaw to focus on one part of a website, select that text before opening the extension
- keep the target Chrome tab open and visible while the agent is controlling it; the extension never exposes arbitrary JavaScript execution
- browser inspection returns stable selectors, labels, and control details so the agent can understand and operate pages more reliably

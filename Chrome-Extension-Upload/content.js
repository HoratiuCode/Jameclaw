const PAGE_TEXT_LIMIT = 5000
const SELECTION_LIMIT = 3000
const DOCK_STORAGE_KEY = "jameclaw-extension-dock-enabled"
const DOCK_ROOT_ID = "jameclaw-dock-root"
const DOCK_IFRAME_ID = "jameclaw-dock-iframe"
const DOCK_CLOSE_ID = "jameclaw-dock-close"
const DOCK_URL = chrome.runtime.getURL("sidepanel.html?mode=dock")
const DOCK_WIDTH = 420
const DOCK_HEIGHT = 620
let lastSelectionText = ""
let browserClientID = ""

function getBrowserClientID() {
  return new Promise((resolve) => {
    chrome.runtime.sendMessage({ type: "jameclaw-extension-browser-client" }, (response) => {
      resolve(response?.ok ? response.clientId : "")
    })
  })
}

async function sendBrowserResult(id, result) {
  await fetch("http://127.0.0.1:18800/api/extension/browser/result", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ id, ...result }) })
}

function pageMap() {
  const controls = Array.from(document.querySelectorAll("a, button, input, textarea, select")).slice(0, 80).map((element) => ({ tag: element.tagName.toLowerCase(), text: trimText((element.innerText || element.value || element.getAttribute("aria-label") || "").trim(), 160), id: element.id || "", name: element.getAttribute("name") || "", href: element instanceof HTMLAnchorElement ? element.href : "" }))
  return JSON.stringify({ title: document.title, url: location.href, text: getPageText(), controls })
}

async function runBrowserCommand(command) {
  try {
    const args = command.args || {}
    switch (command.action) {
      case "inspect": return { content: pageMap() }
      case "navigate": if (!/^https?:/i.test(args.url || "")) throw new Error("Only http and https URLs are allowed."); location.assign(args.url); return { content: `Navigating to ${args.url}` }
      case "click": { const element = document.querySelector(args.selector); if (!element) throw new Error(`No element matches ${args.selector}`); element.click(); return { content: `Clicked ${args.selector}` } }
      case "type": { const element = document.querySelector(args.selector); if (!(element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement || element instanceof HTMLSelectElement)) throw new Error(`Selector is not a form field: ${args.selector}`); element.focus(); element.value = String(args.text || ""); element.dispatchEvent(new Event("input", { bubbles: true })); element.dispatchEvent(new Event("change", { bubbles: true })); return { content: `Typed text into ${args.selector}` } }
      case "scroll": window.scrollBy(Number(args.x) || 0, Number(args.y) || 600); return { content: "Scrolled page" }
      case "go_back": history.back(); return { content: "Navigating back" }
      case "reload": location.reload(); return { content: "Reloading page" }
      default: throw new Error("Unsupported browser action")
    }
  } catch (error) { return { content: "", error: error instanceof Error ? error.message : String(error) } }
}

async function pollBrowserCommands() {
  if (!browserClientID || document.visibilityState !== "visible") return
  try { const response = await fetch("http://127.0.0.1:18800/api/extension/browser/next", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ client_id: browserClientID }) }); if (response.status === 204 || !response.ok) return; const command = await response.json(); if (command?.id) await sendBrowserResult(command.id, await runBrowserCommand(command)) } catch { }
}

function getSelectionText() {
  const selection = window.getSelection()
  return selection ? selection.toString().trim() : ""
}

function trimText(text, limit) {
  if (text.length <= limit) {
    return text
  }

  return `${text.slice(0, limit)}...`
}

function rememberSelection() {
  const selection = trimText(getSelectionText(), SELECTION_LIMIT)
  if (selection) {
    lastSelectionText = selection
  }
}

function getPageText() {
  const root = document.body
  if (!root) {
    return ""
  }

  const text = root.innerText.replace(/\s+/g, " ").trim()
  if (text.length <= PAGE_TEXT_LIMIT) {
    return text
  }

  return `${text.slice(0, PAGE_TEXT_LIMIT)}...`
}

function getDockRoot() {
  return document.getElementById(DOCK_ROOT_ID)
}

function ensureDockPanel() {
  if (getDockRoot()) {
    return
  }

  const root = document.createElement("div")
  root.id = DOCK_ROOT_ID
  root.style.cssText = [
    "all: initial",
    "position: fixed",
    "right: 16px",
    "bottom: 16px",
    `width: ${DOCK_WIDTH}px`,
    `height: ${DOCK_HEIGHT}px`,
    "z-index: 2147483647",
    "border-radius: 18px",
    "overflow: hidden",
    "box-shadow: 0 24px 80px rgba(0, 0, 0, 0.38)",
    "background: rgba(15, 15, 15, 0.98)",
    "pointer-events: auto",
    "contain: layout paint size style",
  ].join(";")

  const frame = document.createElement("iframe")
  frame.id = DOCK_IFRAME_ID
  frame.src = DOCK_URL
  frame.title = "JameClaw Dock"
  frame.allow = "clipboard-read; clipboard-write"
  frame.style.cssText = [
    "display: block",
    "width: 100%",
    "height: 100%",
    "border: 0",
    "background: transparent",
  ].join(";")

  const close = document.createElement("button")
  close.id = DOCK_CLOSE_ID
  close.type = "button"
  close.setAttribute("aria-label", "Close JameClaw Dock")
  close.textContent = "x"
  close.style.cssText = [
    "all: initial",
    "position: absolute",
    "right: 8px",
    "top: 8px",
    "width: 28px",
    "height: 28px",
    "z-index: 2147483647",
    "display: grid",
    "place-items: center",
    "border-radius: 999px",
    "background: rgba(15, 15, 15, 0.72)",
    "border: 1px solid rgba(255, 255, 255, 0.18)",
    "box-shadow: 0 8px 24px rgba(0, 0, 0, 0.28)",
    "color: #fff",
    "font: 20px/1 Arial, sans-serif",
    "cursor: pointer",
    "pointer-events: auto",
  ].join(";")
  close.addEventListener("click", () => {
    if (chrome.storage?.local) {
      chrome.storage.local.set({ [DOCK_STORAGE_KEY]: false })
    }
    removeDockPanel()
  })

  root.appendChild(frame)
  root.appendChild(close)

  const parent = document.body || document.documentElement
  parent.appendChild(root)
}

function removeDockPanel() {
  const root = getDockRoot()
  if (root) {
    root.remove()
  }
}

function setDockEnabled(enabled) {
  if (enabled && chrome.storage?.local) {
    chrome.storage.local.set({ [DOCK_STORAGE_KEY]: false })
  }
  removeDockPanel()
}

function syncDockFromStorage() {
  removeDockPanel()
  if (!chrome.storage?.local) {
    return
  }

  chrome.storage.local.set({ [DOCK_STORAGE_KEY]: false })
}

document.addEventListener("mouseup", rememberSelection, true)
document.addEventListener("keyup", rememberSelection, true)
document.addEventListener("selectionchange", () => {
  const selection = trimText(getSelectionText(), SELECTION_LIMIT)
  if (selection) {
    lastSelectionText = selection
  }
})

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type === "jameclaw-extension-dock-state") {
    setDockEnabled(Boolean(message.enabled))
    sendResponse({ ok: true })
    return false
  }

  if (message?.type !== "jameclaw-extension-get-context") {
    return false
  }

  const liveSelection = trimText(getSelectionText(), SELECTION_LIMIT)
  const selection = liveSelection || lastSelectionText

  sendResponse({
    title: document.title || "",
    url: window.location.href || "",
    selection,
    pageText: getPageText(),
  })

  return false
})

syncDockFromStorage()
getBrowserClientID().then((id) => { browserClientID = id; void pollBrowserCommands(); setInterval(() => void pollBrowserCommands(), 1000) })

const PAGE_TEXT_LIMIT = 5000
const SELECTION_LIMIT = 3000
const DOCK_STORAGE_KEY = "jameclaw-extension-dock-enabled"
const DOCK_ROOT_ID = "jameclaw-dock-root"
const DOCK_IFRAME_ID = "jameclaw-dock-iframe"
const DOCK_URL = chrome.runtime.getURL("sidepanel.html?mode=dock")
const DOCK_WIDTH = 420
const DOCK_HEIGHT = 620
let lastSelectionText = ""

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

  root.appendChild(frame)

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
  if (enabled) {
    ensureDockPanel()
  } else {
    removeDockPanel()
  }
}

function syncDockFromStorage() {
  if (!chrome.storage?.local) {
    return
  }

  chrome.storage.local.get([DOCK_STORAGE_KEY], (result) => {
    if (chrome.runtime.lastError) {
      return
    }

    setDockEnabled(Boolean(result?.[DOCK_STORAGE_KEY]))
  })
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

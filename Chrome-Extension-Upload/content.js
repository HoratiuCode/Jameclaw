const PAGE_TEXT_LIMIT = 9000
const PAGE_OUTLINE_LIMIT = 24
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
  await fetch("http://127.0.0.1:18800/api/extension/browser/result", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id, ...result }),
  })
}

function pageMap() {
  const controls = Array.from(document.querySelectorAll("a, button, input, textarea, select"))
    .slice(0, 80)
    .map((element) => ({
      tag: element.tagName.toLowerCase(),
      text: trimText((element.innerText || element.value || element.getAttribute("aria-label") || "").trim(), 160),
      id: element.id || "",
      name: element.getAttribute("name") || "",
      href: element instanceof HTMLAnchorElement ? element.href : "",
      selector: elementSelector(element),
      role: element.getAttribute("role") || "",
      type: element.getAttribute("type") || "",
      disabled: Boolean(element.disabled),
    }))
  return JSON.stringify({ title: document.title, url: location.href, description: pageDescription(), outline: pageOutline(), text: getPageText(), controls })
}

function elementSelector(element) {
  if (element.id) return `#${CSS.escape(element.id)}`
  const testID = element.getAttribute("data-testid")
  if (testID) return `[data-testid="${CSS.escape(testID)}"]`
  const name = element.getAttribute("name")
  if (name) return `${element.tagName.toLowerCase()}[name="${CSS.escape(name)}"]`
  const label = element.getAttribute("aria-label")
  if (label) return `${element.tagName.toLowerCase()}[aria-label="${CSS.escape(label)}"]`
  return element.tagName.toLowerCase()
}

async function runBrowserCommand(command) {
  try {
    const args = command.args || {}
    switch (command.action) {
      case "inspect": return { content: pageMap() }
      case "navigate":
        if (!/^https?:/i.test(args.url || "")) throw new Error("Only http and https URLs are allowed.")
        location.assign(args.url); return { content: `Navigating to ${args.url}` }
      case "click": {
        const element = document.querySelector(args.selector)
        if (!element) throw new Error(`No element matches ${args.selector}`)
        const actionLabel = [element.innerText, element.getAttribute("aria-label"), element.getAttribute("title"), element.getAttribute("data-testid")].filter(Boolean).join(" ")
        if (/\b(send|post|tweet|reply)\b/i.test(actionLabel) && !window.confirm(`JameClaw is ready to ${actionLabel.trim() || "send this message"}. Review the recipient and text, then choose OK to continue.`)) {
          return { content: "Send cancelled by user" }
        }
        element.click(); return { content: `Clicked ${args.selector}` }
      }
      case "type": {
        const element = document.querySelector(args.selector)
        if (!(element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement || element instanceof HTMLSelectElement || element instanceof HTMLElement && element.isContentEditable)) throw new Error(`Selector is not a text field or message composer: ${args.selector}`)
        const text = String(args.text || "")
        element.focus()
        if (element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement || element instanceof HTMLSelectElement) {
          element.value = text
          element.dispatchEvent(new Event("change", { bubbles: true }))
        } else {
          const selection = window.getSelection()
          const range = document.createRange()
          range.selectNodeContents(element)
          range.collapse(true)
          selection?.removeAllRanges()
          selection?.addRange(range)
          document.execCommand("insertText", false, text)
        }
        element.dispatchEvent(new InputEvent("input", { bubbles: true, inputType: "insertText", data: text }))
        return { content: `Pasted text into ${args.selector}` }
      }
      case "scroll": window.scrollBy(Number(args.x) || 0, Number(args.y) || 600); return { content: "Scrolled page" }
      case "go_back": history.back(); return { content: "Navigating back" }
      case "reload": location.reload(); return { content: "Reloading page" }
      default: throw new Error("Unsupported browser action")
    }
  } catch (error) { return { content: "", error: error instanceof Error ? error.message : String(error) } }
}

async function pollBrowserCommands() {
  if (!browserClientID || document.visibilityState !== "visible") return
  try {
    const response = await fetch("http://127.0.0.1:18800/api/extension/browser/next", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ client_id: browserClientID }) })
    if (response.status === 204 || !response.ok) return
    const command = await response.json()
    if (command?.id) await sendBrowserResult(command.id, await runBrowserCommand(command))
  } catch { /* JameClaw may not be running yet. */ }
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
  const root = getReadableRoot()
  if (!root) {
    return ""
  }

  const clone = root.cloneNode(true)
  clone.querySelectorAll("script, style, noscript, nav, footer, aside, form, button, [aria-hidden='true'], [role='navigation'], [role='banner'], [role='contentinfo']").forEach((element) => element.remove())
  const text = clone.innerText.replace(/\s+/g, " ").trim()
  if (text.length <= PAGE_TEXT_LIMIT) {
    return text
  }

  return `${text.slice(0, PAGE_TEXT_LIMIT)}...`
}

function getReadableRoot() {
  const candidates = Array.from(document.querySelectorAll("main, article, [role='main']"))
    .filter((element) => element instanceof HTMLElement && element.innerText.trim().length > 300)
    .sort((a, b) => b.innerText.length - a.innerText.length)
  return candidates[0] || document.body
}

function pageDescription() {
  return document.querySelector("meta[name='description']")?.getAttribute("content")?.trim() || ""
}

function pageOutline() {
  const root = getReadableRoot()
  if (!root) return []
  return Array.from(root.querySelectorAll("h1, h2, h3"))
    .map((heading) => `${heading.tagName.toLowerCase()}: ${heading.textContent?.replace(/\s+/g, " ").trim() || ""}`)
    .filter(Boolean)
    .slice(0, PAGE_OUTLINE_LIMIT)
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
    description: pageDescription(),
    outline: pageOutline(),
    selection,
    pageText: getPageText(),
  })

  return false
})

syncDockFromStorage()
getBrowserClientID().then((id) => { browserClientID = id; void pollBrowserCommands(); setInterval(() => void pollBrowserCommands(), 1000) })

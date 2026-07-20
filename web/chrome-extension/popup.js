const BOOTSTRAP_URLS = [
  "http://127.0.0.1:18800/api/extension/bootstrap",
  "http://localhost:18800/api/extension/bootstrap",
]
const BOOTSTRAP_TIMEOUT_MS = 1500
const WEBSOCKET_TIMEOUT_MS = 5000
const MAX_RETRY_DELAY_MS = 2000
const SESSION_ID_KEY = "jameclaw-extension-session-id"

const messagesEl = document.getElementById("messages")
const statusEl = document.getElementById("status")
const composerEl = document.getElementById("composer")
const inputEl = document.getElementById("input")
const sendEl = document.getElementById("send")
const refreshContextEl = document.getElementById("refresh-context")
const conversationExplorerEl = document.getElementById("conversation-explorer")
const conversationPanelEl = document.getElementById("conversation-panel")
const conversationListEl = document.getElementById("conversation-list")
const newConversationEl = document.getElementById("new-conversation")
const titleEl = document.querySelector(".title")
const pageContextEl = document.getElementById("page-context")
const contextTitleEl = document.getElementById("context-title")
const contextDetailEl = document.getElementById("context-detail")
const isDock = document.body.classList.contains("dock")

let socket = null
let currentAssistantMessage = null
let pendingContext = null
let reconnectTimer = null
let bootstrapRetryTimer = null
let reconnectAttempts = 0
let bootstrapRetryAttempts = 0
let lastBootstrap = null
let sessionId = null

if (titleEl) {
  titleEl.textContent = "Jame"
}

function scrollToBottom() {
  messagesEl.scrollTop = messagesEl.scrollHeight
}

function setStatus(message) {
  statusEl.textContent = message || ""
}

function errorMessage(error) {
  if (error instanceof Error) {
    return error.message
  }
  return String(error || "Unknown error")
}

function setComposerEnabled(enabled) {
  inputEl.disabled = !enabled
  sendEl.disabled = !enabled
}

function clearReconnectTimer() {
  if (reconnectTimer !== null) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
}

function clearBootstrapRetryTimer() {
  if (bootstrapRetryTimer !== null) {
    clearTimeout(bootstrapRetryTimer)
    bootstrapRetryTimer = null
  }
}

function closeSocketForSessionChange() {
  clearReconnectTimer()
  clearBootstrapRetryTimer()
  lastBootstrap = null
  reconnectAttempts = 0
  bootstrapRetryAttempts = 0
  if (socket) {
    try {
      socket.close(1000, "session change")
    } catch {
      // Ignore close errors from a socket that is already closing.
    }
  }
  socket = null
  currentAssistantMessage = null
  setComposerEnabled(false)
}

function saveSessionId(nextSessionId) {
  return new Promise((resolve) => {
    sessionId = nextSessionId
    if (!chrome.storage?.local) {
      resolve()
      return
    }
    chrome.storage.local.set({ [SESSION_ID_KEY]: nextSessionId }, () => {
      resolve()
    })
  })
}

function getOrCreateSessionId() {
  return new Promise((resolve) => {
    if (!chrome.storage?.local) {
      resolve(crypto.randomUUID())
      return
    }

    chrome.storage.local.get([SESSION_ID_KEY], (result) => {
      if (chrome.runtime.lastError) {
        resolve(crypto.randomUUID())
        return
      }

      const stored = typeof result?.[SESSION_ID_KEY] === "string" ? result[SESSION_ID_KEY] : ""
      if (stored) {
        resolve(stored)
        return
      }

      const generated = crypto.randomUUID()
      chrome.storage.local.set({ [SESSION_ID_KEY]: generated }, () => {
        resolve(generated)
      })
    })
  })
}

function extensionApiOrigins() {
  return BOOTSTRAP_URLS.map((url) => new URL(url).origin)
}

async function fetchExtensionJSON(path) {
  let lastError = null

  for (const origin of extensionApiOrigins()) {
    const controller = new AbortController()
    const timeoutId = window.setTimeout(
      () => controller.abort(),
      BOOTSTRAP_TIMEOUT_MS,
    )

    try {
      const response = await fetch(`${origin}${path}`, {
        method: "GET",
        headers: { Accept: "application/json" },
        signal: controller.signal,
      })

      if (!response.ok) {
        throw new Error(`Request failed: ${response.status}`)
      }

      return await response.json()
    } catch (error) {
      lastError = error
    } finally {
      window.clearTimeout(timeoutId)
    }
  }

  throw lastError || new Error("Could not reach local JameClaw.")
}

function scheduleReconnect(reason = "") {
  if (!lastBootstrap || reconnectTimer !== null) {
    return
  }

  const delay = Math.min(250 * 2 ** reconnectAttempts, MAX_RETRY_DELAY_MS)
  reconnectAttempts += 1
  setStatus(reason ? `${reason} Retrying WebSocket…` : "Retrying WebSocket…")
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null
    if (!lastBootstrap) {
      return
    }
    connectWebSocket(lastBootstrap.wsUrl, lastBootstrap.token)
  }, delay)
}

function scheduleBootstrapRetry(reason = "") {
  if (bootstrapRetryTimer !== null) {
    return
  }

  const delay = Math.min(250 * 2 ** bootstrapRetryAttempts, MAX_RETRY_DELAY_MS)
  bootstrapRetryAttempts += 1
  setStatus(reason ? `${reason} Retrying bootstrap…` : "Retrying bootstrap…")
  bootstrapRetryTimer = setTimeout(() => {
    bootstrapRetryTimer = null
    void bootstrap()
  }, delay)
}

function appendMessage(role, content) {
  const empty = messagesEl.querySelector(".empty")
  if (empty) {
    empty.remove()
  }

  const item = document.createElement("div")
  item.className = `message ${role}`
  if (role === "error") {
    const title = document.createElement("div")
    title.className = "error-title"
    title.textContent = "Couldn’t complete that"
    const detail = document.createElement("div")
    detail.className = "error-content"
    detail.textContent = content
    item.append(title, detail)
  } else {
    item.textContent = content
  }
  messagesEl.appendChild(item)
  scrollToBottom()
  return item
}

function renderConversationMessages(messages) {
  messagesEl.innerHTML = ""
  currentAssistantMessage = null

  for (const message of messages || []) {
    const role = message?.role === "user" ? "user" : "assistant"
    const content = String(message?.content || "").trim()
    if (content) {
      appendMessage(role, content)
    }
  }

  ensureEmptyState()
}

function ensureEmptyState() {
  // The empty chat is intentionally blank. Page context is shown in the
  // compact context card above the conversation instead of a large placeholder.
}

function setConversationPanelOpen(open) {
  if (!conversationPanelEl || !conversationExplorerEl) {
    return
  }
  conversationPanelEl.hidden = !open
  conversationExplorerEl.classList.toggle("is-active", open)
  conversationExplorerEl.setAttribute("aria-expanded", String(open))
}

function formatConversationMeta(item) {
  const count = Number(item?.message_count || 0)
  const countText = count === 1 ? "1 message" : `${count} messages`
  const updated = item?.updated ? new Date(item.updated) : null
  if (!updated || Number.isNaN(updated.getTime())) {
    return countText
  }
  return `${countText} · ${updated.toLocaleDateString([], {
    month: "short",
    day: "numeric",
  })}`
}

function renderConversationList(items) {
  if (!conversationListEl) {
    return
  }

  conversationListEl.innerHTML = ""
  if (!Array.isArray(items) || items.length === 0) {
    const empty = document.createElement("div")
    empty.className = "conversation-empty"
    empty.textContent = "No saved conversations yet."
    conversationListEl.appendChild(empty)
    return
  }

  for (const item of items) {
    if (!item?.id) {
      continue
    }

    const button = document.createElement("button")
    button.className = "conversation-item"
    button.type = "button"
    button.classList.toggle("is-active", item.id === sessionId)
    button.dataset.sessionId = item.id

    const title = document.createElement("span")
    title.className = "conversation-title"
    title.textContent = item.title || item.preview || "Untitled conversation"

    const meta = document.createElement("span")
    meta.className = "conversation-meta"
    meta.textContent = item.preview || formatConversationMeta(item)

    button.append(title, meta)
    button.addEventListener("click", () => {
      void switchConversation(item.id)
    })
    conversationListEl.appendChild(button)
  }
}

async function loadConversationList() {
  if (!conversationListEl) {
    return
  }

  conversationListEl.innerHTML = '<div class="conversation-empty">Loading conversations…</div>'
  try {
    const items = await fetchExtensionJSON("/api/extension/sessions?offset=0&limit=50")
    renderConversationList(items)
  } catch (error) {
    conversationListEl.innerHTML = ""
    const empty = document.createElement("div")
    empty.className = "conversation-empty"
    empty.textContent = errorMessage(error) || "Could not load conversations."
    conversationListEl.appendChild(empty)
  }
}

async function switchConversation(nextSessionId) {
  if (!nextSessionId || nextSessionId === sessionId) {
    setConversationPanelOpen(false)
    return
  }

  closeSocketForSessionChange()
  setStatus("Loading conversation…")

  try {
    const detail = await fetchExtensionJSON(
      `/api/extension/sessions/${encodeURIComponent(nextSessionId)}`,
    )
    await saveSessionId(nextSessionId)
    renderConversationMessages(detail?.messages || [])
    setConversationPanelOpen(false)
    void bootstrap()
  } catch (error) {
    setStatus(errorMessage(error) || "Could not load conversation.")
  }
}

async function startNewConversation() {
  closeSocketForSessionChange()
  await saveSessionId(crypto.randomUUID())
  messagesEl.innerHTML = ""
  ensureEmptyState()
  setConversationPanelOpen(false)
  void bootstrap()
}

function buildContextBlock(context) {
  if (!context) {
    return ""
  }

  return [
    context.title ? `Page title: ${context.title}` : "",
    context.url ? `Page URL: ${context.url}` : "",
    context.description ? `Page summary: ${context.description}` : "",
    context.outline?.length ? `Page outline:\n${context.outline.join("\n")}` : "",
    context.selection ? `Selected text:\n${context.selection}` : "",
    context.pageText ? `Page content excerpt:\n${context.pageText}` : "",
  ]
    .filter(Boolean)
    .join("\n\n")
    .trim()
}

function buildOutgoingMessage(text) {
  const contextBlock = buildContextBlock(pendingContext)
  return contextBlock ? `${text}\n\n${contextBlock}` : text
}

function normalizeContext(context) {
  return {
    title: (context?.title || "").trim(),
    url: (context?.url || "").trim(),
    description: (context?.description || "").trim(),
    outline: Array.isArray(context?.outline) ? context.outline.map((item) => String(item).trim()).filter(Boolean).slice(0, 24) : [],
    selection: (context?.selection || "").trim(),
    pageText: (context?.pageText || "").trim(),
  }
}

function renderPageContext(context) {
  if (!pageContextEl || !contextTitleEl || !contextDetailEl) return
  if (!context?.title && !context?.selection) {
    pageContextEl.hidden = true
    return
  }
  pageContextEl.hidden = false
  contextTitleEl.textContent = context.selection ? "Selected text is attached" : context.title || "Current page is attached"
  const details = []
  if (context.selection) details.push(`${context.selection.length.toLocaleString()} selected characters`)
  else if (context.outline?.length) details.push(`${context.outline.length} headings captured`)
  if (context.pageText) details.push(`${context.pageText.length.toLocaleString()} characters ready`)
  contextDetailEl.textContent = details.join(" · ") || "Current page context is ready"
}

function requestPageContext() {
  chrome.runtime.sendMessage(
    { type: "jameclaw-extension-request-context" },
    (response) => {
      if (chrome.runtime.lastError) {
        setStatus(
          chrome.runtime.lastError.message ||
            "Could not read the active page context.",
        )
        return
      }

      if (!response?.ok) {
        setStatus(response?.error || "Could not read the active page context.")
        return
      }

      pendingContext = normalizeContext(response.context)
      renderPageContext(pendingContext)
      if (pendingContext.selection) {
        setStatus("Using selected text.")
        return
      }

      if (!socket || socket.readyState !== WebSocket.OPEN) {
        setStatus("Connecting…")
      } else {
        setStatus("")
      }
    },
  )
}

function connectWebSocket(wsUrl, token) {
  lastBootstrap = { wsUrl, token }
  clearReconnectTimer()
  const separator = wsUrl.includes("?") ? "&" : "?"
  const url = `${wsUrl}${separator}token=${encodeURIComponent(token)}&session_id=${encodeURIComponent(sessionId)}`
  setStatus(`Opening WebSocket ${new URL(wsUrl).host}…`)
  socket = new WebSocket(url, [`token.${token}`])
  const activeSocket = socket
  const timeoutId = window.setTimeout(() => {
    if (activeSocket.readyState === WebSocket.CONNECTING) {
      setStatus("WebSocket timed out. Retrying…")
      activeSocket.close()
    }
  }, WEBSOCKET_TIMEOUT_MS)

  socket.addEventListener("open", () => {
    window.clearTimeout(timeoutId)
    reconnectAttempts = 0
    bootstrapRetryAttempts = 0
    clearBootstrapRetryTimer()
    setStatus(pendingContext?.selection ? "Using selected text." : "")
    setComposerEnabled(true)
  })

  socket.addEventListener("close", () => {
    window.clearTimeout(timeoutId)
    setComposerEnabled(false)
    scheduleReconnect("WebSocket closed.")
  })

  socket.addEventListener("error", () => {
    window.clearTimeout(timeoutId)
    setComposerEnabled(false)
    scheduleReconnect("Could not connect to local JameClaw WebSocket.")
  })

  socket.addEventListener("message", (event) => {
    let message
    try {
      message = JSON.parse(event.data)
    } catch {
      return
    }

    const payload = message.payload || {}

    switch (message.type) {
      case "typing.start":
        setStatus("Thinking…")
        break

      case "typing.stop":
        setStatus("")
        break

      case "message.create": {
        const content = payload.content || ""
        currentAssistantMessage = appendMessage("assistant", content)
        setStatus("")
        break
      }

      case "message.update": {
        const content = payload.content || ""
        if (currentAssistantMessage) {
          currentAssistantMessage.textContent = content
          scrollToBottom()
        } else {
          currentAssistantMessage = appendMessage("assistant", content)
        }
        setStatus("")
        break
      }

      case "error":
        appendMessage("error", payload.message || payload.error || "Request failed.")
        setStatus("")
        break

      default:
        break
    }
  })
}

async function fetchBootstrap(url) {
  const controller = new AbortController()
  const timeoutId = window.setTimeout(() => controller.abort(), BOOTSTRAP_TIMEOUT_MS)

  try {
    setStatus(`Bootstrap ${new URL(url).host}…`)
    const response = await fetch(url, {
      method: "GET",
      headers: { Accept: "application/json" },
      signal: controller.signal,
    })

    if (!response.ok) {
      throw new Error(`Bootstrap failed: ${response.status}`)
    }

    return await response.json()
  } finally {
    window.clearTimeout(timeoutId)
  }
}

async function bootstrap() {
  sessionId = await getOrCreateSessionId()
  ensureEmptyState()
  setComposerEnabled(false)
  setStatus("Connecting…")
  requestPageContext()
  clearBootstrapRetryTimer()

  try {
    let data = null
    let lastError = null

    for (const url of BOOTSTRAP_URLS) {
      try {
        data = await fetchBootstrap(url)
        break
      } catch (error) {
        lastError = error
      }
    }

    if (!data?.token || !data?.ws_url) {
      throw lastError || new Error("Missing JameClaw token or websocket URL.")
    }

    connectWebSocket(data.ws_url, data.token)
  } catch (error) {
    const message =
      error instanceof Error && error.name === "AbortError"
        ? "Bootstrap timed out"
        : errorMessage(error) || "Could not reach local JameClaw on 127.0.0.1:18800."
    scheduleBootstrapRetry(message)
  }
}

composerEl.addEventListener("submit", (event) => {
  event.preventDefault()

  const text = inputEl.value.trim()
  if (!text || !socket || socket.readyState !== WebSocket.OPEN) {
    return
  }

  currentAssistantMessage = null
  appendMessage("user", text)
  socket.send(
    JSON.stringify({
      type: "message.send",
      id: `msg-${Date.now()}`,
      payload: {
        content: buildOutgoingMessage(text),
      },
    }),
  )
  inputEl.value = ""
  setStatus("Thinking…")
  requestPageContext()
})

inputEl.addEventListener("keydown", (event) => {
  if (event.key === "Enter" && !event.shiftKey) {
    event.preventDefault()
    composerEl.requestSubmit()
  }
})

conversationExplorerEl?.addEventListener("click", () => {
  const willOpen = conversationPanelEl?.hidden !== false
  setConversationPanelOpen(willOpen)
  if (willOpen) {
    void loadConversationList()
  }
})

document.addEventListener("click", (event) => {
  if (!conversationPanelEl || !conversationExplorerEl || conversationPanelEl.hidden) return
  const target = event.target
  if (target instanceof Node && !conversationPanelEl.contains(target) && !conversationExplorerEl.contains(target)) {
    setConversationPanelOpen(false)
  }
})

newConversationEl?.addEventListener("click", () => {
  void startNewConversation()
})

refreshContextEl.addEventListener("click", () => {
  if (isDock) {
    window.close()
    return
  }

  chrome.runtime.sendMessage(
    { type: "jameclaw-extension-open-panel-window" },
    (response) => {
      if (chrome.runtime.lastError) {
        setStatus(
          chrome.runtime.lastError.message || "Could not open JameClaw window.",
        )
        return
      }

      if (!response?.ok) {
        setStatus(response?.error || "Could not open JameClaw window.")
        return
      }
      window.close()
    },
  )
})

refreshContextEl.innerHTML = isDock ? "Close" : '<span aria-hidden="true">↗</span> Pop out'

void bootstrap()

const BOOTSTRAP_URL = "http://localhost:18800/api/extension/bootstrap"
const BOOTSTRAP_TIMEOUT_MS = 1500
const MAX_RETRY_DELAY_MS = 2000
const SESSION_ID_KEY = "jameclaw-extension-session-id"

const messagesEl = document.getElementById("messages")
const statusEl = document.getElementById("status")
const composerEl = document.getElementById("composer")
const inputEl = document.getElementById("input")
const sendEl = document.getElementById("send")
const refreshContextEl = document.getElementById("refresh-context")
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

function scrollToBottom() {
  messagesEl.scrollTop = messagesEl.scrollHeight
}

function setStatus(message) {
  statusEl.textContent = message || ""
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

function scheduleReconnect() {
  if (!lastBootstrap || reconnectTimer !== null) {
    return
  }

  const delay = Math.min(250 * 2 ** reconnectAttempts, MAX_RETRY_DELAY_MS)
  reconnectAttempts += 1
  setStatus("Reconnecting…")
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null
    if (!lastBootstrap) {
      return
    }
    connectWebSocket(lastBootstrap.wsUrl, lastBootstrap.token)
  }, delay)
}

function scheduleBootstrapRetry() {
  if (bootstrapRetryTimer !== null) {
    return
  }

  const delay = Math.min(250 * 2 ** bootstrapRetryAttempts, MAX_RETRY_DELAY_MS)
  bootstrapRetryAttempts += 1
  setStatus("Connecting…")
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
  item.textContent = content
  messagesEl.appendChild(item)
  scrollToBottom()
  return item
}

function ensureEmptyState() {
  if (messagesEl.children.length > 0) {
    return
  }

  const empty = document.createElement("div")
  empty.className = "empty"
  empty.textContent =
    "Start typing. The current page context is attached automatically. Select text on the website before opening the popup if you want JameClaw to focus on it."
  messagesEl.appendChild(empty)
}

function buildContextBlock(context) {
  if (!context) {
    return ""
  }

  return [
    context.title ? `Page title: ${context.title}` : "",
    context.url ? `Page URL: ${context.url}` : "",
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
    selection: (context?.selection || "").trim(),
    pageText: (context?.pageText || "").trim(),
  }
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
  socket = new WebSocket(url)

  socket.addEventListener("open", () => {
    reconnectAttempts = 0
    bootstrapRetryAttempts = 0
    clearBootstrapRetryTimer()
    setStatus(pendingContext?.selection ? "Using selected text." : "")
    setComposerEnabled(true)
  })

  socket.addEventListener("close", () => {
    setStatus("Connection closed.")
    setComposerEnabled(false)
    scheduleReconnect()
  })

  socket.addEventListener("error", () => {
    setStatus("Could not connect to local JameClaw.")
    setComposerEnabled(false)
    scheduleReconnect()
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

async function bootstrap() {
  sessionId = await getOrCreateSessionId()
  ensureEmptyState()
  setComposerEnabled(false)
  setStatus("Connecting…")
  requestPageContext()
  clearBootstrapRetryTimer()

  const controller = new AbortController()
  const timeoutId = window.setTimeout(() => controller.abort(), BOOTSTRAP_TIMEOUT_MS)

  try {
    const response = await fetch(BOOTSTRAP_URL, {
      method: "GET",
      headers: { Accept: "application/json" },
      signal: controller.signal,
    })

    if (!response.ok) {
      throw new Error(`Bootstrap failed: ${response.status}`)
    }

    const data = await response.json()
    if (!data?.token || !data?.ws_url) {
      throw new Error("Missing JameClaw token or websocket URL.")
    }

    connectWebSocket(data.ws_url, data.token)
  } catch (error) {
    if (error instanceof Error && error.name === "AbortError") {
      setStatus("Connecting…")
    } else {
      setStatus(
        error instanceof Error
          ? error.message
          : "Could not reach local JameClaw on localhost:18800.",
      )
    }
    scheduleBootstrapRetry()
  } finally {
    window.clearTimeout(timeoutId)
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

refreshContextEl.addEventListener("click", () => {
  if (isDock) {
    chrome.runtime.sendMessage(
      { type: "jameclaw-extension-set-dock-state", enabled: false },
      (response) => {
        if (chrome.runtime.lastError) {
          setStatus(
            chrome.runtime.lastError.message || "Could not close dock.",
          )
          return
        }

        if (!response?.ok) {
          setStatus(response?.error || "Could not close dock.")
          return
        }
      },
    )
    return
  }

  chrome.runtime.sendMessage(
    { type: "jameclaw-extension-set-dock-state", enabled: true },
    (response) => {
      if (chrome.runtime.lastError) {
        setStatus(
          chrome.runtime.lastError.message || "Could not open dock.",
        )
        return
      }

      if (!response?.ok) {
        setStatus(response?.error || "Could not open dock.")
        return
      }
      window.close()
    },
  )
})

refreshContextEl.textContent = isDock ? "Undock" : "Dock"

void bootstrap()

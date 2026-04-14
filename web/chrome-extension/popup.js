const BOOTSTRAP_URL = "http://localhost:18800/api/extension/bootstrap"

const messagesEl = document.getElementById("messages")
const statusEl = document.getElementById("status")
const composerEl = document.getElementById("composer")
const inputEl = document.getElementById("input")
const sendEl = document.getElementById("send")

let socket = null
let currentAssistantMessage = null
let pendingContext = null
const sessionId = crypto.randomUUID()

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
  empty.textContent = "Start typing. The current page context is attached automatically."
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
      if (!response?.ok) {
        return
      }

      pendingContext = normalizeContext(response.context)
    },
  )
}

function connectWebSocket(wsUrl, token) {
  const separator = wsUrl.includes("?") ? "&" : "?"
  const url = `${wsUrl}${separator}token=${encodeURIComponent(token)}&session_id=${encodeURIComponent(sessionId)}`
  socket = new WebSocket(url)

  socket.addEventListener("open", () => {
    setStatus("")
    setComposerEnabled(true)
  })

  socket.addEventListener("close", () => {
    setStatus("Connection closed.")
    setComposerEnabled(false)
  })

  socket.addEventListener("error", () => {
    setStatus("Could not connect to local JameClaw.")
    setComposerEnabled(false)
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
  ensureEmptyState()
  setComposerEnabled(false)
  requestPageContext()

  try {
    const response = await fetch(BOOTSTRAP_URL, {
      method: "GET",
      headers: { Accept: "application/json" },
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
    setStatus(
      error instanceof Error
        ? error.message
        : "Could not reach local JameClaw on localhost:18800.",
    )
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

bootstrap()

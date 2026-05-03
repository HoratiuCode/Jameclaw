import { useEffect, useRef, useState } from "react"
import { toast, Toaster } from "sonner"

import { AssistantMessage } from "@/components/chat/assistant-message"
import { ChatComposer } from "@/components/chat/chat-composer"
import { TypingIndicator } from "@/components/chat/typing-indicator"
import { UserMessage } from "@/components/chat/user-message"
import { useChatModels } from "@/hooks/use-chat-models"
import { useGateway } from "@/hooks/use-gateway"
import { useJameChat } from "@/hooks/use-jame-chat"

type ExtensionContext = {
  title?: string
  url?: string
  selection?: string
  pageText?: string
}

type ExtensionMessage = {
  type?: string
  context?: ExtensionContext
}

const CONTEXT_LIMIT = 4000

function trimContext(text: string) {
  if (text.length <= CONTEXT_LIMIT) {
    return text
  }

  return `${text.slice(0, CONTEXT_LIMIT)}...`
}

function buildContextBlock(context: ExtensionContext) {
  const parts = [
    context.title ? `Page title: ${context.title}` : "",
    context.url ? `Page URL: ${context.url}` : "",
    context.selection ? `Selected text:\n${context.selection}` : "",
    context.pageText ? `Page content excerpt:\n${context.pageText}` : "",
  ].filter(Boolean)

  return parts.join("\n\n").trim()
}

export function ExtensionPage() {
  const scrollRef = useRef<HTMLDivElement>(null)
  const [input, setInput] = useState("")
  const [pageContext, setPageContext] = useState<ExtensionContext>({})

  const {
    messages,
    connectionState,
    errorMessage,
    isTyping,
    sendMessage,
  } = useJameChat()

  const { state: gwState, canStart, startReason, pid, owned } = useGateway()
  const isGatewayRunning = gwState === "running"
  const { defaultModelName } = useChatModels({ isConnected: isGatewayRunning })
  const canSend =
    isGatewayRunning &&
    connectionState === "connected" &&
    Boolean(defaultModelName)

  const disabledReason = !defaultModelName
    ? "Choose a default model in JameClaw before sending messages."
    : !canStart && startReason
      ? startReason
      : gwState === "running" && !owned
        ? `Another gateway is already running${pid ? ` (PID ${pid})` : ""}.`
      : !isGatewayRunning
        ? "The gateway is not running."
        : connectionState === "connecting"
          ? "Connecting to JameClaw..."
          : connectionState === "error"
            ? (errorMessage ?? "Could not connect to the Jame chat session.")
            : null

  useEffect(() => {
    const handleMessage = (event: MessageEvent<ExtensionMessage>) => {
      if (event.data?.type !== "jameclaw-extension-context") {
        return
      }

      setPageContext({
        title: event.data.context?.title?.trim() || "",
        url: event.data.context?.url?.trim() || "",
        selection: trimContext(event.data.context?.selection?.trim() || ""),
        pageText: trimContext(event.data.context?.pageText?.trim() || ""),
      })
    }

    window.addEventListener("message", handleMessage)
    return () => window.removeEventListener("message", handleMessage)
  }, [])

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messages, isTyping])

  const handleSend = () => {
    const trimmed = input.trim()
    if (!trimmed) {
      return
    }

    if (disabledReason) {
      toast.error(disabledReason)
      return
    }

    const contextBlock = buildContextBlock(pageContext)
    const message = contextBlock
      ? `${trimmed}\n\n${contextBlock}`
      : trimmed

    if (sendMessage(message)) {
      setInput("")
    } else {
      toast.error(
        "Web Console could not send the message. Make sure JameClaw is connected and try again.",
      )
    }
  }

  return (
    <div className="flex h-dvh flex-col overflow-hidden bg-white text-slate-950">
      <div ref={scrollRef} className="min-h-0 flex-1 overflow-y-auto px-3 py-3">
        <div className="mx-auto flex w-full max-w-3xl flex-col gap-4 pb-4">
          {messages.length === 0 && (
            <div className="rounded-2xl border border-slate-200 bg-slate-50 p-4 text-sm leading-6 text-slate-600">
              Start typing. The current page context is attached automatically.
            </div>
          )}

          {messages.map((msg) => (
            <div key={msg.id} className="flex w-full">
              {msg.role === "assistant" ? (
                <AssistantMessage content={msg.content} timestamp={msg.timestamp} />
              ) : (
                <UserMessage content={msg.content} />
              )}
            </div>
          ))}

          {isTyping && <TypingIndicator />}
        </div>
      </div>

      <ChatComposer
        input={input}
        onInputChange={setInput}
        onSend={handleSend}
        disabledReason={disabledReason}
        isConnected={canSend}
        hasDefaultModel={Boolean(defaultModelName)}
      />
      <Toaster position="bottom-center" />
    </div>
  )
}

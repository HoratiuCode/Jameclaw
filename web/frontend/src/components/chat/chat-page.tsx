import { IconPlus } from "@tabler/icons-react"
import { useNavigate, useSearch } from "@tanstack/react-router"
import { useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { AssistantMessage } from "@/components/chat/assistant-message"
import { ChatComposer } from "@/components/chat/chat-composer"
import { ChatEmptyState } from "@/components/chat/chat-empty-state"
import { ModelSelector } from "@/components/chat/model-selector"
import { SessionHistoryMenu } from "@/components/chat/session-history-menu"
import { TypingIndicator } from "@/components/chat/typing-indicator"
import { UserMessage } from "@/components/chat/user-message"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { useAgentDisplayName } from "@/hooks/use-agent-display-name"
import { useChatModels } from "@/hooks/use-chat-models"
import { useGateway } from "@/hooks/use-gateway"
import { useJameChat } from "@/hooks/use-jame-chat"
import { useSessionHistory } from "@/hooks/use-session-history"

const MAX_UPLOAD_BYTES = 25 * 1024 * 1024

export function ChatPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { prompt } = useSearch({ from: "/" })
  const scrollRef = useRef<HTMLDivElement>(null)
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const mediaStreamRef = useRef<MediaStream | null>(null)
  const recordedChunksRef = useRef<Blob[]>([])
  const [isAtBottom, setIsAtBottom] = useState(true)
  const [hasScrolled, setHasScrolled] = useState(false)
  const [input, setInput] = useState("")
  const [isRecording, setIsRecording] = useState(false)
  const [isUploading, setIsUploading] = useState(false)
  const agentName = useAgentDisplayName()

  const {
    messages,
    connectionState,
    errorMessage,
    isTyping,
    activeSessionId,
    sendMessage,
    sendFile,
    sendVoice,
    switchSession,
    newChat,
  } = useJameChat()

  const { state: gwState, canStart, startReason, pid, owned } = useGateway()
  const isGatewayRunning = gwState === "running"

  const {
    defaultModelName,
    hasConfiguredModels,
    apiKeyModels,
    oauthModels,
    localModels,
    handleSetDefault,
  } = useChatModels({ isConnected: isGatewayRunning })
  const canSend = isGatewayRunning && Boolean(defaultModelName)
  const gatewayStopHint =
    gwState === "running" && !owned
      ? `Another gateway is already running${pid ? ` (PID ${pid})` : ""}. Use Stop in the top bar or run ${pid ? `kill ${pid} then kill -9 ${pid}` : "pkill -f 'jameclaw gateway'"}.`
      : null
  const disabledReason = !defaultModelName
    ? "Choose a default model before sending a message."
    : !canStart && startReason
      ? startReason
      : gatewayStopHint
        ? gatewayStopHint
        : !isGatewayRunning
          ? "The gateway is not running."
          : connectionState === "error"
            ? (errorMessage ?? "The Web Console could not connect to JameClaw.")
            : null
  const connectionNotice =
    isGatewayRunning && connectionState === "offline"
      ? (errorMessage ??
        "Your phone is offline. JameClaw will reconnect when the network returns.")
      : isGatewayRunning && connectionState === "reconnecting"
        ? (errorMessage ?? "Reconnecting to JameClaw...")
        : isGatewayRunning && connectionState === "connecting"
          ? "Connecting the Web Console to JameClaw..."
          : null
  const hasActiveAssistantPlaceholder =
    isTyping && messages[messages.length - 1]?.role === "assistant"

  useEffect(() => {
    if (!prompt) {
      return
    }
    setInput(prompt)
    void navigate({
      to: "/",
      search: { prompt: undefined },
      replace: true,
    })
  }, [navigate, prompt])

  const {
    sessions,
    hasMore,
    loadError,
    loadErrorMessage,
    observerRef,
    loadSessions,
    handleDeleteSession,
  } = useSessionHistory({
    activeSessionId,
    onDeletedActiveSession: newChat,
  })

  const syncScrollState = (element: HTMLDivElement) => {
    const { scrollTop, scrollHeight, clientHeight } = element
    setHasScrolled(scrollTop > 0)
    setIsAtBottom(scrollHeight - scrollTop <= clientHeight + 10)
  }

  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    syncScrollState(e.currentTarget)
  }

  useEffect(() => {
    if (scrollRef.current) {
      if (isAtBottom) {
        scrollRef.current.scrollTop = scrollRef.current.scrollHeight
      }
      syncScrollState(scrollRef.current)
    }
  }, [messages, isTyping, isAtBottom])

  const handleSend = () => {
    if (!input.trim()) return
    if (!canSend) {
      if (disabledReason) {
        toast.error(disabledReason)
      }
      return
    }
    if (sendMessage(input.trim())) {
      setInput("")
    } else {
      toast.error(
        "Web Console could not send the message. Make sure JameClaw is connected and try again.",
      )
    }
  }

  const stopRecording = () => {
    const recorder = mediaRecorderRef.current
    if (recorder && recorder.state !== "inactive") {
      recorder.stop()
    }
  }

  const handleVoiceToggle = async () => {
    if (isRecording) {
      stopRecording()
      return
    }
    if (!canSend) {
      if (disabledReason) {
        toast.error(disabledReason)
      }
      return
    }
    if (
      !navigator.mediaDevices?.getUserMedia ||
      typeof MediaRecorder === "undefined"
    ) {
      toast.error("Voice recording is not available in this browser.")
      return
    }

    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      const preferredType = [
        "audio/webm;codecs=opus",
        "audio/webm",
        "audio/ogg;codecs=opus",
        "audio/mp4",
      ].find((type) => MediaRecorder.isTypeSupported(type))
      const recorder = preferredType
        ? new MediaRecorder(stream, { mimeType: preferredType })
        : new MediaRecorder(stream)

      recordedChunksRef.current = []
      mediaStreamRef.current = stream
      mediaRecorderRef.current = recorder

      recorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          recordedChunksRef.current.push(event.data)
        }
      }
      recorder.onstop = () => {
        const chunks = recordedChunksRef.current
        const type = recorder.mimeType || preferredType || "audio/webm"
        recordedChunksRef.current = []
        mediaRecorderRef.current = null
        mediaStreamRef.current?.getTracks().forEach((track) => track.stop())
        mediaStreamRef.current = null
        setIsRecording(false)

        if (chunks.length === 0) {
          toast.error("No audio was recorded.")
          return
        }
        void sendVoice(new Blob(chunks, { type }))
      }
      recorder.onerror = () => {
        toast.error("Voice recording failed.")
        mediaStreamRef.current?.getTracks().forEach((track) => track.stop())
        mediaStreamRef.current = null
        mediaRecorderRef.current = null
        setIsRecording(false)
      }

      recorder.start()
      setIsRecording(true)
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : "Could not start voice recording."
      toast.error(message)
      mediaStreamRef.current?.getTracks().forEach((track) => track.stop())
      mediaStreamRef.current = null
      mediaRecorderRef.current = null
      setIsRecording(false)
    }
  }

  const handleFileSelect = async (files: FileList) => {
    if (!canSend) {
      if (disabledReason) {
        toast.error(disabledReason)
      }
      return
    }

    const selected = Array.from(files)
    if (selected.length === 0) {
      return
    }

    setIsUploading(true)
    try {
      for (const file of selected) {
        if (file.size > MAX_UPLOAD_BYTES) {
          toast.error(`${file.name} is larger than 25 MB.`)
          continue
        }
        const caption = input.trim()
        const sent = await sendFile(file, caption)
        if (sent && caption) {
          setInput("")
        }
      }
    } finally {
      setIsUploading(false)
    }
  }

  useEffect(() => {
    return () => {
      stopRecording()
      mediaStreamRef.current?.getTracks().forEach((track) => track.stop())
    }
  }, [])

  return (
    <div className="bg-background/95 flex h-full flex-col">
      <PageHeader
        title={t("navigation.chat")}
        className={`transition-shadow ${
          hasScrolled ? "shadow-sm" : "shadow-none"
        }`}
        titleExtra={
          hasConfiguredModels && (
            <ModelSelector
              defaultModelName={defaultModelName}
              apiKeyModels={apiKeyModels}
              oauthModels={oauthModels}
              localModels={localModels}
              onValueChange={handleSetDefault}
            />
          )
        }
      >
        <Button
          variant="secondary"
          size="sm"
          onClick={() => {
            void newChat()
          }}
          className="h-9 gap-2"
        >
          <IconPlus className="size-4" />
          <span className="hidden sm:inline">{t("chat.newChat")}</span>
        </Button>

        <SessionHistoryMenu
          sessions={sessions}
          activeSessionId={activeSessionId}
          hasMore={hasMore}
          loadError={loadError}
          loadErrorMessage={loadErrorMessage}
          observerRef={observerRef}
          onOpenChange={(open) => {
            if (open) {
              void loadSessions(true)
            }
          }}
          onSwitchSession={switchSession}
          onDeleteSession={handleDeleteSession}
        />
      </PageHeader>

      {connectionNotice && (
        <div className="border-border/70 bg-muted/70 text-muted-foreground mx-4 mt-2 rounded-lg border px-3 py-2 text-sm md:mx-8 lg:mx-24 xl:mx-48">
          {connectionNotice}
        </div>
      )}

      <div
        ref={scrollRef}
        onScroll={handleScroll}
        className="min-h-0 flex-1 overflow-y-auto px-4 py-6 md:px-8 lg:px-24 xl:px-48"
      >
        <div className="mx-auto flex w-full max-w-250 flex-col gap-8 pb-8">
          {messages.length === 0 && !isTyping && (
            <ChatEmptyState
              hasConfiguredModels={hasConfiguredModels}
              defaultModelName={defaultModelName}
              isConnected={isGatewayRunning}
              onPromptSelect={setInput}
            />
          )}

          {messages.map((msg, index) => (
            <div key={msg.id} className="flex w-full">
              {msg.role === "assistant" ? (
                <AssistantMessage
                  agentName={agentName}
                  content={msg.content}
                  timestamp={msg.timestamp}
                  isTyping={isTyping && index === messages.length - 1}
                  media={msg.media}
                />
              ) : (
                <UserMessage content={msg.content} />
              )}
            </div>
          ))}

          {isTyping && !hasActiveAssistantPlaceholder && (
            <TypingIndicator agentName={agentName} />
          )}
        </div>
      </div>

      <ChatComposer
        input={input}
        onInputChange={setInput}
        onSend={handleSend}
        onFileSelect={handleFileSelect}
        onVoiceToggle={handleVoiceToggle}
        disabledReason={disabledReason}
        isConnected={isGatewayRunning}
        hasDefaultModel={Boolean(defaultModelName)}
        isRecording={isRecording}
        canRecord={typeof MediaRecorder !== "undefined"}
        isUploading={isUploading}
      />
    </div>
  )
}

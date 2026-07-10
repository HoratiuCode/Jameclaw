import { normalizeUnixTimestamp } from "@/features/chat/state"
import type { ChatMediaAttachment } from "@/store/chat"
import { updateChatStore } from "@/store/chat"

export interface JameMessage {
  type: string
  id?: string
  session_id?: string
  timestamp?: number | string
  payload?: Record<string, unknown>
}

export function handleJameMessage(
  message: JameMessage,
  expectedSessionId: string,
) {
  if (message.session_id && message.session_id !== expectedSessionId) {
    return
  }

  const payload = message.payload || {}

  switch (message.type) {
    case "message.create": {
      const content = (payload.content as string) || ""
      const messageId = (payload.message_id as string) || `jame-${Date.now()}`
      const timestamp =
        message.timestamp !== undefined &&
        Number.isFinite(Number(message.timestamp))
          ? normalizeUnixTimestamp(Number(message.timestamp))
          : Date.now()

      updateChatStore((prev) => ({
        messages: [
          ...prev.messages,
          {
            id: messageId,
            role: "assistant",
            content,
            timestamp,
          },
        ],
        isTyping: false,
      }))
      break
    }

    case "message.update": {
      const content = (payload.content as string) || ""
      const messageId = payload.message_id as string
      if (!messageId) {
        break
      }

      updateChatStore((prev) => ({
        messages: prev.messages.map((msg) =>
          msg.id === messageId ? { ...msg, content } : msg,
        ),
      }))
      break
    }

    case "media.create": {
      const data = (payload.data as string) || ""
      if (!data) {
        break
      }
      const contentType =
        (payload.content_type as string) || "application/octet-stream"
      const filename = (payload.filename as string) || "media"
      const kind = normalizeMediaKind(payload.kind, contentType)
      const caption = (payload.caption as string) || ""
      const messageId =
        (payload.message_id as string) || `jame-media-${Date.now()}`
      const timestamp =
        message.timestamp !== undefined &&
        Number.isFinite(Number(message.timestamp))
          ? normalizeUnixTimestamp(Number(message.timestamp))
          : Date.now()
      const url = data.startsWith("data:")
        ? data
        : `data:${contentType};base64,${data}`
      const media: ChatMediaAttachment = {
        url,
        filename,
        contentType,
        kind,
      }

      updateChatStore((prev) => ({
        messages: [
          ...prev.messages,
          {
            id: messageId,
            role: "assistant",
            content: caption,
            timestamp,
            media: [media],
          },
        ],
        isTyping: false,
      }))
      break
    }

    case "typing.start":
      updateChatStore({ isTyping: true, errorMessage: null })
      break

    case "typing.stop":
      updateChatStore({ isTyping: false })
      break

    case "error":
      console.error("Jame error:", payload)
      updateChatStore({
        isTyping: false,
        errorMessage:
          (payload.message as string) ||
          (payload.error as string) ||
          "JameClaw could not process that message.",
      })
      break

    case "pong":
      break

    default:
      console.log("Unknown jame message type:", message.type)
  }
}

function normalizeMediaKind(
  value: unknown,
  contentType: string,
): ChatMediaAttachment["kind"] {
  if (
    value === "image" ||
    value === "audio" ||
    value === "video" ||
    value === "file"
  ) {
    return value
  }
  if (contentType.startsWith("image/")) return "image"
  if (contentType.startsWith("audio/")) return "audio"
  if (contentType.startsWith("video/")) return "video"
  return "file"
}

import { atom, getDefaultStore } from "jotai"

import {
  getInitialActiveSessionId,
  writeStoredSessionId,
} from "@/features/chat/state"

export interface ChatMessage {
  id: string
  role: "user" | "assistant"
  content: string
  timestamp: number | string
  media?: ChatMediaAttachment[]
  activity?: ChatActivity[]
}

// These are deliberately high-level, user-facing status updates. They describe
// what the agent is doing without exposing private model reasoning.
export interface ChatActivity {
  id: string
  label: string
  kind: "context" | "plan" | "tool" | "verify"
  timestamp: number
}

export interface ChatMediaAttachment {
  url: string
  filename: string
  contentType: string
  kind: "image" | "audio" | "video" | "file"
}

export type ConnectionState =
  | "disconnected"
  | "connecting"
  | "connected"
  | "reconnecting"
  | "offline"
  | "error"

export interface ChatStoreState {
  messages: ChatMessage[]
  connectionState: ConnectionState
  errorMessage: string | null
  isTyping: boolean
  activeSessionId: string
  hasHydratedActiveSession: boolean
}

type ChatStorePatch = Partial<ChatStoreState>

const DEFAULT_CHAT_STATE: ChatStoreState = {
  messages: [],
  connectionState: "disconnected",
  errorMessage: null,
  isTyping: false,
  activeSessionId: getInitialActiveSessionId(),
  hasHydratedActiveSession: false,
}

export const chatAtom = atom<ChatStoreState>(DEFAULT_CHAT_STATE)

const store = getDefaultStore()

export function getChatState() {
  return store.get(chatAtom)
}

export function updateChatStore(
  patch:
    | ChatStorePatch
    | ((prev: ChatStoreState) => ChatStorePatch | ChatStoreState),
) {
  store.set(chatAtom, (prev) => {
    const nextPatch = typeof patch === "function" ? patch(prev) : patch
    const next = { ...prev, ...nextPatch }

    if (next.activeSessionId !== prev.activeSessionId) {
      writeStoredSessionId(next.activeSessionId)
    }

    return next
  })
}

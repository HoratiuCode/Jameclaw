import type { ChatActivity, ChatMessage } from "@/store/chat"

const STATUS_STEPS: Record<string, Omit<ChatActivity, "id" | "timestamp">> = {
  "Thinking... 💭": { label: "Starting on your request", kind: "plan" },
  "Reading your request...": { label: "Reading your request", kind: "context" },
  "Loading conversation context...": {
    label: "Loading conversation context",
    kind: "context",
  },
  "Preparing the next action...": {
    label: "Planning the next action",
    kind: "plan",
  },
  "Verifying the result...": { label: "Verifying the result", kind: "verify" },
  "Working on it...": { label: "Working on your request", kind: "plan" },
}

const TOOL_STATUS = /^Running (.+)\.\.\.$/
const TOOL_MARKER = /`(?:🔧|💻|🔍|📝|📖|📁)\s+([^`]+)`/g

export function isActivityOnlyContent(content: string) {
  const trimmed = content.trim()
  return Boolean(STATUS_STEPS[trimmed] || TOOL_STATUS.test(trimmed))
}

export function addActivityFromContent(
  message: ChatMessage,
  content: string,
  timestamp = Date.now(),
): ChatActivity[] | undefined {
  const activities = [...(message.activity ?? [])]
  const labels = new Set(activities.map((item) => item.label))
  const add = (label: string, kind: ChatActivity["kind"]) => {
    if (!labels.has(label)) {
      activities.push({
        id: `${message.id}-${activities.length}-${timestamp}`,
        label,
        kind,
        timestamp,
      })
      labels.add(label)
    }
  }

  const trimmed = content.trim()
  const step = STATUS_STEPS[trimmed]
  if (step) add(step.label, step.kind)

  const toolStatus = trimmed.match(TOOL_STATUS)
  if (toolStatus) add(`Running ${toolStatus[1]}`, "tool")

  for (const match of content.matchAll(TOOL_MARKER)) {
    add(match[1].trim(), "tool")
  }

  return activities.length > 0 ? activities : undefined
}

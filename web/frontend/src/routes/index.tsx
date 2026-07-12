import { createFileRoute } from "@tanstack/react-router"

import { ChatPage } from "@/components/chat/chat-page"

export const Route = createFileRoute("/")({
  validateSearch: (search: Record<string, unknown>) => ({
    prompt: typeof search.prompt === "string" ? search.prompt : undefined,
  }),
  component: ChatPage,
})

import { createFileRoute } from "@tanstack/react-router"

import { AgentMemoryPage } from "@/components/agents/agent-memory-page"

export const Route = createFileRoute("/agent-memory/$id")({
  component: () => {
    const { id } = Route.useParams()
    return <AgentMemoryPage agentID={id} />
  },
})

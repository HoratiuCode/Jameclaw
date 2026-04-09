import { createFileRoute } from "@tanstack/react-router"

import { LearnedPage } from "@/components/skills/learned-page"

export const Route = createFileRoute("/agent/learned")({
  component: AgentLearnedRoute,
})

function AgentLearnedRoute() {
  return <LearnedPage />
}

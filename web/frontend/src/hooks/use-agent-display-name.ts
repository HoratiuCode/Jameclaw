import { useQuery } from "@tanstack/react-query"

import { getAgents } from "@/api/agents"

export function useAgentDisplayName() {
  const { data } = useQuery({
    queryKey: ["agents-admin"],
    queryFn: getAgents,
    staleTime: 30_000,
  })

  const mainAgent =
    data?.agents.find((agent) => agent.id === "main") ?? data?.agents[0]
  return (
    mainAgent?.human.agent_name.trim() ||
    (mainAgent?.name !== "Main" ? mainAgent?.name.trim() : "") ||
    "JameClaw"
  )
}

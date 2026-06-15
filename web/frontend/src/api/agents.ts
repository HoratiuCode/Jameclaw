export interface AgentSummary {
  id: string
  name: string
  default: boolean
  workspace: string
  model: string
  model_fallbacks: string[] | null
  skills: string[] | null
  subagents: string[] | null
  session_count: number
  message_count: number
  tool_calls: number
}

export interface AgentsResponse {
  agents: AgentSummary[]
  default_model: string
  enabled_channels: number
  configured_models: number
}

export async function getAgents(): Promise<AgentsResponse> {
  const res = await fetch("/api/agents")
  if (!res.ok) {
    throw new Error(`Failed to fetch agents: ${res.status}`)
  }
  const data = (await res.json()) as AgentsResponse
  return {
    ...data,
    agents: (data.agents ?? []).map((agent) => ({
      ...agent,
      model_fallbacks: agent.model_fallbacks ?? [],
      skills: agent.skills ?? [],
      subagents: agent.subagents ?? [],
      session_count: agent.session_count ?? 0,
      message_count: agent.message_count ?? 0,
      tool_calls: agent.tool_calls ?? 0,
    })),
    enabled_channels: data.enabled_channels ?? 0,
    configured_models: data.configured_models ?? 0,
  }
}

export async function updateAgent(
  id: string,
  body: {
    default?: boolean
    model?: string
    model_fallbacks?: string[]
  },
): Promise<{ status: string }> {
  const res = await fetch(`/api/agents/${encodeURIComponent(id)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    throw new Error(`Failed to update agent: ${res.status}`)
  }
  return res.json() as Promise<{ status: string }>
}

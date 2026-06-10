export interface AgentSummary {
  id: string
  name: string
  default: boolean
  workspace: string
  model: string
  model_fallbacks: string[]
  skills: string[]
  subagents: string[]
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
  return res.json() as Promise<AgentsResponse>
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

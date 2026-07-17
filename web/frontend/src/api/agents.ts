export interface AgentSummary {
  id: string
  name: string
  default: boolean
  workspace: string
  model: string
  model_fallbacks: string[] | null
  skills: string[] | null
  subagents: string[] | null
  human: AgentHuman
  session_count: number
  message_count: number
  tool_calls: number
}

export interface AgentHuman {
  agent_name: string
  persona: string
  tone: string
  discussion_mode: string
  memory_notes: string
  status_style: string
}

export interface AgentsResponse {
  agents: AgentSummary[]
  default_model: string
  enabled_channels: number
  configured_models: number
}

export interface AgentDailyNote {
  date: string
  path: string
  content: string
}

export interface AgentMemory {
  agent_id: string
  workspace: string
  memory_path: string
  long_term: string
  daily_notes: AgentDailyNote[]
  human_notes?: string
  files_checked: Record<string, string>
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
      human: normalizeHuman(agent.human),
      session_count: agent.session_count ?? 0,
      message_count: agent.message_count ?? 0,
      tool_calls: agent.tool_calls ?? 0,
    })),
    enabled_channels: data.enabled_channels ?? 0,
    configured_models: data.configured_models ?? 0,
  }
}

export async function getAgentMemory(id: string): Promise<AgentMemory> {
  const res = await fetch(`/api/agents/${encodeURIComponent(id)}/memory`)
  if (!res.ok) {
    throw new Error(`Failed to fetch agent memory: ${res.status}`)
  }
  const data = (await res.json()) as AgentMemory
  return {
    ...data,
    daily_notes: data.daily_notes ?? [],
    files_checked: data.files_checked ?? {},
  }
}

export async function updateAgentMemory(id: string, longTerm: string): Promise<AgentMemory> {
  const res = await fetch(`/api/agents/${encodeURIComponent(id)}/memory`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ long_term: longTerm }),
  })
  if (!res.ok) {
    const message = await res.text()
    throw new Error(message.trim() || `Failed to save agent memory: ${res.status}`)
  }
  return getAgentMemory(id)
}

export async function updateAgent(
  id: string,
  body: {
    default?: boolean
    model?: string
    model_fallbacks?: string[]
    human?: Partial<AgentHuman>
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

export async function createAgent(body: {
  id: string
  name?: string
  model?: string
  workspace?: string
  parent_id?: string
  human?: Partial<AgentHuman>
}): Promise<{ status: string; id: string }> {
  const res = await fetch("/api/agents", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const message = await res.text()
    throw new Error(message.trim() || `Failed to create agent: ${res.status}`)
  }
  return res.json() as Promise<{ status: string; id: string }>
}

function normalizeHuman(value?: Partial<AgentHuman> | null): AgentHuman {
  return {
    agent_name: value?.agent_name ?? "",
    persona: value?.persona ?? "",
    tone: value?.tone ?? "",
    discussion_mode: value?.discussion_mode ?? "",
    memory_notes: value?.memory_notes ?? "",
    status_style: value?.status_style ?? "",
  }
}

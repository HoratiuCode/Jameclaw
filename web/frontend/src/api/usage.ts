export interface UsageRoleCounts {
  user: number
  assistant: number
  tool: number
  system: number
  other: number
}

export interface UsageTotals {
  sessions: number
  messages: number
  user_messages: number
  assistant_messages: number
  tool_calls: number
  estimated_chars: number
  estimated_tokens: number
  role_counts: UsageRoleCounts
}

export interface UsageDailyBucket {
  date: string
  sessions: number
  messages: number
  tool_calls: number
  estimated_chars: number
  estimated_tokens: number
}

export interface UsageSessionItem {
  id: string
  key: string
  title: string
  preview: string
  created: string
  updated: string
  message_count: number
  tool_calls: number
  estimated_chars: number
  estimated_tokens: number
  roles: UsageRoleCounts
}

export interface UsageLogItem {
  session_id: string
  role: string
  content: string
  tool_calls: number
  updated: string
}

export interface UsageResponse {
  totals: UsageTotals
  daily: UsageDailyBucket[]
  sessions: UsageSessionItem[]
  logs: UsageLogItem[]
}

export async function getUsage(params: {
  start?: string
  end?: string
  q?: string
} = {}): Promise<UsageResponse> {
  const search = new URLSearchParams()
  if (params.start) search.set("start", params.start)
  if (params.end) search.set("end", params.end)
  if (params.q) search.set("q", params.q)
  const suffix = search.toString() ? `?${search.toString()}` : ""
  const res = await fetch(`/api/usage${suffix}`)
  if (!res.ok) {
    throw new Error(`Failed to fetch usage: ${res.status}`)
  }
  return res.json() as Promise<UsageResponse>
}

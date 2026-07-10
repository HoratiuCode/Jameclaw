export interface AutomationItem {
  id: string
  name: string
  enabled: boolean
  status: "scheduled" | "waiting" | "disabled" | "error" | string
  schedule: string
  prompt: string
  delivery: string
  next_run_at_ms?: number
  last_run_at_ms?: number
  last_status?: string
  last_error?: string
  created_at_ms: number
  updated_at_ms: number
  delete_after_run: boolean
}

interface AutomationResponse {
  items: AutomationItem[]
}

export async function getAutomations(): Promise<AutomationItem[]> {
  const res = await fetch("/api/automation")
  if (!res.ok) {
    throw new Error(`Failed to fetch automations: ${res.status}`)
  }
  const data = (await res.json()) as AutomationResponse
  return data.items ?? []
}

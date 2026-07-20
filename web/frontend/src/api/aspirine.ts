export interface AspirineIssue {
  id: string
  title: string
  severity: "critical" | "warning" | "info"
  status: string
  description: string
  suggestion: string
  recovery_prompt?: string
  auto_fix_action?: string
  auto_fix_label?: string
  affected?: string[]
  last_observed_at: string
}

export interface AspirineSummary {
  status: "critical" | "warning" | "healthy"
  issue_count: number
  critical_count: number
  warning_count: number
  checked_at: string
  issues: AspirineIssue[]
}

export interface AspirineActionResponse {
  status: string
  action?: string
  pid?: number
  message?: string
}

export async function getAspirineSummary(): Promise<AspirineSummary> {
  const res = await fetch("/api/aspirine")
  if (!res.ok) {
    throw new Error(`Failed to fetch Aspirine status: ${res.status}`)
  }
  return res.json() as Promise<AspirineSummary>
}

export async function runAspirineAction(
  action: string,
): Promise<AspirineActionResponse> {
  const res = await fetch(`/api/aspirine/actions/${action}`, {
    method: "POST",
  })
  const data = (await res.json().catch(() => ({}))) as AspirineActionResponse
  if (!res.ok) {
    throw new Error(data.message || `Aspirine action failed: ${res.status}`)
  }
  return data
}

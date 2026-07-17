export interface AutomationItem {
  id: string
  name: string
  enabled: boolean
  status: "scheduled" | "waiting" | "disabled" | "error" | string
  schedule: string
  prompt: string
  delivery: string
  delivery_approved: boolean
  next_run_at_ms?: number
  last_run_at_ms?: number
  last_status?: string
  last_error?: string
  running: boolean
  created_at_ms: number
  updated_at_ms: number
  delete_after_run: boolean
  timezone?: string
  retry_attempts?: number
  retry_delay_seconds?: number
  quiet_hours_start?: string
  quiet_hours_end?: string
  max_runs_per_day?: number
  runs_today?: number
}

export interface AutomationBlueprintField {
  name: string
  type: "time" | "enum" | "text" | "weekdays" | string
  label: string
  default?: string
  options?: string[]
  optional: boolean
  strict: boolean
  help?: string
}

export interface AutomationBlueprint {
  key: string
  title: string
  description: string
  category: string
  fields: AutomationBlueprintField[]
  tags: string[]
}

interface AutomationResponse {
  items: AutomationItem[]
}

interface AutomationBlueprintResponse {
  blueprints: AutomationBlueprint[]
}

interface AutomationBlueprintInstantiateResponse {
  item: AutomationItem
}

export interface AutomationOutput {
  automation_id: string
  status?: string
  ran_at_ms?: number
  content: string
}

export async function getAutomations(): Promise<AutomationItem[]> {
  const res = await fetch("/api/automation")
  if (!res.ok) {
    throw new Error(`Failed to fetch automations: ${res.status}`)
  }
  const data = (await res.json()) as AutomationResponse
  return data.items ?? []
}

export async function getAutomationBlueprints(): Promise<AutomationBlueprint[]> {
  const res = await fetch("/api/automation/blueprints")
  if (!res.ok) {
    throw new Error(`Failed to fetch automation blueprints: ${res.status}`)
  }
  const data = (await res.json()) as AutomationBlueprintResponse
  return data.blueprints ?? []
}

export async function runAutomation(id: string): Promise<void> {
  const res = await fetch(`/api/automation/${encodeURIComponent(id)}/run`, {
    method: "POST",
  })
  if (!res.ok) {
    const message = await res.text()
    throw new Error(message.trim() || `Failed to run automation: ${res.status}`)
  }
}

export async function getAutomationOutput(id: string): Promise<AutomationOutput> {
  const res = await fetch(`/api/automation/${encodeURIComponent(id)}/output`)
  if (!res.ok) {
    const message = await res.text()
    throw new Error(message.trim() || "No output has been recorded yet")
  }
  return res.json() as Promise<AutomationOutput>
}

export async function instantiateAutomationBlueprint(
  blueprint: string,
  values: Record<string, string>,
): Promise<AutomationItem> {
  const res = await fetch("/api/automation/blueprints/instantiate", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ blueprint, values }),
  })
  if (!res.ok) {
    const message = await res.text()
    throw new Error(message.trim() || `Failed to create automation: ${res.status}`)
  }
  const data = (await res.json()) as AutomationBlueprintInstantiateResponse
  return data.item
}

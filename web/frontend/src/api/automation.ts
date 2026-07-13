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
  created_at_ms: number
  updated_at_ms: number
  delete_after_run: boolean
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

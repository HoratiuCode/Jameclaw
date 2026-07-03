export type OnboardingStepStatus =
  | "ready"
  | "attention"
  | "blocked"
  | "optional"

export interface OnboardingStep {
  id: string
  title: string
  description: string
  status: OnboardingStepStatus
  detail?: string
  action?: string
  action_href?: string
}

export interface OnboardingStatus {
  complete: boolean
  should_show: boolean
  completed_at?: string
  config_path: string
  workspace?: string
  version: string
  steps: OnboardingStep[]
  next_step_id?: string
  ready_count: number
  total_count: number
  ready_for_chat: boolean
}

interface OnboardingActionResponse {
  status: string
  completed_at?: string
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, options)
  if (!res.ok) {
    const message = await res.text()
    throw new Error(message || `API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getOnboardingStatus(): Promise<OnboardingStatus> {
  return request<OnboardingStatus>("/api/onboarding/status")
}

export async function completeOnboarding(): Promise<OnboardingActionResponse> {
  return request<OnboardingActionResponse>("/api/onboarding/complete", {
    method: "POST",
  })
}

export async function resetOnboarding(): Promise<OnboardingActionResponse> {
  return request<OnboardingActionResponse>("/api/onboarding/reset", {
    method: "POST",
  })
}

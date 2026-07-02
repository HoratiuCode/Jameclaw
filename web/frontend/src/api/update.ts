export interface UpdateStatusResponse {
  current_version: string
  latest_version?: string
  latest_name?: string
  release_url?: string
  published_at?: string
  update_available: boolean
  dismissed: boolean
  check_error?: string
  update_action: string
  update_action_text: string
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, options)
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getUpdateStatus(): Promise<UpdateStatusResponse> {
  return request<UpdateStatusResponse>("/api/update/status")
}

export async function dismissUpdate(version: string): Promise<void> {
  await request("/api/update/dismiss", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ version }),
  })
}

export async function openUpdatePage(): Promise<{ release_url?: string }> {
  return request<{ release_url?: string }>("/api/update/open", {
    method: "POST",
  })
}

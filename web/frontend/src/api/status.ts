export interface AppStatusResponse {
  status: string
  version: string
  uptime: string
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, options)
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getAppStatus(): Promise<AppStatusResponse> {
  return request<AppStatusResponse>("/api/status")
}

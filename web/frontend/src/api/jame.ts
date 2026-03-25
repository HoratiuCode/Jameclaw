// API client for Jame Channel configuration.

interface JameTokenResponse {
  token: string
  ws_url: string
  enabled: boolean
}

interface JameSetupResponse {
  token: string
  ws_url: string
  enabled: boolean
  changed: boolean
}

const BASE_URL = ""

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, options)
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getJameToken(): Promise<JameTokenResponse> {
  return request<JameTokenResponse>("/api/jame/token")
}

export async function regenJameToken(): Promise<JameTokenResponse> {
  return request<JameTokenResponse>("/api/jame/token", { method: "POST" })
}

export async function setupJame(): Promise<JameSetupResponse> {
  return request<JameSetupResponse>("/api/jame/setup", { method: "POST" })
}

export type { JameTokenResponse, JameSetupResponse }

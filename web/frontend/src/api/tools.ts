export interface ToolSupportItem {
  name: string
  description: string
  category: string
  config_key: string
  status: "enabled" | "disabled" | "blocked"
  reason_code?: string
}

interface ToolsResponse {
  tools: ToolSupportItem[]
}

interface ToolActionResponse {
	status: string
}

export interface MCPServer {
	name: string
	enabled: boolean
	transport: "stdio" | "http" | "sse"
	command?: string
	args?: string[]
	url?: string
}

interface MCPServersResponse {
	enabled: boolean
	servers: MCPServer[]
}

export interface SaveMCPServerPayload {
	name: string
	enabled: boolean
	transport: "stdio" | "http" | "sse"
	command?: string
	args?: string[]
	url?: string
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, options)
  if (!res.ok) {
    let message = `API error: ${res.status} ${res.statusText}`
    try {
      const body = (await res.json()) as {
        error?: string
        errors?: string[]
      }
      if (Array.isArray(body.errors) && body.errors.length > 0) {
        message = body.errors.join("; ")
      } else if (typeof body.error === "string" && body.error.trim() !== "") {
        message = body.error
      }
    } catch {
      // ignore invalid body
    }
    throw new Error(message)
  }
  return res.json() as Promise<T>
}

export async function getTools(): Promise<ToolsResponse> {
  return request<ToolsResponse>("/api/tools")
}

export async function setToolEnabled(
  name: string,
  enabled: boolean,
): Promise<ToolActionResponse> {
  return request<ToolActionResponse>(
    `/api/tools/${encodeURIComponent(name)}/state`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled }),
    },
  )
}

export async function getMCPServers(): Promise<MCPServersResponse> {
	return request<MCPServersResponse>("/api/tools/mcp/servers")
}

export async function saveMCPServer(
	payload: SaveMCPServerPayload,
): Promise<{ status: string; gateway_restarted: boolean }> {
	return request("/api/tools/mcp/servers", {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(payload),
	})
}

export type ArtifactKind = "app" | "code"

export interface Artifact {
  id: string
  name: string
  kind: ArtifactKind
  html: string
  css: string
  javascript: string
  created_at_ms: number
  updated_at_ms: number
}

export interface ArtifactFile {
  name: string
  language: "html" | "css" | "javascript" | string
  content: string
  size: number
}

type ArtifactDraft = Omit<Artifact, "id" | "created_at_ms" | "updated_at_ms">

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(path, options)
  if (!response.ok) {
    const message = await response.text()
    throw new Error(message.trim() || `Artifact request failed: ${response.status}`)
  }
  return response.json() as Promise<T>
}

export async function getArtifacts(): Promise<Artifact[]> {
  const data = await request<{ items: Artifact[] }>("/api/artifacts")
  return data.items ?? []
}

export function getArtifactFiles(id: string): Promise<{ folder: string; files: ArtifactFile[] }> {
  return request(`/api/artifacts/${encodeURIComponent(id)}/files`)
}

export function createArtifact(draft: ArtifactDraft): Promise<Artifact> {
  return request("/api/artifacts", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(draft),
  })
}

export function updateArtifact(id: string, draft: ArtifactDraft): Promise<Artifact> {
  return request(`/api/artifacts/${encodeURIComponent(id)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(draft),
  })
}

export async function deleteArtifact(id: string): Promise<void> {
  const response = await fetch(`/api/artifacts/${encodeURIComponent(id)}`, { method: "DELETE" })
  if (!response.ok) {
    throw new Error((await response.text()).trim() || "Failed to delete artifact")
  }
}

export function artifactPreviewURL(id: string): string {
  return `/api/artifacts/${encodeURIComponent(id)}/preview`
}

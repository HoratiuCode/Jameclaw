export interface LocalFileSearchItem {
  name: string
  path: string
  directory: string
  kind: "file" | "folder" | string
  size: number
  modified_at_ms: number
}

interface LocalFileSearchResponse {
  items: LocalFileSearchItem[]
}

export async function searchLocalFiles(
  query: string,
  limit = 12,
): Promise<LocalFileSearchItem[]> {
  const params = new URLSearchParams({
    q: query,
    limit: String(limit),
  })
  const res = await fetch(`/api/files/search?${params.toString()}`)
  if (!res.ok) {
    throw new Error(`Failed to search local files: ${res.status}`)
  }
  const data = (await res.json()) as LocalFileSearchResponse
  return data.items ?? []
}

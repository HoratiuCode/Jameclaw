export interface DashboardCard {
  id: string
  title: string
  status: string
  description?: string
  count?: number
}

export interface DashboardResponse {
  items: DashboardCard[]
}

export async function getDashboardCards(kind: string): Promise<DashboardCard[]> {
  const res = await fetch(`/api/${kind}`)
  if (!res.ok) {
    throw new Error(`Failed to fetch ${kind}: ${res.status}`)
  }
  const data = (await res.json()) as DashboardResponse
  return data.items ?? []
}

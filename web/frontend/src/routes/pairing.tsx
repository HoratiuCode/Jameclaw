import { createFileRoute } from "@tanstack/react-router"

import { DashboardPage } from "@/components/dashboard/dashboard-page"

export const Route = createFileRoute("/pairing")({
  component: () => (
    <DashboardPage
      title="Pairing"
      kind="pairing"
      empty="No enabled messaging channels are available for pairing."
    />
  ),
})

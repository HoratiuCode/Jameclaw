import { createFileRoute } from "@tanstack/react-router"

import { DashboardPage } from "@/components/dashboard/dashboard-page"

export const Route = createFileRoute("/plugins")({
  component: () => (
    <DashboardPage
      title="Plugins"
      kind="plugins"
      empty="No plugin surfaces are available."
    />
  ),
})

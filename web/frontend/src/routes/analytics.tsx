import { createFileRoute } from "@tanstack/react-router"

import { DashboardPage } from "@/components/dashboard/dashboard-page"

export const Route = createFileRoute("/analytics")({
  component: () => (
    <DashboardPage
      title="Analytics"
      kind="analytics"
      empty="No analytics data has been recorded yet."
    />
  ),
})

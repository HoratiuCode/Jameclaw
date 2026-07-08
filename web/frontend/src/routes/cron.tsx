import { createFileRoute } from "@tanstack/react-router"

import { DashboardPage } from "@/components/dashboard/dashboard-page"

export const Route = createFileRoute("/cron")({
  component: () => (
    <DashboardPage
      title="Cron"
      kind="cron"
      empty="No scheduled jobs are configured."
    />
  ),
})

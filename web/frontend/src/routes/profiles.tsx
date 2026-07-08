import { createFileRoute } from "@tanstack/react-router"

import { DashboardPage } from "@/components/dashboard/dashboard-page"

export const Route = createFileRoute("/profiles")({
  component: () => (
    <DashboardPage
      title="Profiles"
      kind="profiles"
      empty="No additional agent profiles are configured."
    />
  ),
})

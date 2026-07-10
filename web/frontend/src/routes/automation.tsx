import { createFileRoute } from "@tanstack/react-router"

import { AutomationPage } from "@/components/automation/automation-page"

export const Route = createFileRoute("/automation")({
  component: AutomationPage,
})

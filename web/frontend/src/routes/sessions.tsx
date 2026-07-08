import { createFileRoute } from "@tanstack/react-router"

import { SessionsPage } from "@/components/dashboard/sessions-page"

export const Route = createFileRoute("/sessions")({
  component: SessionsPage,
})

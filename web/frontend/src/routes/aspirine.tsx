import { createFileRoute } from "@tanstack/react-router"

import { AspirinePage } from "@/components/aspirine/aspirine-page"

export const Route = createFileRoute("/aspirine")({
  component: AspirinePage,
})

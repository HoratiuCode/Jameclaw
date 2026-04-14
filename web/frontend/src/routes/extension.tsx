import { createFileRoute } from "@tanstack/react-router"

import { ExtensionPage } from "@/components/chat/extension-page"

export const Route = createFileRoute("/extension")({
  component: ExtensionPage,
})

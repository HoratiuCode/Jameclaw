import { Outlet, createRootRoute, useRouterState } from "@tanstack/react-router"
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools"
import { useEffect } from "react"

import { AppLayout } from "@/components/app-layout"
import { initializeChatStore } from "@/features/chat/controller"

const RootLayout = () => {
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const isStandaloneLanding = pathname === "/landing"

  useEffect(() => {
    if (!isStandaloneLanding) {
      initializeChatStore()
    }
  }, [isStandaloneLanding])

  const content = (
    <>
      <Outlet />
      <TanStackRouterDevtools />
    </>
  )

  if (isStandaloneLanding) {
    return content
  }

  return (
    <AppLayout>
      {content}
    </AppLayout>
  )
}

export const Route = createRootRoute({ component: RootLayout })

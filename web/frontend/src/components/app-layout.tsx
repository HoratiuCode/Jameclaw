import type { ReactNode } from "react"
import { Toaster } from "sonner"

import { AppHeader } from "@/components/app-header"
import { AppSidebar } from "@/components/app-sidebar"
import { AppStartupSplash } from "@/components/app-startup-splash"
import { OnboardingWizard } from "@/components/onboarding-wizard"
import { SidebarProvider } from "@/components/ui/sidebar"
import { TooltipProvider } from "@/components/ui/tooltip"
import { UpdateBanner } from "@/components/update-banner"

export function AppLayout({ children }: { children: ReactNode }) {
  return (
    <TooltipProvider>
      <SidebarProvider className="flex h-dvh flex-col overflow-hidden">
        <AppStartupSplash />
        <OnboardingWizard />
        <UpdateBanner />
        <AppHeader />

        <div className="flex flex-1 overflow-hidden">
          <AppSidebar />
          <div className="flex w-full flex-col overflow-hidden">
            <main className="flex min-h-0 w-full max-w-full flex-1 flex-col overflow-hidden">
              {children}
            </main>
          </div>
        </div>
        <Toaster position="bottom-center" />
      </SidebarProvider>
    </TooltipProvider>
  )
}

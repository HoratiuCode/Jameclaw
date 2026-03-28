import {
  IconDeviceDesktop,
  IconFolder,
  IconUser,
} from "@tabler/icons-react"
import { useEffect, useState } from "react"

import { getHostInfo, type HostInfo } from "@/api/system"

const shellLaunchSplashCookie = "jameclaw_shell_launch"

function consumeShellLaunchCookie(): boolean {
  const hasCookie = document.cookie
    .split("; ")
    .some((part) => part.startsWith(`${shellLaunchSplashCookie}=`))

  if (hasCookie) {
    document.cookie = `${shellLaunchSplashCookie}=; Path=/; Max-Age=0; SameSite=Lax`
  }

  return hasCookie
}

export function AppStartupSplash() {
  const [visible, setVisible] = useState(false)
  const [hostInfo, setHostInfo] = useState<HostInfo | null>(null)

  useEffect(() => {
    if (!consumeShellLaunchCookie()) {
      return
    }

    let active = true
    setVisible(true)

    void getHostInfo()
      .then((data) => {
        if (active) {
          setHostInfo(data)
        }
      })
      .catch(() => {
        if (active) {
          setHostInfo(null)
        }
      })

    const timeout = window.setTimeout(() => {
      if (active) {
        setVisible(false)
      }
    }, 1000)

    return () => {
      active = false
      window.clearTimeout(timeout)
    }
  }, [])

  if (!visible || !hostInfo) {
    return null
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-[radial-gradient(circle_at_top,rgba(239,68,68,0.22),transparent_42%),rgba(10,10,10,0.74)] p-6 backdrop-blur-md">
      <div className="from-red-50/90 via-background to-red-100/50 w-full max-w-5xl rounded-[2rem] border border-red-200/70 bg-gradient-to-br px-8 py-10 shadow-[0_28px_90px_-30px_rgba(127,29,29,0.65)]">
        <div className="mb-6 text-center text-xs font-semibold tracking-[0.36em] uppercase text-red-700/80">
          This Web Console
        </div>
        <div className="grid gap-4 text-center md:grid-cols-3">
          <div className="rounded-2xl border border-red-200/70 bg-white/70 px-5 py-6">
            <IconDeviceDesktop className="mx-auto mb-3 h-6 w-6 text-red-500" />
            <div className="mb-1 text-[11px] font-semibold tracking-[0.22em] uppercase text-red-700/75">
              Device
            </div>
            <span className="block truncate text-lg font-semibold">
              {hostInfo.hostname}
            </span>
          </div>
          <div className="rounded-2xl border border-red-200/70 bg-white/70 px-5 py-6">
            <IconUser className="mx-auto mb-3 h-6 w-6 text-red-500" />
            <div className="mb-1 text-[11px] font-semibold tracking-[0.22em] uppercase text-red-700/75">
              User
            </div>
            <span className="block truncate text-lg">{hostInfo.username}</span>
          </div>
          <div className="rounded-2xl border border-red-200/70 bg-white/70 px-5 py-6">
            <IconFolder className="mx-auto mb-3 h-6 w-6 text-red-500" />
            <div className="mb-1 text-[11px] font-semibold tracking-[0.22em] uppercase text-red-700/75">
              Folder
            </div>
            <span className="block truncate text-lg">
              {hostInfo.documents_path}
            </span>
          </div>
        </div>
      </div>
    </div>
  )
}

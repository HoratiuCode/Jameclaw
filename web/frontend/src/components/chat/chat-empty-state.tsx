import {
  IconArrowRight,
  IconDeviceDesktop,
  IconFolder,
  IconMessageCircle2,
  IconPlayerPlay,
  IconPlugConnectedX,
  IconRobot,
  IconRobotOff,
  IconSparkles,
  IconStar,
  IconUser,
} from "@tabler/icons-react"
import { Link } from "@tanstack/react-router"
import { type ReactNode, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import { getHostInfo, type HostInfo } from "@/api/system"
import { Button } from "@/components/ui/button"

interface ChatEmptyStateProps {
  hasConfiguredModels: boolean
  defaultModelName: string
  isConnected: boolean
  onPromptSelect: (prompt: string) => void
}

const starterPrompts = [
  "Summarize my workspace and tell me where to start.",
  "Create a plan for today's most important tasks.",
  "Review my config and explain what I should improve.",
]

const shellLaunchSplashCookie = "jameclaw_shell_launch"

function StepCard({
  step,
  title,
  description,
  actionLabel,
  actionIcon,
}: {
  step: string
  title: string
  description: string
  actionLabel: string
  actionIcon: ReactNode
}) {
  return (
    <div className="bg-background/80 rounded-2xl border p-4 shadow-sm backdrop-blur">
      <div className="text-muted-foreground mb-2 text-xs font-semibold tracking-[0.24em] uppercase">
        {step}
      </div>
      <div className="mb-2 text-base font-semibold">{title}</div>
      <p className="text-muted-foreground text-sm leading-6">{description}</p>
      <div className="mt-4 flex items-center gap-2 text-sm font-medium">
        {actionIcon}
        <span>{actionLabel}</span>
      </div>
    </div>
  )
}

function HostSummaryCard({
  hostInfo,
  visible,
}: {
  hostInfo: HostInfo | null
  visible: boolean
}) {
  if (!hostInfo || !visible) {
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

export function ChatEmptyState({
  hasConfiguredModels,
  defaultModelName,
  isConnected,
  onPromptSelect,
}: ChatEmptyStateProps) {
  const { t } = useTranslation()
  const [hostInfo, setHostInfo] = useState<HostInfo | null>(null)
  const [showHostSplash, setShowHostSplash] = useState(false)

  useEffect(() => {
    const shouldShowSplash = document.cookie
      .split("; ")
      .some((part) => part.startsWith(`${shellLaunchSplashCookie}=`))

    if (!shouldShowSplash) {
      return
    }

    document.cookie = `${shellLaunchSplashCookie}=; Path=/; Max-Age=0; SameSite=Lax`

    let active = true

    void getHostInfo()
      .then((data) => {
        if (active) {
          setHostInfo(data)
          setShowHostSplash(true)
        }
      })
      .catch(() => {
        if (active) {
          setHostInfo(null)
        }
      })

    return () => {
      active = false
    }
  }, [])

  useEffect(() => {
    if (!showHostSplash) {
      return
    }

    const timeout = window.setTimeout(() => {
      setShowHostSplash(false)
    }, 1000)

    return () => {
      window.clearTimeout(timeout)
    }
  }, [showHostSplash])

  if (!hasConfiguredModels) {
    return (
      <div>
        <HostSummaryCard hostInfo={hostInfo} visible={showHostSplash} />
        <div className="relative overflow-hidden rounded-[2rem] border bg-gradient-to-br from-red-50 via-background to-background p-8 shadow-sm">
          <div className="absolute top-0 right-0 h-40 w-40 rounded-full bg-red-400/10 blur-3xl" />
          <div className="relative grid gap-6 lg:grid-cols-[minmax(0,1.1fr)_minmax(18rem,0.9fr)]">
            <div className="flex flex-col justify-center">
              <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-red-500/12 text-red-600">
                <IconRobotOff className="h-7 w-7" />
              </div>
              <div className="mb-3 text-xs font-semibold tracking-[0.28em] uppercase text-red-700">
                Step 1 of 3
              </div>
              <h3 className="mb-3 text-3xl font-semibold tracking-tight">
                {t("chat.empty.noConfiguredModel")}
              </h3>
              <p className="text-muted-foreground max-w-xl text-sm leading-7">
                {t("chat.empty.noConfiguredModelDescription")}
              </p>
              <div className="mt-6 flex flex-wrap gap-3">
                <Button asChild size="sm" className="gap-2 px-4">
                  <Link to="/models">
                    {t("chat.empty.goToModels")}
                    <IconArrowRight className="h-4 w-4" />
                  </Link>
                </Button>
              </div>
            </div>

            <div className="grid gap-3">
              <StepCard
                step="Next"
                title="Add your first model"
                description="Connect OpenAI, Anthropic, OpenRouter, Ollama, or any compatible endpoint so JameClaw has something to run."
                actionLabel="Open Models"
                actionIcon={<IconSparkles className="h-4 w-4 text-red-600" />}
              />
              <StepCard
                step="After that"
                title="Pick a default"
                description="Set one model as the default so every new chat can start immediately without extra setup."
                actionLabel="Choose a default model"
                actionIcon={<IconStar className="h-4 w-4 text-red-600" />}
              />
              <StepCard
                step="Then"
                title="Start chatting"
                description="Launch the gateway from the top bar and the chat will be ready without any more onboarding friction."
                actionLabel="Start the gateway when you're ready"
                actionIcon={<IconPlayerPlay className="h-4 w-4 text-red-600" />}
              />
            </div>
          </div>
        </div>
      </div>
    )
  }

  if (!defaultModelName) {
    return (
      <div>
        <HostSummaryCard hostInfo={hostInfo} visible={showHostSplash} />
        <div className="relative overflow-hidden rounded-[2rem] border bg-gradient-to-br from-red-50 via-background to-lime-50 p-8 shadow-sm">
          <div className="absolute bottom-0 left-0 h-32 w-32 rounded-full bg-red-400/10 blur-3xl" />
          <div className="relative grid gap-6 lg:grid-cols-[minmax(0,1.05fr)_minmax(18rem,0.95fr)]">
            <div className="flex flex-col justify-center">
              <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-red-500/12 text-red-600">
                <IconStar className="h-7 w-7" />
              </div>
              <div className="mb-3 text-xs font-semibold tracking-[0.28em] uppercase text-red-700">
                Step 2 of 3
              </div>
              <h3 className="mb-3 text-3xl font-semibold tracking-tight">
                {t("chat.empty.noSelectedModel")}
              </h3>
              <p className="text-muted-foreground max-w-xl text-sm leading-7">
                {t("chat.empty.noSelectedModelDescription")}
              </p>
              <div className="mt-6">
                <Button asChild size="sm" className="gap-2 px-4">
                  <Link to="/models">
                    Select a Default Model
                    <IconArrowRight className="h-4 w-4" />
                  </Link>
                </Button>
              </div>
            </div>

            <div className="grid gap-3">
              <StepCard
                step="Current focus"
                title="Choose the model you trust most"
                description="The default model is used for fresh chats, quick actions, and a smoother first-run experience."
                actionLabel="Set one default and continue"
                actionIcon={<IconStar className="h-4 w-4 text-red-600" />}
              />
              <StepCard
                step="Next"
                title="Start the gateway"
                description="Once the default is set, launch the gateway from the top bar so the chat connection comes online."
                actionLabel="Gateway start is step 3"
                actionIcon={<IconPlayerPlay className="h-4 w-4 text-lime-600" />}
              />
            </div>
          </div>
        </div>
      </div>
    )
  }

  if (!isConnected) {
    return (
      <div>
        <HostSummaryCard hostInfo={hostInfo} visible={showHostSplash} />
        <div className="relative overflow-hidden rounded-[2rem] border bg-gradient-to-br from-emerald-50 via-background to-background p-8 shadow-sm">
          <div className="absolute top-6 right-6 h-28 w-28 rounded-full bg-emerald-400/10 blur-3xl" />
          <div className="relative grid gap-6 lg:grid-cols-[minmax(0,1.05fr)_minmax(18rem,0.95fr)]">
            <div className="flex flex-col justify-center">
              <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-emerald-500/12 text-emerald-600">
                <IconPlugConnectedX className="h-7 w-7" />
              </div>
              <div className="mb-3 text-xs font-semibold tracking-[0.28em] uppercase text-emerald-700">
                Step 3 of 3
              </div>
              <h3 className="mb-3 text-3xl font-semibold tracking-tight">
                {t("chat.empty.notRunning")}
              </h3>
              <p className="text-muted-foreground max-w-xl text-sm leading-7">
                {t("chat.empty.notRunningDescription")}
              </p>
            </div>

            <div className="grid gap-3">
              <StepCard
                step="Do this now"
                title="Start the gateway"
                description="Use the green button in the top bar. JameClaw only needs a few seconds before the chat connects."
                actionLabel="Use Start Gateway above"
                actionIcon={
                  <IconPlayerPlay className="h-4 w-4 text-emerald-600" />
                }
              />
              <StepCard
                step="Tip"
                title="Prefer the terminal?"
                description='You can also run `jameclaw gateway` directly if you want the service started outside the web launcher.'
                actionLabel="CLI and web stay in sync"
                actionIcon={
                  <IconMessageCircle2 className="h-4 w-4 text-emerald-600" />
                }
              />
            </div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div>
      <HostSummaryCard hostInfo={hostInfo} visible={showHostSplash} />
      <div className="relative overflow-hidden rounded-[2rem] border bg-gradient-to-br from-zinc-50 via-background to-background p-8 shadow-sm">
        <div className="absolute right-6 bottom-4 h-36 w-36 rounded-full bg-zinc-300/20 blur-3xl" />
        <div className="relative grid gap-6 lg:grid-cols-[minmax(0,1.05fr)_minmax(18rem,0.95fr)]">
          <div className="flex flex-col justify-center">
            <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-zinc-900 text-white">
              <IconRobot className="h-7 w-7" />
            </div>
            <div className="mb-3 text-xs font-semibold tracking-[0.28em] uppercase text-zinc-500">
              Ready
            </div>
            <h3 className="mb-3 text-3xl font-semibold tracking-tight">
              {t("chat.welcome")}
            </h3>
            <p className="text-muted-foreground max-w-xl text-sm leading-7">
              {t("chat.welcomeDesc")}
            </p>
          </div>

          <div className="grid gap-3">
            {starterPrompts.map((prompt) => (
              <button
                key={prompt}
                type="button"
                className="bg-background/80 hover:bg-background flex items-start gap-3 rounded-2xl border p-4 text-left shadow-sm transition-colors"
                onClick={() => onPromptSelect(prompt)}
              >
                <IconSparkles className="mt-0.5 h-4 w-4 shrink-0 text-zinc-500" />
                <span className="text-sm leading-6">{prompt}</span>
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}

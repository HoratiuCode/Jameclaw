import {
  IconCheck,
  IconCircle,
  IconLoader2,
  IconPlayerPlay,
  IconRefresh,
  IconX,
} from "@tabler/icons-react"
import { useNavigate } from "@tanstack/react-router"
import { useMemo, useState } from "react"
import { toast } from "sonner"

import type { OnboardingStep } from "@/api/onboarding"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { useGateway } from "@/hooks/use-gateway"
import { useOnboardingStatus } from "@/hooks/use-onboarding-status"

const stepOrder = [
  "config",
  "workspace",
  "model",
  "credentials",
  "gateway",
  "channels",
  "chat",
]

function statusLabel(step: OnboardingStep) {
  if (step.status === "ready") return "Ready"
  if (step.status === "optional") return "Optional"
  if (step.status === "blocked") return "Blocked"
  return "Needs review"
}

function statusClass(step: OnboardingStep) {
  if (step.status === "ready") {
    return "border-green-500/30 bg-green-500/10 text-green-700 dark:text-green-300"
  }
  if (step.status === "optional") {
    return "border-blue-500/30 bg-blue-500/10 text-blue-700 dark:text-blue-300"
  }
  if (step.status === "blocked") {
    return "border-destructive/30 bg-destructive/10 text-destructive"
  }
  return "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300"
}

function StepIcon({ step }: { step: OnboardingStep }) {
  if (step.status === "ready") {
    return <IconCheck className="size-4" />
  }
  if (step.status === "blocked") {
    return <IconX className="size-4" />
  }
  return <IconCircle className="size-4" />
}

export function OnboardingWizard() {
  const navigate = useNavigate()
  const [dismissed, setDismissed] = useState(false)
  const {
    data,
    isLoading,
    error,
    refetch,
    completeOnboarding,
    completing,
  } = useOnboardingStatus()
  const { start, loading: gatewayLoading, canStart } = useGateway()

  const steps = useMemo(() => {
    const list = data?.steps ?? []
    return [...list].sort(
      (a, b) => stepOrder.indexOf(a.id) - stepOrder.indexOf(b.id),
    )
  }, [data?.steps])

  if (dismissed || isLoading || !data?.should_show) {
    return null
  }

  const nextStep = steps.find((step) => step.id === data.next_step_id)
  const canFinish = data.ready_for_chat || nextStep?.id === "chat"

  const goToStep = async (step: OnboardingStep) => {
    if (step.id === "gateway") {
      if (canStart) {
        await start()
      } else {
        toast.error(step.detail || "Gateway cannot start yet.")
      }
      void refetch()
      return
    }

    if (step.action_href) {
      await navigate({ to: step.action_href })
    }
  }

  const finish = async () => {
    try {
      await completeOnboarding()
      toast.success("Onboarding complete")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not complete onboarding")
    }
  }

  return (
    <div className="fixed inset-0 z-60 flex items-center justify-center bg-background/70 px-3 py-6 backdrop-blur-sm">
      <Card className="max-h-[92dvh] w-full max-w-3xl overflow-hidden rounded-lg shadow-xl">
        <CardHeader className="border-b">
          <div className="flex items-start justify-between gap-3">
            <div>
              <CardTitle className="text-xl">Set up JameClaw</CardTitle>
              <CardDescription>
                Complete the seven readiness checks before the agent is marked
                ready.
              </CardDescription>
            </div>
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label="Dismiss onboarding"
              onClick={() => setDismissed(true)}
            >
              <IconX className="size-4" />
            </Button>
          </div>
          <div className="mt-3 flex items-center gap-3">
            <div className="bg-muted h-2 flex-1 overflow-hidden rounded-full">
              <div
                className="bg-primary h-full rounded-full transition-all"
                style={{
                  width: `${Math.round((data.ready_count / data.total_count) * 100)}%`,
                }}
              />
            </div>
            <span className="text-muted-foreground text-xs font-medium">
              {data.ready_count}/{data.total_count}
            </span>
          </div>
        </CardHeader>

        <CardContent className="overflow-y-auto px-4 py-4 sm:px-6">
          {error ? (
            <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2 text-sm">
              Failed to load onboarding status.
            </div>
          ) : (
            <div className="grid gap-3">
              {steps.map((step, index) => (
                <div
                  key={step.id}
                  className="grid gap-3 rounded-lg border p-3 sm:grid-cols-[2rem_1fr_auto] sm:items-center"
                >
                  <div className="bg-muted text-muted-foreground flex size-8 items-center justify-center rounded-full text-sm font-semibold">
                    {index + 1}
                  </div>
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <h3 className="text-sm font-semibold">{step.title}</h3>
                      <span
                        className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium ${statusClass(step)}`}
                      >
                        <StepIcon step={step} />
                        {statusLabel(step)}
                      </span>
                    </div>
                    <p className="text-muted-foreground mt-1 text-sm">
                      {step.description}
                    </p>
                    {step.detail && (
                      <p className="text-muted-foreground mt-1 truncate text-xs">
                        {step.detail}
                      </p>
                    )}
                  </div>
                  <Button
                    variant={step.id === data.next_step_id ? "default" : "outline"}
                    size="sm"
                    className="w-full sm:w-auto"
                    disabled={step.id === "gateway" && gatewayLoading}
                    onClick={() => void goToStep(step)}
                  >
                    {step.id === "gateway" && gatewayLoading ? (
                      <IconLoader2 className="size-4 animate-spin" />
                    ) : step.id === "gateway" ? (
                      <IconPlayerPlay className="size-4" />
                    ) : null}
                    {step.action || "Open"}
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>

        <CardFooter className="flex flex-col gap-3 border-t sm:flex-row sm:items-center sm:justify-between">
          <p className="text-muted-foreground text-xs">
            Version {data.version} · {data.workspace || data.config_path}
          </p>
          <div className="flex w-full gap-2 sm:w-auto">
            <Button
              variant="outline"
              size="sm"
              className="flex-1 sm:flex-none"
              onClick={() => void refetch()}
            >
              <IconRefresh className="size-4" />
              Refresh
            </Button>
            <Button
              size="sm"
              className="flex-1 sm:flex-none"
              disabled={!canFinish || completing}
              onClick={() => void finish()}
            >
              {completing && <IconLoader2 className="size-4 animate-spin" />}
              Complete
            </Button>
          </div>
        </CardFooter>
      </Card>
    </div>
  )
}

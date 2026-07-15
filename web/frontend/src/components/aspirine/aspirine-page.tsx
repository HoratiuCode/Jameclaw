import {
  IconActivityHeartbeat,
  IconAlertTriangle,
  IconCircleCheck,
  IconRefresh,
  IconSparkles,
  IconTool,
} from "@tabler/icons-react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import {
  type AspirineIssue,
  getAspirineSummary,
  runAspirineAction,
} from "@/api/aspirine"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

export function AspirinePage() {
  const queryClient = useQueryClient()
  const { data, error, isFetching, isLoading, refetch } = useQuery({
    queryKey: ["aspirine"],
    queryFn: getAspirineSummary,
    refetchInterval: 20_000,
  })

  const actionMutation = useMutation({
    mutationFn: runAspirineAction,
    onSuccess: async () => {
      toast.success("Aspirine action completed")
      await queryClient.invalidateQueries({ queryKey: ["aspirine"] })
      await queryClient.invalidateQueries({ queryKey: ["gateway-status"] })
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Aspirine action failed")
    },
  })

  const issues = data?.issues ?? []

  return (
    <div className="bg-background flex h-full flex-col">
      <PageHeader title="Aspirine">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => void refetch()}
          disabled={isFetching}
        >
          <IconRefresh className={cn("size-4", isFetching && "animate-spin")} />
          Refresh
        </Button>
      </PageHeader>

      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
        <div className="mx-auto flex w-full max-w-6xl flex-col gap-5">
          <div className="grid gap-3 md:grid-cols-3">
            <StatusTile
              label="System state"
              value={data?.status ?? "checking"}
              tone={data?.status ?? "healthy"}
            />
            <StatusTile
              label="Active problems"
              value={String(data?.issue_count ?? 0)}
              tone={(data?.critical_count ?? 0) > 0 ? "critical" : "healthy"}
            />
            <StatusTile
              label="Last check"
              value={data?.checked_at ? formatTime(data.checked_at) : "..."}
              tone="healthy"
            />
          </div>

          {isLoading ? (
            <div className="text-muted-foreground text-sm">Checking system...</div>
          ) : error ? (
            <div className="border-destructive/30 bg-destructive/5 text-destructive rounded-lg border p-4 text-sm">
              Could not load Aspirine diagnostics.
            </div>
          ) : (
            <div className="grid gap-3">
              {issues.map((issue) => (
                <IssueCard
                  key={issue.id}
                  issue={issue}
                  runningAction={actionMutation.variables}
                  actionPending={actionMutation.isPending}
                  onRunAction={(action) => actionMutation.mutate(action)}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function StatusTile({
  label,
  value,
  tone,
}: {
  label: string
  value: string
  tone: "critical" | "warning" | "healthy" | string
}) {
  const Icon =
    tone === "critical"
      ? IconAlertTriangle
      : tone === "warning"
        ? IconActivityHeartbeat
        : IconCircleCheck

  return (
    <div className="border-border/70 bg-card flex items-center gap-3 rounded-lg border p-4">
      <div
        className={cn(
          "flex size-9 items-center justify-center rounded-md",
          tone === "critical" &&
            "bg-destructive/10 text-destructive",
          tone === "warning" &&
            "bg-amber-500/10 text-amber-700 dark:text-amber-300",
          tone !== "critical" &&
            tone !== "warning" &&
            "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
        )}
      >
        <Icon className="size-5" />
      </div>
      <div className="min-w-0">
        <div className="text-muted-foreground text-xs">{label}</div>
        <div className="text-foreground truncate text-lg font-semibold capitalize">
          {value}
        </div>
      </div>
    </div>
  )
}

function IssueCard({
  issue,
  runningAction,
  actionPending,
  onRunAction,
}: {
  issue: AspirineIssue
  runningAction?: string
  actionPending: boolean
  onRunAction: (action: string) => void
}) {
  const isHealthy = issue.status === "healthy"
  const Icon = isHealthy
    ? IconCircleCheck
    : issue.severity === "critical"
      ? IconAlertTriangle
      : IconSparkles
  const pendingThisAction =
    actionPending && runningAction === issue.auto_fix_action

  return (
    <article className="border-border/70 bg-card rounded-lg border p-4">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="flex min-w-0 gap-3">
          <div
            className={cn(
              "mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-md",
              issue.severity === "critical" &&
                "bg-destructive/10 text-destructive",
              issue.severity === "warning" &&
                "bg-amber-500/10 text-amber-700 dark:text-amber-300",
              issue.severity === "info" &&
                "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
            )}
          >
            <Icon className="size-5" />
          </div>
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="text-foreground text-sm font-semibold">
                {issue.title}
              </h3>
              <span className="border-border bg-muted text-muted-foreground rounded-md border px-2 py-0.5 text-xs capitalize">
                {issue.status.replaceAll("_", " ")}
              </span>
            </div>
            <p className="text-muted-foreground mt-2 text-sm">
              {issue.description}
            </p>
            <p className="text-foreground mt-3 text-sm">{issue.suggestion}</p>
            {issue.affected && issue.affected.length > 0 ? (
              <div className="mt-3 flex flex-wrap gap-1.5">
                {issue.affected.map((item) => (
                  <span
                    key={item}
                    className="bg-muted text-muted-foreground rounded-md px-2 py-0.5 text-xs"
                  >
                    {item}
                  </span>
                ))}
              </div>
            ) : null}
          </div>
        </div>
        {issue.auto_fix_action ? (
          <Button
            type="button"
            size="sm"
            className="w-full gap-2 lg:w-auto"
            disabled={actionPending}
            onClick={() => onRunAction(issue.auto_fix_action!)}
          >
            <IconTool className="size-4" />
            {pendingThisAction
              ? "Working..."
              : issue.auto_fix_label || "Run fix"}
          </Button>
        ) : null}
      </div>
    </article>
  )
}

function formatTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  })
}

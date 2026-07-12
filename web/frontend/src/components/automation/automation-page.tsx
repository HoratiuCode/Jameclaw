import {
  IconAlertCircle,
  IconCalendarTime,
  IconClockHour4,
  IconMessageCircle,
  IconPlus,
} from "@tabler/icons-react"
import { Link } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import type { ComponentType } from "react"

import { type AutomationItem, getAutomations } from "@/api/automation"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { cn } from "@/lib/utils"

export function AutomationPage() {
  const { data, error, isLoading } = useQuery({
    queryKey: ["automation"],
    queryFn: getAutomations,
  })

  const automations = data ?? []
  const activeCount = automations.filter((item) => item.enabled).length
  const nextAutomation = automations
    .filter((item) => item.enabled && item.next_run_at_ms)
    .sort((a, b) => (a.next_run_at_ms ?? 0) - (b.next_run_at_ms ?? 0))[0]

  return (
    <div className="bg-background flex h-full flex-col">
      <PageHeader title="Automations">
        <Button asChild size="sm" className="h-9 gap-2">
          <Link
            to="/"
            search={{
              prompt: "Create an automation for: ",
            }}
          >
            <IconPlus className="size-4" />
            <span className="hidden sm:inline">New automation</span>
          </Link>
        </Button>
      </PageHeader>

      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
        <div className="mx-auto flex w-full max-w-6xl flex-col gap-5">
          <div className="grid gap-3 md:grid-cols-3">
            <MetricCard
              label="Active automations"
              value={String(activeCount)}
              icon={IconCalendarTime}
            />
            <MetricCard
              label="Total configured"
              value={String(automations.length)}
              icon={IconClockHour4}
            />
            <MetricCard
              label="Next run"
              value={
                nextAutomation?.next_run_at_ms
                  ? formatDateTime(nextAutomation.next_run_at_ms)
                  : "None scheduled"
              }
              icon={IconMessageCircle}
            />
          </div>

          {isLoading ? (
            <div className="text-muted-foreground text-sm">Loading...</div>
          ) : error ? (
            <div className="text-destructive text-sm">
              Could not load automations.
            </div>
          ) : automations.length === 0 ? (
            <Card className="border-border/70 bg-card">
              <CardContent className="text-muted-foreground py-8 text-sm">
                No automations have been scheduled yet.
              </CardContent>
            </Card>
          ) : (
            <div className="grid gap-3">
              {automations.map((automation) => (
                <AutomationCard
                  key={automation.id}
                  automation={automation}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function MetricCard({
  label,
  value,
  icon: Icon,
}: {
  label: string
  value: string
  icon: ComponentType<{ className?: string }>
}) {
  return (
    <Card className="border-border/70 bg-card" size="sm">
      <CardContent className="flex items-center gap-3">
        <div className="bg-muted text-muted-foreground flex h-9 w-9 shrink-0 items-center justify-center rounded-md">
          <Icon className="h-4 w-4" />
        </div>
        <div className="min-w-0">
          <div className="text-muted-foreground text-xs">{label}</div>
          <div className="text-foreground mt-0.5 truncate text-sm font-medium">
            {value}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function AutomationCard({ automation }: { automation: AutomationItem }) {
  return (
    <Card
      className={cn(
        "border-border/70 bg-card gap-4",
        automation.status === "error" && "border-destructive/40",
      )}
      size="sm"
    >
      <CardHeader>
        <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
          <div className="min-w-0">
            <CardTitle className="truncate text-base">
              {automation.name}
            </CardTitle>
            <CardDescription className="mt-1">
              {automation.schedule}
            </CardDescription>
          </div>
          <StatusBadge status={automation.status} enabled={automation.enabled} />
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3 text-sm lg:grid-cols-[1.4fr_1fr_1fr]">
          <Field label="Automated request" value={automation.prompt} />
          <Field label="Delivery" value={automation.delivery} />
          <Field
            label="Next run"
            value={
              automation.next_run_at_ms
                ? formatDateTime(automation.next_run_at_ms)
                : "Not scheduled"
            }
          />
        </div>
        <div className="grid gap-3 text-sm md:grid-cols-3">
          <Field
            label="Last run"
            value={
              automation.last_run_at_ms
                ? formatDateTime(automation.last_run_at_ms)
                : "Never"
            }
          />
          <Field
            label="Last result"
            value={automation.last_status || "No result yet"}
          />
          <Field
            label="Created"
            value={formatDateTime(automation.created_at_ms)}
          />
        </div>
        {automation.last_error ? (
          <div className="text-destructive flex items-start gap-2 rounded-md border border-destructive/30 px-3 py-2 text-sm">
            <IconAlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
            <span className="break-words">{automation.last_error}</span>
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <div className="text-muted-foreground text-xs">{label}</div>
      <div className="text-foreground mt-1 break-words">{value}</div>
    </div>
  )
}

function StatusBadge({
  status,
  enabled,
}: {
  status: string
  enabled: boolean
}) {
  const label = enabled ? status : "disabled"
  return (
    <span
      className={cn(
        "w-fit shrink-0 rounded-md px-2 py-1 text-xs font-medium",
        label === "scheduled" && "bg-emerald-100 text-emerald-700",
        label === "waiting" && "bg-blue-100 text-blue-700",
        label === "disabled" && "bg-muted text-muted-foreground",
        label === "error" && "bg-destructive/10 text-destructive",
      )}
    >
      {label}
    </span>
  )
}

function formatDateTime(value: number) {
  if (!Number.isFinite(value) || value <= 0) {
    return "Unknown"
  }
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value))
}

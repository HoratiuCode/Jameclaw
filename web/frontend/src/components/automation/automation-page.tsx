import {
  IconAlertCircle,
  IconCalendarTime,
  IconClockHour4,
  IconMessageCircle,
  IconPlus,
  IconSparkles,
} from "@tabler/icons-react"
import { Link } from "@tanstack/react-router"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useState, type ComponentType } from "react"

import {
  type AutomationBlueprint,
  type AutomationBlueprintField,
  type AutomationItem,
  getAutomationBlueprints,
  getAutomations,
  instantiateAutomationBlueprint,
} from "@/api/automation"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { cn } from "@/lib/utils"

export function AutomationPage() {
  const { data, error, isLoading } = useQuery({
    queryKey: ["automation"],
    queryFn: getAutomations,
  })
  const { data: blueprints } = useQuery({
    queryKey: ["automation-blueprints"],
    queryFn: getAutomationBlueprints,
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

          <BlueprintGallery blueprints={blueprints ?? []} />

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

function BlueprintGallery({ blueprints }: { blueprints: AutomationBlueprint[] }) {
  if (blueprints.length === 0) {
    return null
  }
  return (
    <section className="space-y-3">
      <div className="flex items-center gap-2">
        <IconSparkles className="text-muted-foreground h-4 w-4" />
        <h2 className="text-sm font-medium">Automation blueprints</h2>
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        {blueprints.map((blueprint) => (
          <BlueprintCard key={blueprint.key} blueprint={blueprint} />
        ))}
      </div>
    </section>
  )
}

function BlueprintCard({ blueprint }: { blueprint: AutomationBlueprint }) {
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(false)
  const [values, setValues] = useState<Record<string, string>>(() =>
    initialBlueprintValues(blueprint),
  )
  const [error, setError] = useState<string | null>(null)

  const mutation = useMutation({
    mutationFn: () => instantiateAutomationBlueprint(blueprint.key, values),
    onSuccess: () => {
      setOpen(false)
      setError(null)
      setValues(initialBlueprintValues(blueprint))
      void queryClient.invalidateQueries({ queryKey: ["automation"] })
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : "Could not create automation")
    },
  })

  return (
    <Card className="border-border/70 bg-card" size="sm">
      <CardHeader>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            <CardTitle className="text-base">{blueprint.title}</CardTitle>
            <CardDescription className="mt-1">
              {blueprint.description}
            </CardDescription>
            <div className="mt-2 flex flex-wrap gap-1">
              {blueprint.tags.map((tag) => (
                <span
                  key={tag}
                  className="bg-muted text-muted-foreground rounded-md px-2 py-0.5 text-xs"
                >
                  {tag}
                </span>
              ))}
            </div>
          </div>
          <Button
            type="button"
            variant={open ? "outline" : "secondary"}
            size="sm"
            onClick={() => setOpen((current) => !current)}
          >
            {open ? "Cancel" : "Set up"}
          </Button>
        </div>
      </CardHeader>
      {open ? (
        <CardContent className="space-y-3 border-t pt-4">
          <div className="grid gap-3 md:grid-cols-2">
            {blueprint.fields.map((field) => (
              <BlueprintField
                key={field.name}
                field={field}
                value={values[field.name] ?? ""}
                onChange={(value) =>
                  setValues((current) => ({ ...current, [field.name]: value }))
                }
              />
            ))}
          </div>
          {error ? (
            <div className="text-destructive text-sm">{error}</div>
          ) : null}
          <Button
            type="button"
            size="sm"
            disabled={mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? "Scheduling..." : "Schedule blueprint"}
          </Button>
        </CardContent>
      ) : null}
    </Card>
  )
}

function BlueprintField({
  field,
  value,
  onChange,
}: {
  field: AutomationBlueprintField
  value: string
  onChange: (value: string) => void
}) {
  const options =
    field.name === "deliver"
      ? (field.options ?? []).filter((option) => option === "local")
      : field.options ?? []
  return (
    <label className="space-y-1.5">
      <span className="text-muted-foreground text-xs">{field.label}</span>
      {field.type === "enum" || field.type === "weekdays" ? (
        <Select value={value} onValueChange={onChange}>
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {options.map((option) => (
              <SelectItem key={option} value={option}>
                {option}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ) : (
        <Input
          type={field.type === "time" ? "time" : "text"}
          value={value}
          placeholder={field.help || field.label}
          onChange={(event) => onChange(event.target.value)}
        />
      )}
      {field.help ? (
        <span className="text-muted-foreground block text-xs">{field.help}</span>
      ) : null}
    </label>
  )
}

function initialBlueprintValues(blueprint: AutomationBlueprint) {
  return Object.fromEntries(
    blueprint.fields.map((field) => [field.name, field.default ?? ""]),
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

import { IconLoader2, IconRefresh, IconSearch } from "@tabler/icons-react"
import { useMemo, useState } from "react"

import { getUsage, type UsageResponse, type UsageSessionItem } from "@/api/usage"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"

import { useQuery } from "@tanstack/react-query"

function formatNumber(value: number) {
  return new Intl.NumberFormat().format(value)
}

function formatDate(value: string) {
  if (!value) return "-"
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value))
}

function todayMinus(days: number) {
  const date = new Date()
  date.setDate(date.getDate() - days)
  return date.toISOString().slice(0, 10)
}

export function UsagePage() {
  const [start, setStart] = useState(todayMinus(30))
  const [end, setEnd] = useState(new Date().toISOString().slice(0, 10))
  const [query, setQuery] = useState("")
  const [appliedQuery, setAppliedQuery] = useState("")
  const [sortKey, setSortKey] = useState<"updated" | "tokens" | "messages" | "tools">("tokens")
  const [selectedID, setSelectedID] = useState<string | null>(null)

  const { data, error, isLoading, refetch } = useQuery({
    queryKey: ["usage", start, end, appliedQuery],
    queryFn: () => getUsage({ start, end, q: appliedQuery }),
  })

  const sessions = useMemo(() => {
    const list = [...(data?.sessions ?? [])]
    return list.sort((a, b) => {
      switch (sortKey) {
        case "messages":
          return b.message_count - a.message_count
        case "tools":
          return b.tool_calls - a.tool_calls
        case "updated":
          return b.updated.localeCompare(a.updated)
        case "tokens":
        default:
          return b.estimated_tokens - a.estimated_tokens
      }
    })
  }, [data?.sessions, sortKey])

  const selected = sessions.find((session) => session.id === selectedID) ?? sessions[0] ?? null

  return (
    <div className="flex h-full flex-col">
      <PageHeader title="Usage">
        <Button variant="outline" size="sm" onClick={() => void refetch()}>
          <IconRefresh className="size-4" />
          Refresh
        </Button>
      </PageHeader>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 pb-8 sm:px-6">
        <div className="grid gap-4 py-4 lg:grid-cols-[1fr_22rem]">
          <div className="space-y-4">
            <UsageFilters
              start={start}
              end={end}
              query={query}
              sortKey={sortKey}
              onStartChange={setStart}
              onEndChange={setEnd}
              onQueryChange={setQuery}
              onApplyQuery={() => setAppliedQuery(query.trim())}
              onSortChange={setSortKey}
            />

            {isLoading ? (
              <div className="text-muted-foreground flex items-center gap-2 py-12 text-sm">
                <IconLoader2 className="size-4 animate-spin" />
                Loading usage
              </div>
            ) : error ? (
              <div className="text-destructive bg-destructive/10 rounded-lg px-4 py-3 text-sm">
                {error instanceof Error ? error.message : "Usage failed to load"}
              </div>
            ) : (
              <>
                <UsageTotals data={data} />
                <DailyBars days={data?.daily ?? []} />
                <SessionTable sessions={sessions} selectedID={selected?.id ?? null} onSelect={setSelectedID} />
              </>
            )}
          </div>

          <UsageDetail data={data} session={selected} />
        </div>
      </div>
    </div>
  )
}

function UsageFilters(props: {
  start: string
  end: string
  query: string
  sortKey: "updated" | "tokens" | "messages" | "tools"
  onStartChange: (value: string) => void
  onEndChange: (value: string) => void
  onQueryChange: (value: string) => void
  onApplyQuery: () => void
  onSortChange: (value: "updated" | "tokens" | "messages" | "tools") => void
}) {
  return (
    <Card size="sm">
      <CardContent className="grid gap-3 sm:grid-cols-[9rem_9rem_1fr_10rem_auto]">
        <Input type="date" value={props.start} onChange={(e) => props.onStartChange(e.target.value)} />
        <Input type="date" value={props.end} onChange={(e) => props.onEndChange(e.target.value)} />
        <div className="relative">
          <IconSearch className="text-muted-foreground pointer-events-none absolute top-2.5 left-2.5 size-4" />
          <Input
            className="pl-8"
            placeholder="Search sessions, roles, or tools"
            value={props.query}
            onChange={(e) => props.onQueryChange(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") props.onApplyQuery()
            }}
          />
        </div>
        <select
          className="border-input bg-background h-9 rounded-md border px-2 text-sm"
          value={props.sortKey}
          onChange={(e) => props.onSortChange(e.target.value as typeof props.sortKey)}
        >
          <option value="tokens">Est. tokens</option>
          <option value="messages">Messages</option>
          <option value="tools">Tool calls</option>
          <option value="updated">Updated</option>
        </select>
        <Button size="sm" onClick={props.onApplyQuery}>Apply</Button>
      </CardContent>
    </Card>
  )
}

function UsageTotals({ data }: { data?: UsageResponse }) {
  const totals = data?.totals
  const cards = [
    ["Sessions", totals?.sessions ?? 0],
    ["Messages", totals?.messages ?? 0],
    ["Est. tokens", totals?.estimated_tokens ?? 0],
    ["Tool calls", totals?.tool_calls ?? 0],
  ]
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      {cards.map(([label, value]) => (
        <Card key={label} size="sm">
          <CardContent>
            <div className="text-muted-foreground text-xs">{label}</div>
            <div className="mt-1 text-2xl font-semibold tabular-nums">{formatNumber(value as number)}</div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

function DailyBars({ days }: { days: UsageResponse["daily"] }) {
  const max = Math.max(...days.map((day) => day.estimated_tokens), 1)
  return (
    <Card>
      <CardHeader>
        <CardTitle>Daily Activity</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex h-36 items-end gap-1">
          {days.length === 0 ? (
            <div className="text-muted-foreground text-sm">No sessions in this range.</div>
          ) : (
            days.map((day) => (
              <div key={day.date} className="group flex min-w-3 flex-1 flex-col items-center justify-end gap-1">
                <div
                  className="bg-primary/75 w-full rounded-t-sm"
                  style={{ height: `${Math.max(6, (day.estimated_tokens / max) * 128)}px` }}
                  title={`${day.date}: ${formatNumber(day.estimated_tokens)} estimated tokens`}
                />
              </div>
            ))
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function SessionTable(props: {
  sessions: UsageSessionItem[]
  selectedID: string | null
  onSelect: (id: string) => void
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Sessions</CardTitle>
      </CardHeader>
      <CardContent className="overflow-x-auto">
        <table className="w-full min-w-180 text-sm">
          <thead className="text-muted-foreground border-b text-left text-xs">
            <tr>
              <th className="py-2 pr-3 font-medium">Session</th>
              <th className="py-2 pr-3 font-medium">Updated</th>
              <th className="py-2 pr-3 text-right font-medium">Messages</th>
              <th className="py-2 pr-3 text-right font-medium">Tools</th>
              <th className="py-2 text-right font-medium">Est. tokens</th>
            </tr>
          </thead>
          <tbody>
            {props.sessions.map((session) => (
              <tr
                key={session.id}
                className={`hover:bg-muted/50 cursor-pointer border-b last:border-0 ${
                  props.selectedID === session.id ? "bg-muted/70" : ""
                }`}
                onClick={() => props.onSelect(session.id)}
              >
                <td className="max-w-96 py-2 pr-3">
                  <div className="truncate font-medium">{session.title}</div>
                  <div className="text-muted-foreground truncate text-xs">{session.preview}</div>
                </td>
                <td className="text-muted-foreground py-2 pr-3 whitespace-nowrap">{formatDate(session.updated)}</td>
                <td className="py-2 pr-3 text-right tabular-nums">{formatNumber(session.message_count)}</td>
                <td className="py-2 pr-3 text-right tabular-nums">{formatNumber(session.tool_calls)}</td>
                <td className="py-2 text-right tabular-nums">{formatNumber(session.estimated_tokens)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </CardContent>
    </Card>
  )
}

function UsageDetail({ data, session }: { data?: UsageResponse; session: UsageSessionItem | null }) {
  const logs = (data?.logs ?? []).filter((log) => !session || log.session_id === session.id).slice(0, 40)
  return (
    <Card className="lg:sticky lg:top-18 lg:max-h-[calc(100svh-6rem)]">
      <CardHeader>
        <CardTitle>{session ? "Session Detail" : "Logs"}</CardTitle>
      </CardHeader>
      <CardContent className="min-h-0 overflow-y-auto">
        {session && (
          <div className="mb-4 space-y-2 text-sm">
            <div className="font-medium">{session.title}</div>
            <div className="text-muted-foreground break-all text-xs">{session.id}</div>
            <div className="grid grid-cols-3 gap-2 text-xs">
              <span>{formatNumber(session.roles.user)} user</span>
              <span>{formatNumber(session.roles.assistant)} assistant</span>
              <span>{formatNumber(session.roles.tool)} tool</span>
            </div>
          </div>
        )}
        <div className="space-y-2">
          {logs.map((log, index) => (
            <div key={`${log.session_id}-${index}`} className="bg-muted/40 rounded-md px-3 py-2 text-xs">
              <div className="text-muted-foreground mb-1 flex justify-between gap-2">
                <span className="font-medium">{log.role || "message"}</span>
                {log.tool_calls > 0 && <span>{log.tool_calls} tools</span>}
              </div>
              <div className="line-clamp-4 whitespace-pre-wrap">{log.content || "(tool call)"}</div>
            </div>
          ))}
          {logs.length === 0 && <div className="text-muted-foreground text-sm">No logs available.</div>}
        </div>
      </CardContent>
    </Card>
  )
}

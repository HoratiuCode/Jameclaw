import { useQuery } from "@tanstack/react-query"
import { IconMessageCircle, IconPin, IconPinned, IconRefresh } from "@tabler/icons-react"
import { useRouterState } from "@tanstack/react-router"

import { getSessions, setSessionPinned } from "@/api/sessions"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"

function formatDate(value: string) {
  if (!value) return "-"
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value))
}

function label(value?: string) {
  return value?.trim() || "unknown"
}

export function SessionsPage() {
  const fixedOnly = useRouterState({ select: (state) => state.location.href.includes("fixed=1") })
  const { data, error, isLoading, refetch, isFetching } = useQuery({
    queryKey: ["sessions", "dashboard"],
    queryFn: () => getSessions(0, 100),
    refetchInterval: 5000,
  })

  return (
    <div className="bg-background/95 flex h-full flex-col">
      <PageHeader title={fixedOnly ? "Fixed Chats" : "Sessions"}>
        <Button variant="outline" size="sm" onClick={() => void refetch()} disabled={isFetching}>
          <IconRefresh className={`size-4 ${isFetching ? "animate-spin" : ""}`} />
          Refresh
        </Button>
      </PageHeader>
      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
        {isLoading ? (
          <div className="text-muted-foreground text-sm">Loading...</div>
        ) : error ? (
          <div className="text-destructive text-sm">Could not load sessions.</div>
        ) : !data || data.filter((session) => !fixedOnly || session.pinned).length === 0 ? (
          <div className="border-border/70 bg-card text-muted-foreground rounded-lg border p-5 text-sm">
            No saved chat sessions yet.
          </div>
        ) : (
          <div className="divide-border overflow-hidden rounded-lg border">
            {data.filter((session) => !fixedOnly || session.pinned).map((session) => (
              <div key={session.id} className="bg-card px-4 py-3">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex min-w-0 gap-3">
                    <div className="bg-muted text-muted-foreground mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-md">
                      <IconMessageCircle className="size-4" />
                    </div>
                    <div className="min-w-0">
                      <div className="flex min-w-0 flex-wrap items-center gap-2">
                        <span className="text-foreground truncate text-sm font-medium">
                          {session.title}
                        </span>
                        <span className="border-border bg-muted text-muted-foreground rounded px-1.5 py-0.5 text-[11px] font-medium uppercase">
                          {label(session.channel)}
                        </span>
                        <span className="text-muted-foreground text-xs">
                          {label(session.chat_type)} · {label(session.chat_id)}
                        </span>
                      </div>
                      <div className="text-muted-foreground mt-1 truncate text-xs">
                        {session.preview}
                      </div>
                    </div>
                  </div>
                  <div className="text-muted-foreground shrink-0 text-right text-xs">
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label={session.pinned ? "Unfix chat" : "Fix chat"}
                      onClick={() => void setSessionPinned(session.id, !session.pinned).then(() => refetch())}
                    >
                      {session.pinned ? <IconPinned className="size-4" /> : <IconPin className="size-4" />}
                    </Button>
                    <div>{session.message_count} messages</div>
                    <div className="mt-1">{formatDate(session.updated)}</div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

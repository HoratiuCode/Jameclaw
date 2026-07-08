import { useQuery } from "@tanstack/react-query"

import { getSessions } from "@/api/sessions"
import { PageHeader } from "@/components/page-header"

export function SessionsPage() {
  const { data, error, isLoading } = useQuery({
    queryKey: ["sessions", "dashboard"],
    queryFn: () => getSessions(0, 100),
  })

  return (
    <div className="bg-background/95 flex h-full flex-col">
      <PageHeader title="Sessions" />
      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
        {isLoading ? (
          <div className="text-muted-foreground text-sm">Loading...</div>
        ) : error ? (
          <div className="text-destructive text-sm">Could not load sessions.</div>
        ) : !data || data.length === 0 ? (
          <div className="border-border/70 bg-card text-muted-foreground rounded-lg border p-5 text-sm">
            No saved chat sessions yet.
          </div>
        ) : (
          <div className="divide-border overflow-hidden rounded-lg border">
            {data.map((session) => (
              <div key={session.id} className="bg-card px-4 py-3">
                <div className="flex items-center justify-between gap-4">
                  <div className="min-w-0">
                    <div className="text-foreground truncate text-sm font-medium">
                      {session.title}
                    </div>
                    <div className="text-muted-foreground mt-1 truncate text-xs">
                      {session.preview}
                    </div>
                  </div>
                  <div className="text-muted-foreground shrink-0 text-xs">
                    {session.message_count} messages
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

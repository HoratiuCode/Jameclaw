import { useQuery } from "@tanstack/react-query"

import { getDashboardCards } from "@/api/dashboard"
import { PageHeader } from "@/components/page-header"

interface DashboardPageProps {
  title: string
  kind: string
  empty: string
}

export function DashboardPage({ title, kind, empty }: DashboardPageProps) {
  const { data, error, isLoading } = useQuery({
    queryKey: ["dashboard", kind],
    queryFn: () => getDashboardCards(kind),
  })

  return (
    <div className="bg-background/95 flex h-full flex-col">
      <PageHeader title={title} />
      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-6">
        {isLoading ? (
          <div className="text-muted-foreground text-sm">Loading...</div>
        ) : error ? (
          <div className="text-destructive text-sm">
            Could not load {title.toLowerCase()}.
          </div>
        ) : !data || data.length === 0 ? (
          <div className="border-border/70 bg-card text-muted-foreground rounded-lg border p-5 text-sm">
            {empty}
          </div>
        ) : (
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            {data.map((item) => (
              <article
                key={item.id}
                className="border-border/70 bg-card rounded-lg border p-4"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <h3 className="text-foreground truncate text-sm font-medium">
                      {item.title}
                    </h3>
                    {item.description && (
                      <p className="text-muted-foreground mt-2 line-clamp-3 text-sm">
                        {item.description}
                      </p>
                    )}
                  </div>
                  <span className="border-border bg-muted text-muted-foreground shrink-0 rounded-md border px-2 py-1 text-xs">
                    {item.status}
                  </span>
                </div>
                {typeof item.count === "number" && item.count > 0 && (
                  <div className="text-muted-foreground mt-4 text-xs">
                    {item.count} linked item{item.count === 1 ? "" : "s"}
                  </div>
                )}
              </article>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

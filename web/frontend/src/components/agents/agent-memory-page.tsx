import { IconArrowLeft, IconLoader2, IconRefresh } from "@tabler/icons-react"
import { Link } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"

import { getAgentMemory, type AgentMemory } from "@/api/agents"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

export function AgentMemoryPage({ agentID }: { agentID: string }) {
  const memoryQuery = useQuery({
    queryKey: ["agent-memory", agentID],
    queryFn: () => getAgentMemory(agentID),
  })

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={`Agent Memory: ${agentID}`}>
        <Button asChild variant="outline" size="sm">
          <Link to="/agents">
            <IconArrowLeft className="size-4" />
            Agents
          </Link>
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => void memoryQuery.refetch()}
          disabled={memoryQuery.isFetching}
        >
          <IconRefresh
            className={`size-4 ${memoryQuery.isFetching ? "animate-spin" : ""}`}
          />
          Refresh
        </Button>
      </PageHeader>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 pb-8 sm:px-6">
        <div className="mx-auto max-w-5xl py-4">
          <AgentMemoryCard
            memory={memoryQuery.data}
            error={memoryQuery.error}
            isLoading={memoryQuery.isLoading}
          />
        </div>
      </div>
    </div>
  )
}

function AgentMemoryCard({
  memory,
  error,
  isLoading,
}: {
  memory?: AgentMemory
  error: unknown
  isLoading: boolean
}) {
  const hasLongTerm = Boolean(memory?.long_term.trim())
  const hasHumanNotes = Boolean(memory?.human_notes?.trim())
  const hasDailyNotes = Boolean(memory?.daily_notes.length)
  const isEmpty = memory && !hasLongTerm && !hasHumanNotes && !hasDailyNotes

  return (
    <Card>
      <CardHeader>
        <CardTitle>Stored Memory</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4 text-sm">
        {isLoading ? (
          <div className="text-muted-foreground flex items-center gap-2">
            <IconLoader2 className="size-4 animate-spin" />
            Loading memory
          </div>
        ) : error ? (
          <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2">
            {error instanceof Error
              ? error.message
              : "Agent memory failed to load"}
          </div>
        ) : memory ? (
          <>
            <Field
              label="Workspace"
              value={memory.workspace || "Default workspace"}
              mono
            />
            <Field label="Long-term memory file" value={memory.memory_path} mono />
            {isEmpty ? (
              <div className="text-muted-foreground rounded-md border border-dashed px-3 py-4">
                No memory has been written for this agent yet.
              </div>
            ) : null}
            {hasHumanNotes ? (
              <MemoryBlock
                title="Configured human memory notes"
                content={memory.human_notes ?? ""}
              />
            ) : null}
            {hasLongTerm ? (
              <MemoryBlock title="Long-term memory" content={memory.long_term} />
            ) : null}
            {hasDailyNotes ? (
              <div className="space-y-3">
                <div className="text-muted-foreground text-xs">
                  Recent daily notes
                </div>
                {memory.daily_notes.map((note) => (
                  <MemoryBlock
                    key={note.path}
                    title={`${note.date} · ${note.path}`}
                    content={note.content}
                  />
                ))}
              </div>
            ) : null}
          </>
        ) : null}
      </CardContent>
    </Card>
  )
}

function Field({
  label,
  value,
  mono = false,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div>
      <div className="text-muted-foreground text-xs">{label}</div>
      <div className={`mt-1 break-words ${mono ? "font-mono text-xs" : "font-medium"}`}>
        {value}
      </div>
    </div>
  )
}

function MemoryBlock({ title, content }: { title: string; content: string }) {
  return (
    <section className="space-y-2">
      <div className="text-muted-foreground text-xs">{title}</div>
      <pre className="border-border/70 bg-muted/40 max-h-[60vh] overflow-auto rounded-md border p-3 text-xs whitespace-pre-wrap">
        {content}
      </pre>
    </section>
  )
}

import { IconArrowLeft, IconDeviceFloppy, IconLoader2, IconRefresh } from "@tabler/icons-react"
import { Link } from "@tanstack/react-router"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useState } from "react"
import { toast } from "sonner"

import { getAgentMemory, updateAgentMemory, type AgentMemory } from "@/api/agents"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

export function AgentMemoryPage({ agentID }: { agentID: string }) {
  const queryClient = useQueryClient()
  const memoryQuery = useQuery({
    queryKey: ["agent-memory", agentID],
    queryFn: () => getAgentMemory(agentID),
  })
  const saveMutation = useMutation({
    mutationFn: (longTerm: string) => updateAgentMemory(agentID, longTerm),
    onSuccess: async (memory) => {
      queryClient.setQueryData(["agent-memory", agentID], memory)
      toast.success("Long-term memory saved")
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "Could not save memory")
    },
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
            isSaving={saveMutation.isPending}
            onSave={(longTerm) => saveMutation.mutate(longTerm)}
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
  isSaving,
  onSave,
}: {
  memory?: AgentMemory
  error: unknown
  isLoading: boolean
  isSaving: boolean
  onSave: (longTerm: string) => void
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
            <LongTermMemoryEditor
              value={memory.long_term}
              isSaving={isSaving}
              onSave={onSave}
            />
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

export function LongTermMemoryEditor({
  value,
  isSaving,
  onSave,
}: {
  value: string
  isSaving: boolean
  onSave: (longTerm: string) => void
}) {
  const [draft, setDraft] = useState(value)
  useEffect(() => setDraft(value), [value])
  const dirty = draft !== value

  return (
    <section className="space-y-2">
      <div className="flex items-center justify-between gap-3">
        <div>
          <div className="text-muted-foreground text-xs">Long-term memory</div>
          <div className="text-muted-foreground mt-1 text-xs">
            Stable preferences, project facts, and standing instructions. Saved locally for this agent.
          </div>
        </div>
        <Button
          type="button"
          size="sm"
          disabled={!dirty || isSaving}
          onClick={() => onSave(draft)}
        >
          {isSaving ? <IconLoader2 className="size-4 animate-spin" /> : <IconDeviceFloppy className="size-4" />}
          Save memory
        </Button>
      </div>
      <textarea
        className="border-input bg-background min-h-64 w-full rounded-md border p-3 font-mono text-xs leading-5"
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
        placeholder="# Memory\n\n- User prefers concise status updates."
      />
    </section>
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

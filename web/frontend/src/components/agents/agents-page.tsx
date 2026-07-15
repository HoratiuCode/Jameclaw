import {
  IconBrain,
  IconDatabase,
  IconExternalLink,
  IconLoader2,
  IconPlus,
  IconRefresh,
  IconStar,
} from "@tabler/icons-react"
import { useEffect, useMemo, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import {
  createAgent,
  getAgents,
  updateAgent,
  type AgentSummary,
} from "@/api/agents"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"

function formatNumber(value: number) {
  return new Intl.NumberFormat().format(value)
}

const EMPTY_STRING_LIST: string[] = []

export function AgentsPage() {
  const [selectedID, setSelectedID] = useState<string | null>(null)
  const queryClient = useQueryClient()
  const { data, error, isLoading, refetch } = useQuery({
    queryKey: ["agents-admin"],
    queryFn: getAgents,
  })
  const mutation = useMutation({
    mutationFn: ({ id, body }: { id: string; body: Parameters<typeof updateAgent>[1] }) =>
      updateAgent(id, body),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["agents-admin"] })
      toast.success("Agent updated")
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Agent update failed")
    },
  })
  const createMutation = useMutation({
    mutationFn: createAgent,
    onSuccess: async (result) => {
      await queryClient.invalidateQueries({ queryKey: ["agents-admin"] })
      setSelectedID(result.id)
      toast.success("Subagent created")
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Subagent creation failed")
    },
  })

  const agents = useMemo(() => data?.agents ?? [], [data?.agents])
  const selected = useMemo(
    () => agents.find((agent) => agent.id === selectedID) ?? agents[0] ?? null,
    [agents, selectedID],
  )

  return (
    <div className="flex h-full flex-col">
      <PageHeader title="Agents">
        <Button variant="outline" size="sm" onClick={() => void refetch()}>
          <IconRefresh className="size-4" />
          Refresh
        </Button>
      </PageHeader>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 pb-8 sm:px-6">
        {isLoading ? (
          <div className="text-muted-foreground flex items-center gap-2 py-12 text-sm">
            <IconLoader2 className="size-4 animate-spin" />
            Loading agents
          </div>
        ) : error ? (
          <div className="text-destructive bg-destructive/10 mt-4 rounded-lg px-4 py-3 text-sm">
            {error instanceof Error ? error.message : "Agents failed to load"}
          </div>
        ) : (
          <div className="grid gap-4 py-4 lg:grid-cols-[18rem_1fr]">
            <AgentList agents={agents} selectedID={selected?.id ?? null} onSelect={setSelectedID} />
            <div className="space-y-4">
              <AgentOverview data={data} selected={selected} />
              <AgentPanels
                key={selected?.id ?? "none"}
                selected={selected}
                isSaving={mutation.isPending}
                isCreating={createMutation.isPending}
                defaultModel={data?.default_model ?? ""}
                onSave={(agent, body) => mutation.mutate({ id: agent.id, body })}
                onCreateSubagent={(body) => createMutation.mutate(body)}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function AgentList(props: {
  agents: AgentSummary[]
  selectedID: string | null
  onSelect: (id: string) => void
}) {
  return (
    <Card className="lg:sticky lg:top-18 lg:self-start">
      <CardHeader>
        <CardTitle>Agent Directory</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {props.agents.map((agent) => (
          <button
            key={agent.id}
            type="button"
            onClick={() => props.onSelect(agent.id)}
            className={`hover:bg-muted/70 flex w-full items-center gap-3 rounded-md px-3 py-2 text-left text-sm ${
              props.selectedID === agent.id ? "bg-muted" : ""
            }`}
          >
            <span className="bg-primary/10 text-primary flex size-8 shrink-0 items-center justify-center rounded-md">
              <IconBrain className="size-4" />
            </span>
            <span className="min-w-0 flex-1">
              <span className="flex items-center gap-1 truncate font-medium">
                {agent.name || agent.id}
                {agent.default && <IconStar className="size-3.5 shrink-0" />}
              </span>
              <span className="text-muted-foreground block truncate text-xs">{agent.model || "No model"}</span>
            </span>
          </button>
        ))}
      </CardContent>
    </Card>
  )
}

function AgentOverview({
  data,
  selected,
}: {
  data?: Awaited<ReturnType<typeof getAgents>>
  selected: AgentSummary | null
}) {
  const cards = [
    ["Configured models", data?.configured_models ?? 0],
    ["Enabled channels", data?.enabled_channels ?? 0],
    ["Main sessions", selected?.session_count ?? 0],
    ["Main tool calls", selected?.tool_calls ?? 0],
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

function AgentPanels({
  selected,
  isSaving,
  isCreating,
  defaultModel,
  onSave,
  onCreateSubagent,
}: {
  selected: AgentSummary | null
  isSaving: boolean
  isCreating: boolean
  defaultModel: string
  onSave: (agent: AgentSummary, body: Parameters<typeof updateAgent>[1]) => void
  onCreateSubagent: (body: Parameters<typeof createAgent>[0]) => void
}) {
  const selectedFallbacks = selected?.model_fallbacks ?? EMPTY_STRING_LIST
  const selectedSkills = selected?.skills ?? EMPTY_STRING_LIST
  const selectedSubagents = selected?.subagents ?? EMPTY_STRING_LIST
  const [model, setModel] = useState(selected?.model ?? "")
  const [fallbacksText, setFallbacksText] = useState(selectedFallbacks.join("\n"))
  const [humanAgentName, setHumanAgentName] = useState(selected?.human.agent_name ?? "")
  const [humanPersona, setHumanPersona] = useState(selected?.human.persona ?? "")
  const [humanTone, setHumanTone] = useState(selected?.human.tone ?? "")
  const [humanMode, setHumanMode] = useState(selected?.human.discussion_mode ?? "")
  const [humanStatus, setHumanStatus] = useState(selected?.human.status_style ?? "")
  const [humanNotes, setHumanNotes] = useState(selected?.human.memory_notes ?? "")

  useEffect(() => {
    setModel(selected?.model ?? "")
    setFallbacksText((selected?.model_fallbacks ?? EMPTY_STRING_LIST).join("\n"))
    setHumanAgentName(selected?.human.agent_name ?? "")
    setHumanPersona(selected?.human.persona ?? "")
    setHumanTone(selected?.human.tone ?? "")
    setHumanMode(selected?.human.discussion_mode ?? "")
    setHumanStatus(selected?.human.status_style ?? "")
    setHumanNotes(selected?.human.memory_notes ?? "")
  }, [selected])

  if (!selected) {
    return (
      <Card>
        <CardContent className="text-muted-foreground text-sm">No agent selected.</CardContent>
      </Card>
    )
  }

  return (
    <div className="grid gap-4 xl:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle>Overview</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <Field label="Agent ID" value={selected.id} mono />
          <Field label="Workspace" value={selected.workspace || "Default workspace"} mono />
          <Field label="Primary model" value={selected.model || "Not configured"} />
          <Field
            label="Fallbacks"
            value={selectedFallbacks.length > 0 ? selectedFallbacks.join(", ") : "None"}
          />
          <Button
            asChild
            variant="outline"
            size="sm"
            className="mt-2"
          >
            <a
              href={`/agent-memory/${encodeURIComponent(selected.id)}`}
              target="_blank"
              rel="noreferrer"
            >
              <IconDatabase className="size-4" />
              View memory
              <IconExternalLink className="size-3.5" />
            </a>
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Model Assignment</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <div>
            <div className="text-muted-foreground mb-1 text-xs">Primary model</div>
            <Input value={model} onChange={(e) => setModel(e.target.value)} placeholder="model name" />
          </div>
          <div>
            <div className="text-muted-foreground mb-1 text-xs">Fallback models</div>
            <textarea
              className="border-input bg-background min-h-24 w-full rounded-md border px-2.5 py-2 text-sm"
              value={fallbacksText}
              onChange={(e) => setFallbacksText(e.target.value)}
              placeholder="One fallback model per line"
            />
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              size="sm"
              disabled={isSaving}
              onClick={() =>
                onSave(selected, {
                  model: model.trim(),
                  model_fallbacks: fallbacksText
                    .split(/\n|,/)
                    .map((value) => value.trim())
                    .filter(Boolean),
                })
              }
            >
              Save model
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={isSaving || selected.default}
              onClick={() => onSave(selected, { default: true })}
            >
              {selected.default ? "Default agent" : "Set default"}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Runtime</CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-3 gap-3 text-sm">
          <Metric label="Sessions" value={selected.session_count} />
          <Metric label="Messages" value={selected.message_count} />
          <Metric label="Tool calls" value={selected.tool_calls} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Human Discussion</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <div>
            <div className="text-muted-foreground mb-1 text-xs">Default name</div>
            <Input
              value={humanAgentName}
              onChange={(event) => setHumanAgentName(event.target.value)}
              placeholder={selected.name || "JameClaw"}
            />
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            <div>
              <div className="text-muted-foreground mb-1 text-xs">Persona</div>
              <Input
                value={humanPersona}
                onChange={(event) => setHumanPersona(event.target.value)}
                placeholder="Senior engineer, research partner, coach"
              />
            </div>
            <div>
              <div className="text-muted-foreground mb-1 text-xs">Tone</div>
              <Input
                value={humanTone}
                onChange={(event) => setHumanTone(event.target.value)}
                placeholder="Direct, warm, concise"
              />
            </div>
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            <div>
              <div className="text-muted-foreground mb-1 text-xs">Discussion mode</div>
              <Input
                value={humanMode}
                onChange={(event) => setHumanMode(event.target.value)}
                placeholder="Collaborative, do-it-now, brainstorming"
              />
            </div>
            <div>
              <div className="text-muted-foreground mb-1 text-xs">Status updates</div>
              <Input
                value={humanStatus}
                onChange={(event) => setHumanStatus(event.target.value)}
                placeholder="Brief concrete progress updates"
              />
            </div>
          </div>
          <div>
            <div className="text-muted-foreground mb-1 text-xs">Memory notes</div>
            <textarea
              className="border-input bg-background min-h-24 w-full rounded-md border px-2.5 py-2 text-sm"
              value={humanNotes}
              onChange={(event) => setHumanNotes(event.target.value)}
              placeholder="Stable preferences, recurring context, or conversation rules"
            />
          </div>
          <Button
            size="sm"
            disabled={isSaving}
            onClick={() =>
              onSave(selected, {
                human: {
                  agent_name: humanAgentName.trim(),
                  persona: humanPersona.trim(),
                  tone: humanTone.trim(),
                  discussion_mode: humanMode.trim(),
                  status_style: humanStatus.trim(),
                  memory_notes: humanNotes.trim(),
                },
              })
            }
          >
            Save discussion style
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Skills</CardTitle>
        </CardHeader>
        <CardContent>
          <PillList values={selectedSkills} empty="No agent-specific skills configured." />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Subagents</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <PillList values={selectedSubagents} empty="No subagents allowed." />
          <CreateSubagentForm
            parent={selected}
            defaultModel={defaultModel}
            isCreating={isCreating}
            onCreate={onCreateSubagent}
          />
        </CardContent>
      </Card>
    </div>
  )
}

function CreateSubagentForm({
  parent,
  defaultModel,
  isCreating,
  onCreate,
}: {
  parent: AgentSummary
  defaultModel: string
  isCreating: boolean
  onCreate: (body: Parameters<typeof createAgent>[0]) => void
}) {
  const [name, setName] = useState("")
  const [id, setID] = useState("")
  const [model, setModel] = useState(parent.model || defaultModel)
  const [workspace, setWorkspace] = useState(parent.workspace)
  const [persona, setPersona] = useState("")
  const [tone, setTone] = useState(parent.human.tone)
  const [idTouched, setIDTouched] = useState(false)

  const handleNameChange = (value: string) => {
    setName(value)
    if (!idTouched) {
      setID(slugAgentID(value))
    }
  }

  const submit = () => {
    const cleanID = id.trim()
    if (!cleanID) {
      toast.error("Subagent ID is required")
      return
    }
    onCreate({
      id: cleanID,
      name: name.trim() || cleanID,
      model: model.trim(),
      workspace: workspace.trim(),
      parent_id: parent.id,
      human: {
        agent_name: name.trim() || cleanID,
        persona: persona.trim(),
        tone: tone.trim(),
        discussion_mode: "collaborative",
        status_style: "brief concrete progress updates",
      },
    })
  }

  return (
    <div className="border-border/70 space-y-3 border-t pt-4 text-sm">
      <div className="grid gap-3 sm:grid-cols-2">
        <div>
          <div className="text-muted-foreground mb-1 text-xs">Subagent name</div>
          <Input
            value={name}
            onChange={(event) => handleNameChange(event.target.value)}
            placeholder="Research assistant"
          />
        </div>
        <div>
          <div className="text-muted-foreground mb-1 text-xs">Agent ID</div>
          <Input
            value={id}
            onChange={(event) => {
              setIDTouched(true)
              setID(event.target.value)
            }}
            placeholder="research-assistant"
          />
        </div>
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <div>
          <div className="text-muted-foreground mb-1 text-xs">Primary model</div>
          <Input value={model} onChange={(event) => setModel(event.target.value)} placeholder="model name" />
        </div>
        <div>
          <div className="text-muted-foreground mb-1 text-xs">Workspace</div>
          <Input
            value={workspace}
            onChange={(event) => setWorkspace(event.target.value)}
            placeholder="Default workspace"
          />
        </div>
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <div>
          <div className="text-muted-foreground mb-1 text-xs">Persona</div>
          <Input
            value={persona}
            onChange={(event) => setPersona(event.target.value)}
            placeholder="Research partner"
          />
        </div>
        <div>
          <div className="text-muted-foreground mb-1 text-xs">Tone</div>
          <Input value={tone} onChange={(event) => setTone(event.target.value)} placeholder="Direct and warm" />
        </div>
      </div>
      <Button size="sm" disabled={isCreating} onClick={submit}>
        {isCreating ? <IconLoader2 className="size-4 animate-spin" /> : <IconPlus className="size-4" />}
        Create subagent
      </Button>
    </div>
  )
}

function slugAgentID(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_-]+/g, "-")
    .replace(/^-+|-+$/g, "")
}

function Field({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <div className="text-muted-foreground text-xs">{label}</div>
      <div className={`mt-1 break-words ${mono ? "font-mono text-xs" : "font-medium"}`}>{value}</div>
    </div>
  )
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="bg-muted/50 rounded-md px-3 py-2">
      <div className="text-muted-foreground text-xs">{label}</div>
      <div className="mt-1 text-lg font-semibold tabular-nums">{formatNumber(value)}</div>
    </div>
  )
}

function PillList({ values, empty }: { values: string[] | null | undefined; empty: string }) {
  if (!values || values.length === 0) {
    return <div className="text-muted-foreground text-sm">{empty}</div>
  }
  return (
    <div className="flex flex-wrap gap-2">
      {values.map((value) => (
        <span key={value} className="bg-muted rounded-md px-2 py-1 text-xs font-medium">
          {value}
        </span>
      ))}
    </div>
  )
}

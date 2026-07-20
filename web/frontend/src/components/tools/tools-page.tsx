import { IconLoader2 } from "@tabler/icons-react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { type MCPServer, type ToolSupportItem, getMCPServers, getTools, saveMCPServer, setToolEnabled } from "@/api/tools"
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
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"

export function ToolsPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { data, isLoading, error } = useQuery({
    queryKey: ["tools"],
    queryFn: getTools,
  })

  const toggleMutation = useMutation({
    mutationFn: async ({ name, enabled }: { name: string; enabled: boolean }) =>
      setToolEnabled(name, enabled),
    onSuccess: (_, variables) => {
      toast.success(
        variables.enabled
          ? t("pages.agent.tools.enable_success")
          : t("pages.agent.tools.disable_success"),
      )
      void queryClient.invalidateQueries({ queryKey: ["tools"] })
    },
    onError: (err) => {
      toast.error(
        err instanceof Error
          ? err.message
          : t("pages.agent.tools.toggle_error"),
      )
    },
  })

  const groupedTools = (() => {
    if (!data) return [] as Array<[string, ToolSupportItem[]]>
    const buckets = new Map<string, ToolSupportItem[]>()
    for (const item of data.tools) {
      const list = buckets.get(item.category) ?? []
      list.push(item)
      buckets.set(item.category, list)
    }
    return Array.from(buckets.entries())
  })()

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("navigation.tools")} />

      <div className="flex-1 overflow-auto px-6 py-3">
        <div className="w-full max-w-6xl space-y-6">
          {isLoading ? (
            <div className="text-muted-foreground py-6 text-sm">
              {t("labels.loading")}
            </div>
          ) : error ? (
            <div className="text-destructive py-6 text-sm">
              {t("pages.agent.load_error")}
            </div>
          ) : (
            <section className="space-y-5">
              <p className="text-muted-foreground mt-1 text-sm">
                {t("pages.agent.tools.description")}
              </p>

              <MCPSetupCard />

              {data?.tools.length ? (
                groupedTools.map(([category, items]) => (
                  <div key={category} className="space-y-3">
                    <div className="text-foreground/85 text-sm font-semibold tracking-wide">
                      {t(`pages.agent.tools.categories.${category}`)}
                    </div>
                    <div className="grid gap-4 lg:grid-cols-2">
                      {items.map((tool) => {
                        const reasonText = tool.reason_code
                          ? t(`pages.agent.tools.reasons.${tool.reason_code}`)
                          : ""
                        const isPending =
                          toggleMutation.isPending &&
                          toggleMutation.variables?.name === tool.name
                        const nextEnabled = tool.status !== "enabled"

                        return (
                          <Card
                            key={tool.name}
                            className={cn(
                              "gap-4 border transition-colors",
                              tool.status === "enabled" &&
                                "border-emerald-200/70 bg-emerald-50/50",
                              tool.status === "blocked" &&
                                "border-amber-200/80 bg-amber-50/60",
                              tool.status === "disabled" &&
                                "border-border/60 bg-card/70",
                            )}
                            size="sm"
                          >
                            <CardHeader>
                              <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                                <div className="min-w-0 flex-1">
                                  <CardTitle className="font-mono text-sm break-all">
                                    {tool.name}
                                  </CardTitle>
                                  <CardDescription className="mt-1 break-words">
                                    {tool.description}
                                  </CardDescription>
                                </div>
                                <div className="flex shrink-0 items-center gap-2 self-start">
                                  <ToolStatusBadge status={tool.status} />
                                  <Button
                                    variant={
                                      nextEnabled ? "default" : "outline"
                                    }
                                    size="sm"
                                    disabled={isPending}
                                    onClick={() =>
                                      toggleMutation.mutate({
                                        name: tool.name,
                                        enabled: nextEnabled,
                                      })
                                    }
                                  >
                                    {isPending ? (
                                      <IconLoader2 className="size-4 animate-spin" />
                                    ) : null}
                                    {nextEnabled
                                      ? t("pages.agent.tools.enable")
                                      : t("pages.agent.tools.disable")}
                                  </Button>
                                </div>
                              </div>
                            </CardHeader>
                            <CardContent className="space-y-2">
                              <div className="text-muted-foreground text-xs">
                                {t("pages.agent.tools.config_key", {
                                  key: tool.config_key,
                                })}
                              </div>
                              {reasonText ? (
                                <div className="text-sm text-amber-800">
                                  {reasonText}
                                </div>
                              ) : null}
                            </CardContent>
                          </Card>
                        )
                      })}
                    </div>
                  </div>
                ))
              ) : (
                <Card className="border-dashed">
                  <CardContent className="text-muted-foreground py-10 text-center text-sm">
                    {t("pages.agent.tools.empty")}
                  </CardContent>
                </Card>
              )}
            </section>
          )}
        </div>
      </div>
    </div>
  )
}

function MCPSetupCard() {
  const [open, setOpen] = useState(false)
  const queryClient = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: ["mcp-servers"], queryFn: getMCPServers })

  return (
    <Card className="border-primary/25 bg-primary/3">
      <CardHeader>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <CardTitle>Connect MCP servers</CardTitle>
            <CardDescription className="mt-1">
              MCP (Model Context Protocol) lets Jame use tools from another application or service, through a local CLI or a remote endpoint.
            </CardDescription>
          </div>
          <Button onClick={() => setOpen(true)}>New connection</Button>
        </div>
      </CardHeader>
      <CardContent>
        <MCPServerList loading={isLoading} servers={data?.servers ?? []} />
      </CardContent>
      <MCPConnectionSheet open={open} onOpenChange={setOpen} onSaved={() => {
        void queryClient.invalidateQueries({ queryKey: ["mcp-servers"] })
        void queryClient.invalidateQueries({ queryKey: ["tools"] })
      }} />
    </Card>
  )
}

function MCPConnectionSheet({ open, onOpenChange, onSaved }: { open: boolean; onOpenChange: (open: boolean) => void; onSaved: () => void }) {
  const [name, setName] = useState("")
  const [transport, setTransport] = useState<"stdio" | "http" | "sse">("stdio")
  const [command, setCommand] = useState("")
  const [args, setArgs] = useState("")
  const [url, setURL] = useState("")
  const [enabled, setEnabled] = useState(true)
  const saveMutation = useMutation({
    mutationFn: saveMCPServer,
    onSuccess: () => {
      toast.success("MCP server saved. The gateway was refreshed to load it.")
      onSaved()
      onOpenChange(false)
      setName("")
      setCommand("")
      setArgs("")
      setURL("")
    },
    onError: (error) => toast.error(error instanceof Error ? error.message : "Could not save MCP server."),
  })

  const submit = () => {
    saveMutation.mutate({
      name,
      enabled,
      transport,
      command: transport === "stdio" ? command : undefined,
      args: transport === "stdio" ? args.split("\n").map((value) => value.trim()).filter(Boolean) : undefined,
      url: transport === "stdio" ? undefined : url,
    })
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>New MCP connection</SheetTitle>
          <SheetDescription>Connect a local CLI MCP server, such as a database or GitHub integration, or a remote HTTP/SSE MCP endpoint.</SheetDescription>
        </SheetHeader>
        <div className="space-y-4 px-4">
        <div className="grid gap-3 md:grid-cols-3">
          <div className="grid gap-2"><Label htmlFor="mcp-name">Server name</Label><Input id="mcp-name" value={name} onChange={(event) => setName(event.target.value)} placeholder="github" /></div>
          <div className="grid gap-2"><Label>Connection type</Label><Select value={transport} onValueChange={(value) => setTransport(value as "stdio" | "http" | "sse")}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="stdio">Local CLI (stdio)</SelectItem><SelectItem value="http">Remote HTTP</SelectItem><SelectItem value="sse">Remote SSE</SelectItem></SelectContent></Select></div>
          <div className="flex items-end gap-2 pb-2"><Switch checked={enabled} onCheckedChange={setEnabled} /><Label>Enable this server</Label></div>
        </div>
        {transport === "stdio" ? (
          <div className="grid gap-3 md:grid-cols-2">
            <div className="grid gap-2"><Label htmlFor="mcp-command">Command</Label><Input id="mcp-command" value={command} onChange={(event) => setCommand(event.target.value)} placeholder="npx" /></div>
            <div className="grid gap-2"><Label htmlFor="mcp-args">Arguments — one per line</Label><Textarea id="mcp-args" value={args} onChange={(event) => setArgs(event.target.value)} placeholder={"-y\n@modelcontextprotocol/server-github"} /></div>
          </div>
        ) : <div className="grid gap-2"><Label htmlFor="mcp-url">Server URL</Label><Input id="mcp-url" value={url} onChange={(event) => setURL(event.target.value)} placeholder="https://example.com/mcp" /></div>}
          <p className="text-muted-foreground text-xs">For API keys or headers, add them in the config file under <code>tools.mcp.servers</code>; they are intentionally not shown in this page.</p>
        </div>
        <SheetFooter><Button onClick={submit} disabled={saveMutation.isPending}>{saveMutation.isPending ? "Connecting…" : "Save and connect"}</Button></SheetFooter>
      </SheetContent>
    </Sheet>
  )
}

function MCPServerList({ loading, servers }: { loading: boolean; servers: MCPServer[] }) {
  if (loading) return <p className="text-muted-foreground text-sm">Loading MCP servers…</p>
  if (!servers.length) return <p className="text-muted-foreground text-sm">No MCP servers connected yet. Select New connection to add a local CLI or remote endpoint.</p>
  return <div className="space-y-2 border-t pt-3">{servers.map((server) => <div key={server.name} className="flex flex-wrap items-center justify-between gap-2 text-sm"><span className="font-medium">{server.name}</span><span className="text-muted-foreground">{server.transport === "stdio" ? [server.command, ...(server.args ?? [])].filter(Boolean).join(" ") : server.url}</span><span className={server.enabled ? "text-emerald-700" : "text-muted-foreground"}>{server.enabled ? "Enabled" : "Disabled"}</span></div>)}</div>
}

function ToolStatusBadge({ status }: { status: ToolSupportItem["status"] }) {
  const { t } = useTranslation()

  return (
    <span
      className={cn(
        "shrink-0 rounded-md px-2 py-1 text-[11px] font-semibold",
        status === "enabled" && "bg-emerald-100 text-emerald-700",
        status === "blocked" && "bg-amber-100 text-amber-700",
        status === "disabled" && "bg-muted text-muted-foreground",
      )}
    >
      {t(`pages.agent.tools.status.${status}`)}
    </span>
  )
}

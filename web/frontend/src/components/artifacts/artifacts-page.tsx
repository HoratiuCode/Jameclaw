import {
  IconChevronRight,
  IconCode,
  IconDeviceFloppy,
  IconExternalLink,
  IconFileCode,
  IconFolder,
  IconFolderOpen,
  IconPlus,
  IconPlayerPlay,
  IconPlayerStop,
  IconRefresh,
  IconTrash,
} from "@tabler/icons-react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useMemo, useState } from "react"
import { toast } from "sonner"

import {
  artifactPreviewURL,
  createArtifact,
  deleteArtifact,
  getArtifacts,
  getArtifactFiles,
  type Artifact,
  type ArtifactKind,
  updateArtifact,
} from "@/api/artifacts"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"

type Draft = Pick<Artifact, "name" | "kind" | "html" | "css" | "javascript">
type SourceFile = "html" | "css" | "javascript"

const starterApp: Draft = {
  name: "Untitled app",
  kind: "app",
  html: '<main class="app">\n  <h1>Hello from JameClaw</h1>\n  <p>Build something useful.</p>\n</main>',
  css: "body { margin: 0; font-family: system-ui, sans-serif; }\n.app { padding: 48px; }",
  javascript: "",
}

const starterCode: Draft = {
  name: "Untitled code",
  kind: "code",
  html: "",
  css: "",
  javascript: "// Keep reusable code, notes, or a work-in-progress here.\n",
}

function toDraft(artifact: Artifact): Draft {
  return {
    name: artifact.name,
    kind: artifact.kind,
    html: artifact.html,
    css: artifact.css,
    javascript: artifact.javascript,
  }
}

function formatDate(value: number) {
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(value)
}

export function ArtifactsPage() {
  const queryClient = useQueryClient()
  const { data: artifacts = [], isLoading, isFetching } = useQuery({ queryKey: ["artifacts"], queryFn: getArtifacts })
  const [selectedID, setSelectedID] = useState<string | null>(null)
  const [draft, setDraft] = useState<Draft>(starterApp)
  const [previewVersion, setPreviewVersion] = useState(0)
  const [runningArtifactID, setRunningArtifactID] = useState<string | null>(null)
  const [folderOpen, setFolderOpen] = useState(true)
  const [selectedFile, setSelectedFile] = useState<SourceFile>("html")

  const selected = useMemo(() => artifacts.find((artifact) => artifact.id === selectedID) ?? null, [artifacts, selectedID])
  const { data: projectFiles } = useQuery({
    queryKey: ["artifact-files", selectedID],
    queryFn: () => getArtifactFiles(selectedID!),
    enabled: Boolean(selectedID),
  })

  useEffect(() => {
    if (selected) {
      setDraft(toDraft(selected))
      setSelectedFile("html")
    }
    else if (artifacts.length > 0) setSelectedID(artifacts[0].id)
  }, [selected, artifacts])

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["artifacts"] })
  const save = useMutation({
    mutationFn: async () => {
      if (selected) return updateArtifact(selected.id, draft)
      return createArtifact(draft)
    },
    onSuccess: (artifact) => {
      setSelectedID(artifact.id)
      setPreviewVersion((value) => value + 1)
      void invalidate()
      toast.success("Artifact saved")
    },
    onError: (error) => toast.error(error instanceof Error ? error.message : "Could not save artifact"),
  })
  const remove = useMutation({
    mutationFn: deleteArtifact,
    onSuccess: () => {
      setRunningArtifactID(null)
      setSelectedID(null)
      setDraft(starterApp)
      void invalidate()
      toast.success("Artifact deleted")
    },
    onError: () => toast.error("Could not delete artifact"),
  })

  const startNew = (kind: ArtifactKind) => {
    setSelectedID(null)
    setDraft(kind === "app" ? { ...starterApp } : { ...starterCode })
    setSelectedFile("html")
  }

  const sourceFiles: { key: SourceFile; name: string; label: string; value: string; placeholder: string }[] = [
    { key: "html", name: "index.html", label: "HTML", value: draft.html, placeholder: "HTML markup" },
    { key: "css", name: "styles.css", label: "CSS", value: draft.css, placeholder: "Styles" },
    { key: "javascript", name: "script.js", label: "JavaScript", value: draft.javascript, placeholder: "JavaScript" },
  ]
  const activeSourceFile = sourceFiles.find((file) => file.key === selectedFile) ?? sourceFiles[0]
  const setActiveFileContent = (value: string) => setDraft({ ...draft, [selectedFile]: value })
  const canRunSelected = selected?.kind === "app"
  const isRunning = selected?.id === runningArtifactID

  return (
    <div className="bg-background flex h-full min-h-0 flex-col">
      <PageHeader title="Artifacts" titleExtra={<span className="text-muted-foreground hidden text-sm font-normal sm:inline">Apps and code made with JameClaw</span>}>
        <Button variant="outline" size="sm" onClick={() => void invalidate()} disabled={isFetching}>
          <IconRefresh className={isFetching ? "size-4 animate-spin" : "size-4"} /> Refresh
        </Button>
        <Button variant="outline" size="sm" onClick={() => startNew("code")}><IconCode /> New code</Button>
        <Button size="sm" onClick={() => startNew("app")}><IconPlus /> New app</Button>
      </PageHeader>
      <div className="grid min-h-0 flex-1 grid-cols-1 border-t lg:grid-cols-[250px_minmax(0,1fr)_minmax(320px,0.9fr)]">
        <aside className="border-border min-h-0 overflow-y-auto border-r p-3">
          <p className="text-muted-foreground px-2 pb-2 text-xs font-medium tracking-wide uppercase">Saved artifacts</p>
          {isLoading ? <p className="text-muted-foreground px-2 text-sm">Loading...</p> : null}
          {!isLoading && artifacts.length === 0 ? <p className="text-muted-foreground rounded-md border border-dashed p-3 text-sm">Create an app or code artifact. JameClaw saves it in your workspace.</p> : null}
          <div className="space-y-1">
            {artifacts.map((artifact) => {
              const isSelected = artifact.id === selectedID
              return <div key={artifact.id} className={`rounded-md transition-colors ${isSelected ? "bg-accent text-foreground" : "hover:bg-muted text-muted-foreground"}`}>
                <button onClick={() => { setSelectedID(artifact.id); setFolderOpen(true) }} onDoubleClick={() => { setSelectedID(artifact.id); setFolderOpen(true) }} className="w-full px-3 py-2 text-left">
                  <span className="flex items-center gap-2"><IconChevronRight className={`size-3.5 transition-transform ${isSelected && folderOpen ? "rotate-90" : ""}`} />{isSelected && folderOpen ? <IconFolderOpen className="size-4" /> : <IconFolder className="size-4" />}<span className="truncate text-sm font-medium">{artifact.name}</span></span>
                  <span className="mt-1 block pl-9 text-[11px] opacity-70">{artifact.kind === "app" ? "Runnable app" : "Code"} · {formatDate(artifact.updated_at_ms)}</span>
                </button>
                {isSelected && folderOpen ? <div className="mb-2 ml-5 border-l border-border/70 pl-2">
                  <p className="text-muted-foreground px-2 py-1 text-[10px]">artifacts/{artifact.id}</p>
                  {sourceFiles.map((file) => {
                    const stored = projectFiles?.files.find((item) => item.name === file.name)
                    return <button key={file.key} onClick={() => setSelectedFile(file.key)} className={`flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-xs ${file.key === selectedFile ? "bg-background/80 text-foreground" : "hover:bg-background/50"}`}><IconFileCode className="size-3.5" /><span className="flex-1">{file.name}</span><span className="opacity-60">{stored ? `${stored.size} B` : "new"}</span></button>
                  })}
                </div> : null}
              </div>
            })}
          </div>
        </aside>
        <section className="min-h-0 overflow-y-auto p-4 sm:p-5">
          <div className="mb-4 flex flex-wrap items-center gap-2">
            <Input value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} aria-label="Artifact name" className="max-w-sm font-medium" />
            <select value={draft.kind} onChange={(event) => setDraft({ ...draft, kind: event.target.value as ArtifactKind })} className="border-input bg-background h-9 rounded-md border px-2.5 text-sm">
              <option value="app">Runnable app</option><option value="code">Code only</option>
            </select>
            {selected ? <span className="text-muted-foreground text-xs">Saved {formatDate(selected.updated_at_ms)}</span> : <span className="text-muted-foreground text-xs">New, unsaved artifact</span>}
          </div>
          <div className="mb-2 flex items-center gap-2"><IconFileCode className="text-muted-foreground size-4" /><span className="text-sm font-medium">{activeSourceFile.name}</span><span className="text-muted-foreground text-xs">Project file</span></div>
          <Editor label={activeSourceFile.label} value={activeSourceFile.value} onChange={setActiveFileContent} placeholder={activeSourceFile.placeholder} />
          <div className="mt-4 flex items-center gap-2">
            <Button onClick={() => save.mutate()} disabled={save.isPending || !draft.name.trim()}><IconDeviceFloppy /> {save.isPending ? "Saving..." : "Save artifact"}</Button>
            {selected ? <Button variant="destructive" size="sm" onClick={() => { if (window.confirm(`Delete “${selected.name}”?`)) remove.mutate(selected.id) }} disabled={remove.isPending}><IconTrash /> Delete</Button> : null}
          </div>
        </section>
        <section className="bg-muted/20 border-border flex min-h-0 flex-col border-l">
          <div className="flex h-12 shrink-0 items-center justify-between gap-2 border-b px-4"><span className="text-sm font-medium">App runner</span><div className="flex items-center gap-1">{canRunSelected ? <Button variant={isRunning ? "secondary" : "default"} size="sm" onClick={() => { setRunningArtifactID(selected.id); setPreviewVersion((value) => value + 1) }}><IconPlayerPlay /> {isRunning ? "Restart" : "Start app"}</Button> : null}{isRunning ? <Button variant="ghost" size="icon-sm" aria-label="Stop app" onClick={() => setRunningArtifactID(null)}><IconPlayerStop /></Button> : null}{canRunSelected ? <Button variant="ghost" size="icon-sm" aria-label="Open app in a new tab" onClick={() => window.open(artifactPreviewURL(selected!.id), "_blank", "noopener,noreferrer")}><IconExternalLink /></Button> : null}</div></div>
          {canRunSelected && isRunning ? <iframe key={`${selected!.id}-${previewVersion}`} title={`${selected!.name} preview`} sandbox="allow-scripts allow-forms allow-modals" src={artifactPreviewURL(selected!.id)} className="bg-white min-h-0 w-full flex-1" /> : canRunSelected ? <div className="text-muted-foreground flex flex-1 items-center justify-center p-8 text-center text-sm">This website is saved and ready. Press <strong className="mx-1 text-foreground">Start app</strong> to run it inside JameClaw Desktop.</div> : <div className="text-muted-foreground flex flex-1 items-center justify-center p-8 text-center text-sm">Save a runnable app to start it here. Code-only artifacts are kept for reference and future edits.</div>}
        </section>
      </div>
    </div>
  )
}

function Editor({ label, value, onChange, placeholder }: { label: string; value: string; onChange: (value: string) => void; placeholder: string }) {
  return <label className="block"><span className="mb-1.5 block text-xs font-semibold tracking-wide text-muted-foreground uppercase">{label}</span><Textarea value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} spellCheck={false} className="min-h-36 resize-y font-mono text-xs leading-5" /></label>
}

import { IconBook2, IconCommand, IconLoader2 } from "@tabler/icons-react"
import { useQuery } from "@tanstack/react-query"

import { getLearnedSkills, type LearnedSkillItem } from "@/api/skills"
import { PageHeader } from "@/components/page-header"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"

function formatOrigin(skill: LearnedSkillItem): string {
  const origin = skill.origin
  if (!origin) {
    return skill.source
  }

  if (origin.kind === "registry") {
    const pieces = [origin.registry, origin.slug].filter(Boolean)
    let label = pieces.join(" / ")
    if (origin.installed_version) {
      label += ` @ ${origin.installed_version}`
    }
    return label || "registry"
  }

  return origin.kind || skill.source
}

export function LearnedPage() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["skills", "learned"],
    queryFn: getLearnedSkills,
  })
  const skills = data?.skills ?? []

  return (
    <div className="flex h-full flex-col">
      <PageHeader title="Learned" />

      <div className="flex-1 overflow-auto px-6 py-3">
        <div className="w-full max-w-6xl space-y-6">
          {isLoading ? (
            <div className="text-muted-foreground flex items-center gap-2 py-6 text-sm">
              <IconLoader2 className="size-4 animate-spin" />
              Loading learned skills...
            </div>
          ) : error ? (
            <div className="text-destructive py-6 text-sm">
              Failed to load learned skills.
            </div>
          ) : skills.length ? (
            <div className="grid gap-4 lg:grid-cols-2">
              {skills.map((skill) => (
                <Card
                  key={`${skill.source}:${skill.name}`}
                  className="border-border/60 gap-4 bg-white/80"
                  size="sm"
                >
                  <CardHeader>
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <CardTitle className="font-semibold">
                          {skill.name}
                        </CardTitle>
                        <CardDescription className="mt-3">
                          {skill.description || "No description available."}
                        </CardDescription>
                      </div>
                      <span className="bg-muted text-muted-foreground rounded-full px-2 py-1 text-[11px] font-medium uppercase tracking-[0.18em]">
                        {skill.source}
                      </span>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="space-y-1">
                      <div className="text-muted-foreground text-[11px] tracking-[0.18em] uppercase">
                        Origin
                      </div>
                      <div className="text-sm font-medium">{formatOrigin(skill)}</div>
                    </div>

                    <div className="space-y-1">
                      <div className="text-muted-foreground text-[11px] tracking-[0.18em] uppercase">
                        Path
                      </div>
                      <div className="bg-muted/60 overflow-x-auto rounded-lg px-3 py-2 font-mono text-xs leading-relaxed">
                        {skill.path}
                      </div>
                    </div>

                    {skill.command_examples.length > 0 ? (
                      <div className="space-y-2">
                        <div className="flex items-center gap-2 text-[11px] tracking-[0.18em] uppercase text-emerald-700 dark:text-emerald-300">
                          <IconCommand className="size-3.5" />
                          Commands
                        </div>
                        <div className="space-y-2">
                          {skill.command_examples.map((command) => (
                            <div
                              key={command}
                              className="bg-emerald-500/10 overflow-x-auto rounded-lg border border-emerald-500/20 px-3 py-2 font-mono text-xs leading-relaxed"
                            >
                              {command}
                            </div>
                          ))}
                        </div>
                      </div>
                    ) : null}

                    <div className="space-y-2">
                      <div className="flex items-center gap-2 text-[11px] tracking-[0.18em] uppercase text-slate-600 dark:text-slate-300">
                        <IconBook2 className="size-3.5" />
                        Skill Content
                      </div>
                      <pre className="bg-muted/60 max-h-56 overflow-auto rounded-lg px-3 py-3 text-xs leading-relaxed whitespace-pre-wrap">
                        {skill.content}
                      </pre>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : (
            <Card className="border-dashed">
              <CardContent className="text-muted-foreground py-10 text-center text-sm">
                No learned skills found.
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  )
}

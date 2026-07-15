import {
  IconActivityHeartbeat,
  IconBook2,
  IconBrain,
  IconMessageCircle,
  IconRobot,
  IconRoute,
  IconSearch,
  IconSettings,
  IconShieldCheck,
  IconSparkles,
  IconX,
} from "@tabler/icons-react"
import { useMemo, useState, type ComponentType } from "react"

import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

interface GuideSection {
  title: string
  description: string
  keywords: string[]
  steps: string[]
  icon: ComponentType<{ className?: string }>
}

const guideSections: GuideSection[] = [
  {
    title: "Set up the agent",
    description:
      "Before chatting, configure a model and start the gateway that connects the web console to JameClaw.",
    keywords: ["setup", "model", "gateway", "credentials", "connection"],
    steps: [
      "Open Models and add a provider or local model. Configure its authentication method and select it as the default model.",
      "Use Credentials when a provider requires an API key, OAuth sign-in, or another login method.",
      "Use the Start button in the top bar to start the gateway. The status should show that the service is running before you send messages.",
      "When a configuration page asks for a restart, use the restart action in the top bar so the gateway loads the new settings.",
    ],
    icon: IconSettings,
  },
  {
    title: "Start and manage chats",
    description:
      "Chat is the main workspace for tasks, follow-up questions, files, and ongoing conversations with the agent.",
    keywords: ["chat", "conversation", "new chat", "history", "session", "file"],
    steps: [
      "Select a model beside the Chat title if more than one configured model is available.",
      "Write the outcome you want first, then give the agent context, source material, constraints, and the format you expect.",
      "Attach files when the task depends on their contents. You can also use @ in the composer to call local files, tools, skills, or automations.",
      "Use the history menu to return to a previous session. Use New chat when a task needs a separate context.",
      "Keep related corrections in the same conversation so the agent can use the previous result and your feedback.",
    ],
    icon: IconMessageCircle,
  },
  {
    title: "Write effective requests",
    description:
      "The agent works best when the request identifies the desired result, the available information, and the decision boundaries.",
    keywords: ["prompt", "request", "task", "instructions", "writing", "answer"],
    steps: [
      "Start with an action and result: create a plan, fix an error, summarize a document, compare options, or prepare a response.",
      "Name the target audience and output format, such as a short email, a table, source code, or a numbered implementation plan.",
      "Add constraints that matter: due date, budget, required tools, files to use, things to avoid, or approval requirements.",
      "For larger work, ask the agent to state assumptions and produce the first useful piece before continuing.",
      "Review important outputs. Ask the agent to check its work, explain changes, or revise a specific section rather than starting over.",
    ],
    icon: IconSparkles,
  },
  {
    title: "Manage main agents and subagents",
    description:
      "Agents can have separate workspaces, instructions, tools, and memories so responsibilities stay focused.",
    keywords: ["agent", "subagent", "main agent", "workspace", "role", "tools"],
    steps: [
      "Open Agents to see the main agent and every subagent available in the current JameClaw configuration.",
      "Select an agent to review its role, workspace, model configuration, and other available settings.",
      "Create a focused subagent for a repeatable responsibility, such as research, support, development, or operations.",
      "Give each subagent clear boundaries: what it owns, what it may change, what it should report, and when it should ask for help.",
      "Avoid giving one agent unrelated roles. Separate responsibilities make instructions and stored memory easier to maintain.",
    ],
    icon: IconRobot,
  },
  {
    title: "Use agent memory",
    description:
      "Memory provides useful continuity between tasks. Each agent has its own long-term memory and recent daily notes.",
    keywords: ["memory", "view memory", "daily notes", "long-term", "context", "workspace"],
    steps: [
      "From Agents, choose an agent and press View memory. It opens a complete memory page in a new browser tab.",
      "Long-term memory is best for stable preferences, project facts, active decisions, and instructions that should persist.",
      "Daily notes are better for short-lived work, progress updates, temporary findings, and the current task state.",
      "Review memory when an answer seems outdated or inconsistent. Correct or remove information that should no longer guide the agent.",
      "Keep sensitive information out of general memory unless it is necessary and you are comfortable retaining it locally.",
    ],
    icon: IconBrain,
  },
  {
    title: "Connect channels and automations",
    description:
      "Channels let people reach the agent outside the web console. Automations run recurring work on a schedule.",
    keywords: ["telegram", "channel", "automation", "schedule", "cron", "message", "integration"],
    steps: [
      "Open the relevant channel from the sidebar, enter its credentials, and save the configuration.",
      "Restart the gateway when prompted after a channel change. Check the channel status before relying on it.",
      "Use Automations to schedule recurring work. Give every automation a clear name, schedule, instruction, and expected result.",
      "Test a new channel or automation with a small task before using it for important production work.",
      "Use Logs to inspect connection failures, rejected credentials, delivery errors, or automation failures.",
    ],
    icon: IconRoute,
  },
  {
    title: "Diagnose and fix problems",
    description:
      "Aspirine checks the launcher and gateway for common failures and can offer safe recovery actions.",
    keywords: ["aspirine", "telegram not responding", "error", "logs", "restart", "troubleshooting", "fix"],
    steps: [
      "Open Aspirine when the gateway is stopped, Telegram is not responding, or another connected service appears unavailable.",
      "Read the detected issue and the suggested resolution before applying a change.",
      "Use the available action to start or restart the gateway when Aspirine identifies that as the safe recovery step.",
      "Open Logs for the detailed event history if the issue remains after a recovery action.",
      "Check configuration, credentials, and the relevant channel page when a problem is limited to one integration.",
    ],
    icon: IconActivityHeartbeat,
  },
  {
    title: "Keep control of changes",
    description:
      "Use the agent as a collaborator: inspect important actions and keep sensitive integrations under your control.",
    keywords: ["security", "safety", "approval", "review", "permissions", "sensitive", "control"],
    steps: [
      "Review generated messages, external actions, and configuration changes before relying on them.",
      "Use clear approval boundaries when asking an agent to modify files, send communications, or change a live integration.",
      "Store API keys only in the credentials and configuration areas intended for them, not in chat messages or general memory.",
      "Use separate agents and channels when different teams or projects need different access and context.",
    ],
    icon: IconShieldCheck,
  },
]

function sectionMatches(section: GuideSection, query: string) {
  const searchableText = [
    section.title,
    section.description,
    ...section.keywords,
    ...section.steps,
  ]
    .join(" ")
    .toLowerCase()

  return query
    .toLowerCase()
    .split(/\s+/)
    .filter(Boolean)
    .every((term) => searchableText.includes(term))
}

export function DocsPage() {
  const [query, setQuery] = useState("")
  const normalizedQuery = query.trim()
  const visibleSections = useMemo(
    () =>
      normalizedQuery
        ? guideSections.filter((section) => sectionMatches(section, normalizedQuery))
        : guideSections,
    [normalizedQuery],
  )

  return (
    <div className="flex h-full flex-col bg-background">
      <PageHeader title="Agent documentation" />

      <div className="min-h-0 flex-1 overflow-y-auto px-4 pb-10 sm:px-6">
        <div className="mx-auto w-full max-w-4xl py-6">
          <div className="border-b pb-6">
            <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
              <IconBook2 className="size-4" />
              JameClaw guide
            </div>
            <h1 className="mt-2 text-2xl font-semibold text-foreground">
              Learn how to work with your agent
            </h1>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">
              Find setup instructions, everyday chat workflows, agent and memory
              guidance, channel configuration, automations, and troubleshooting.
            </p>

            <div className="relative mt-5 max-w-xl">
              <IconSearch className="pointer-events-none absolute top-2.5 left-3 size-4 text-muted-foreground" />
              <Input
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Search documentation"
                aria-label="Search documentation"
                className="pr-10 pl-9"
              />
              {query && (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  className="absolute top-0.5 right-0.5 size-8"
                  onClick={() => setQuery("")}
                  aria-label="Clear documentation search"
                >
                  <IconX className="size-4" />
                </Button>
              )}
            </div>
            <p className="mt-2 text-xs text-muted-foreground">
              {normalizedQuery
                ? `${visibleSections.length} matching section${visibleSections.length === 1 ? "" : "s"}`
                : `${guideSections.length} documentation sections`}
            </p>
          </div>

          {visibleSections.length > 0 ? (
            <div className="divide-y">
              {visibleSections.map((section) => {
                const Icon = section.icon
                const sectionNumber = guideSections.indexOf(section) + 1

                return (
                  <section
                    key={section.title}
                    className="grid gap-4 py-6 sm:grid-cols-[2rem_1fr]"
                  >
                    <div className="flex size-8 items-center justify-center rounded-md bg-muted text-muted-foreground">
                      <Icon className="size-4" />
                    </div>
                    <div>
                      <div className="flex items-baseline gap-3">
                        <span className="text-xs font-medium text-muted-foreground">
                          {String(sectionNumber).padStart(2, "0")}
                        </span>
                        <h2 className="text-base font-semibold text-foreground">
                          {section.title}
                        </h2>
                      </div>
                      <p className="mt-2 text-sm leading-6 text-muted-foreground">
                        {section.description}
                      </p>
                      <ol className="mt-3 space-y-2 text-sm leading-6 text-foreground/90">
                        {section.steps.map((step) => (
                          <li key={step} className="flex gap-3">
                            <span className="text-muted-foreground">-</span>
                            <span>{step}</span>
                          </li>
                        ))}
                      </ol>
                    </div>
                  </section>
                )
              })}
            </div>
          ) : (
            <div className="py-12 text-center">
              <IconSearch className="mx-auto size-5 text-muted-foreground" />
              <p className="mt-3 text-sm font-medium text-foreground">No matching documentation</p>
              <p className="mt-1 text-sm text-muted-foreground">
                Try a shorter phrase such as "Telegram", "memory", or "gateway".
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

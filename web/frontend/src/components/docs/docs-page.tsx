import {
  IconActivityHeartbeat,
  IconBook2,
  IconBrain,
  IconMessageCircle,
  IconRobot,
  IconRoute,
  IconSparkles,
} from "@tabler/icons-react"

import { PageHeader } from "@/components/page-header"

interface GuideSection {
  title: string
  description: string
  steps: string[]
  icon: typeof IconMessageCircle
}

const guideSections: GuideSection[] = [
  {
    title: "Start a conversation",
    description: "Use Chat for direct work with your main agent.",
    steps: [
      "Check that the gateway is running from the top bar.",
      "Choose a configured model beside the Chat title.",
      "Describe the outcome you want, then add relevant details, files, or constraints.",
      "Use New chat when you want a separate task and clean context.",
    ],
    icon: IconMessageCircle,
  },
  {
    title: "Give the agent useful tasks",
    description: "Clear requests produce more useful results and fewer follow-up questions.",
    steps: [
      "State the result first: for example, create, fix, compare, summarize, or plan.",
      "Include the important source, deadline, format, and constraints.",
      "Attach files when the answer depends on their contents.",
      "Review the response, then ask for a correction or the next action in the same chat.",
    ],
    icon: IconSparkles,
  },
  {
    title: "Work with agents",
    description: "Agents can use separate workspaces, instructions, tools, and memory.",
    steps: [
      "Open Agents to inspect the main agent and every subagent.",
      "Select an agent to review its role and workspace.",
      "Use View memory to open that agent's stored long-term and daily notes in a new browser tab.",
      "Create focused subagents for repeatable responsibilities rather than mixing unrelated work in one agent.",
    ],
    icon: IconRobot,
  },
  {
    title: "Use memory intentionally",
    description: "Memory helps an agent carry useful context between tasks.",
    steps: [
      "Keep stable preferences, project facts, and decisions in long-term memory.",
      "Use daily notes for temporary progress and current task context.",
      "Review memory from Agents when an answer seems to rely on outdated context.",
      "Remove or correct information that should no longer guide the agent.",
    ],
    icon: IconBrain,
  },
  {
    title: "Connect channels and automation",
    description: "Channels let users reach the agent outside the web console; automations run scheduled work.",
    steps: [
      "Configure a channel such as Telegram from the Channels section.",
      "Start or restart the gateway after configuration changes when requested.",
      "Use Automations for recurring tasks with a clear schedule and expected result.",
      "Check Logs when a channel or automation does not behave as expected.",
    ],
    icon: IconRoute,
  },
  {
    title: "Resolve system problems",
    description: "Aspirine continuously checks common launcher and gateway problems.",
    steps: [
      "Open Aspirine when a channel is not responding or the gateway is unavailable.",
      "Read the detected issue and its suggested resolution.",
      "Use the available action to start or restart the gateway when it is safe to do so.",
      "Return to Logs for the detailed error history if the issue remains.",
    ],
    icon: IconActivityHeartbeat,
  },
]

export function DocsPage() {
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
              Use your agent with confidence
            </h1>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">
              This guide explains the everyday workflow for working with JameClaw,
              managing agents, and keeping connected services reliable.
            </p>
          </div>

          <div className="divide-y">
            {guideSections.map((section, index) => {
              const Icon = section.icon

              return (
                <section key={section.title} className="grid gap-4 py-6 sm:grid-cols-[2rem_1fr]">
                  <div className="flex size-8 items-center justify-center rounded-md bg-muted text-muted-foreground">
                    <Icon className="size-4" />
                  </div>
                  <div>
                    <div className="flex items-baseline gap-3">
                      <span className="text-xs font-medium text-muted-foreground">
                        {String(index + 1).padStart(2, "0")}
                      </span>
                      <h2 className="text-base font-semibold text-foreground">{section.title}</h2>
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
        </div>
      </div>
    </div>
  )
}

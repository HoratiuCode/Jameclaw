import {
  IconBrain,
  IconCheck,
  IconChevronDown,
  IconCircle,
  IconCircleCheck,
  IconCopy,
  IconLoader2,
} from "@tabler/icons-react"
import { useEffect, useState } from "react"
import ReactMarkdown from "react-markdown"
import rehypeRaw from "rehype-raw"
import rehypeSanitize from "rehype-sanitize"
import remarkGfm from "remark-gfm"

import { Button } from "@/components/ui/button"
import { isActivityOnlyContent } from "@/features/chat/activity"
import { formatMessageTime } from "@/hooks/use-jame-chat"
import type { ChatActivity, ChatMediaAttachment } from "@/store/chat"

interface AssistantMessageProps {
  content: string
  agentName?: string
  timestamp?: string | number
  isTyping?: boolean
  media?: ChatMediaAttachment[]
  activity?: ChatActivity[]
}

export function AssistantMessage({
  content,
  agentName = "JameClaw",
  timestamp = "",
  isTyping = false,
  media = [],
  activity = [],
}: AssistantMessageProps) {
  const [isCopied, setIsCopied] = useState(false)
  const formattedTimestamp =
    timestamp !== "" ? formatMessageTime(timestamp) : ""
  const displayContent = isActivityOnlyContent(content) ? "" : content

  const handleCopy = () => {
    navigator.clipboard.writeText(content).then(() => {
      setIsCopied(true)
      setTimeout(() => setIsCopied(false), 2000)
    })
  }

  const previewHref = (href: string) => localPreviewHref(href)

  return (
    <div className="group flex w-full flex-col gap-1.5">
      <div className="text-muted-foreground flex items-center justify-between gap-2 px-1 text-xs opacity-70">
        <div className="flex items-center gap-2">
          <span>{agentName}</span>
          {formattedTimestamp && (
            <>
              <span className="opacity-50">•</span>
              <span>{formattedTimestamp}</span>
            </>
          )}
        </div>
      </div>

      <div className="bg-card text-card-foreground relative overflow-hidden rounded-xl border">
        {activity.length > 0 && (
          <ActivityTimeline activities={activity} isActive={isTyping} />
        )}
        <div className="prose dark:prose-invert prose-p:my-2 prose-pre:my-2 prose-pre:rounded-lg prose-pre:border prose-pre:bg-zinc-950 prose-pre:p-3 max-w-none p-4 text-[15px] leading-relaxed">
          {displayContent.trim() && (
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              rehypePlugins={[rehypeRaw, rehypeSanitize]}
              components={{
				a(props) {
					const { href = "", children, ...rest } = props
					const resolvedHref = previewHref(href)
					const isLocalPreview = resolvedHref !== href
					return (
						<a
							href={resolvedHref}
							target={isLocalPreview ? "_blank" : undefined}
							rel={isLocalPreview ? "noreferrer" : undefined}
							{...rest}
						>
							{children}
						</a>
					)
				},
                code(props) {
                  const { children, className, ...rest } = props
                  const text = String(children).trim()
                  const match = text.match(
                    /^([\uD800-\uDBFF][\uDCC0-\uDFFF]|[\u2600-\u27BF]|\uD83C[\uDF00-\uDFFF]|\uD83D[\uDC00-\uDFFF]|\uD83E[\uDD00-\uDFFF]|[\u2600-\u2B55]|💻|🔍|📝|📖|📁|🔧)\s+(.+)$/,
                  )
                  const isBlock = className && className.startsWith("language-")

                  if (match && !isBlock) {
                    const emoji = match[1]
                    const label = match[2]
                    const cleanContent = content.trim()
                    const cleanText = text.trim()
                    const isRunning =
                      isTyping &&
                      (cleanContent.endsWith("`" + cleanText + "`") ||
                        cleanContent.endsWith(cleanText))

                    return (
                      <span
                        className={`my-0.5 inline-flex items-center gap-1.5 rounded-md border px-2.5 py-0.5 align-middle text-[13px] font-medium shadow-xs transition-all select-none ${
                          isRunning
                            ? "animate-pulse border-amber-200/60 bg-amber-50/40 text-amber-800 dark:border-amber-800/40 dark:bg-amber-950/20 dark:text-amber-300"
                            : "border-zinc-200/80 bg-zinc-50/80 text-zinc-700 dark:border-zinc-800/80 dark:bg-zinc-900/60 dark:text-zinc-300"
                        }`}
                      >
                        <span className="text-sm">{emoji}</span>
                        <span className="max-w-xs truncate font-mono text-xs">
                          {label}
                        </span>
                        {isRunning ? (
                          <IconLoader2 className="h-3.5 w-3.5 animate-spin text-amber-500 dark:text-amber-400" />
                        ) : (
                          <IconCircleCheck className="h-3.5 w-3.5 text-emerald-500 dark:text-emerald-400" />
                        )}
                      </span>
                    )
                  }

                  return (
                    <code className={className} {...rest}>
                      {children}
                    </code>
                  )
                },
              }}
            >
              {displayContent}
            </ReactMarkdown>
          )}
          {media.length > 0 && (
            <div className="not-prose mt-3 flex flex-col gap-3">
              {media.map((item) => (
                <MediaAttachment
                  key={`${item.filename}-${item.url.slice(0, 32)}`}
                  item={item}
                />
              ))}
            </div>
          )}
        </div>
        <Button
          variant="ghost"
          size="icon"
          className="bg-background/50 hover:bg-background/80 absolute top-2 right-2 h-7 w-7 opacity-0 transition-opacity group-hover:opacity-100"
          onClick={handleCopy}
        >
          {isCopied ? (
            <IconCheck className="h-4 w-4 text-green-500" />
          ) : (
            <IconCopy className="text-muted-foreground h-4 w-4" />
          )}
        </Button>
      </div>
    </div>
  )
}

function localPreviewHref(href: string): string {
	try {
		const url = new URL(href)
		const loopbackHosts = new Set(["localhost", "127.0.0.1", "[::1]"])
		if (
			(url.protocol !== "http:" && url.protocol !== "https:") ||
			!loopbackHosts.has(url.hostname) ||
			!url.port
		) {
			return href
		}
		const port = Number(url.port)
		if (!Number.isInteger(port) || port < 1024 || port > 65535) {
			return href
		}
		return `/_preview/${port}${url.pathname}${url.search}${url.hash}`
	} catch {
		return href
	}
}

function ActivityTimeline({
  activities,
  isActive,
}: {
  activities: ChatActivity[]
  isActive: boolean
}) {
  const [isOpen, setIsOpen] = useState(isActive)

  useEffect(() => {
    setIsOpen(isActive)
  }, [isActive])

  const latest = activities.at(-1)

  return (
    <div className="not-prose border-b bg-[var(--jame-accent-soft)] px-4 py-3">
      <button
        type="button"
        onClick={() => setIsOpen((open) => !open)}
        className="flex w-full items-center gap-2 text-left text-xs font-medium text-[var(--jame-accent)]"
      >
        <IconBrain className="size-4" />
        <span>
          {isActive ? "Working live" : "How JameClaw approached this"}
        </span>
        {isActive && (
          <span className="text-[var(--jame-accent-muted)]">
            · live updates
          </span>
        )}
        <IconChevronDown
          className={`ml-auto size-4 transition-transform ${isOpen ? "rotate-180" : ""}`}
        />
      </button>

      {isOpen && (
        <ol className="mt-3 space-y-2 border-l border-[var(--jame-accent-muted)] pl-3">
          {activities.map((activity, index) => {
            const current = isActive && index === activities.length - 1
            return (
              <li
                key={activity.id}
                className="text-muted-foreground relative flex items-center gap-2 text-xs"
              >
                {current ? (
                  <IconLoader2 className="absolute -left-[19px] size-3.5 animate-spin text-[var(--jame-accent)]" />
                ) : (
                  <IconCircle className="absolute -left-[18px] size-3 text-[var(--jame-accent-muted)]" />
                )}
                <span className={current ? "text-foreground font-medium" : ""}>
                  {activity.label}
                </span>
                {current && latest?.kind === "tool" && (
                  <span className="text-[var(--jame-accent)]">in progress</span>
                )}
              </li>
            )
          })}
        </ol>
      )}
    </div>
  )
}

function MediaAttachment({ item }: { item: ChatMediaAttachment }) {
  if (item.kind === "image") {
    return (
      <a href={item.url} download={item.filename} className="block">
        <img
          src={item.url}
          alt={item.filename}
          className="max-h-[70vh] max-w-full rounded-md border object-contain"
        />
      </a>
    )
  }

  if (item.kind === "audio") {
    return <audio controls src={item.url} className="w-full" />
  }

  if (item.kind === "video") {
    return (
      <video
        controls
        src={item.url}
        className="max-h-[70vh] max-w-full rounded-md border"
      />
    )
  }

  return (
    <a
      href={item.url}
      download={item.filename}
      className="inline-flex w-fit items-center rounded-md border px-3 py-2 text-sm font-medium"
    >
      {item.filename}
    </a>
  )
}

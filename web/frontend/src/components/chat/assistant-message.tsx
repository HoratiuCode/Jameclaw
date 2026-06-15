import { IconCheck, IconCopy, IconCircleCheck, IconLoader2 } from "@tabler/icons-react"
import { useState } from "react"
import ReactMarkdown from "react-markdown"
import rehypeRaw from "rehype-raw"
import rehypeSanitize from "rehype-sanitize"
import remarkGfm from "remark-gfm"

import { Button } from "@/components/ui/button"
import { formatMessageTime } from "@/hooks/use-jame-chat"

interface AssistantMessageProps {
  content: string
  timestamp?: string | number
  isTyping?: boolean
}

export function AssistantMessage({
  content,
  timestamp = "",
  isTyping = false,
}: AssistantMessageProps) {
  const [isCopied, setIsCopied] = useState(false)
  const formattedTimestamp =
    timestamp !== "" ? formatMessageTime(timestamp) : ""

  const handleCopy = () => {
    navigator.clipboard.writeText(content).then(() => {
      setIsCopied(true)
      setTimeout(() => setIsCopied(false), 2000)
    })
  }

  return (
    <div className="group flex w-full flex-col gap-1.5">
      <div className="text-muted-foreground flex items-center justify-between gap-2 px-1 text-xs opacity-70">
        <div className="flex items-center gap-2">
          <span>JameClaw</span>
          {formattedTimestamp && (
            <>
              <span className="opacity-50">•</span>
              <span>{formattedTimestamp}</span>
            </>
          )}
        </div>
      </div>

      <div className="bg-card text-card-foreground relative overflow-hidden rounded-xl border">
        <div className="prose dark:prose-invert prose-p:my-2 prose-pre:my-2 prose-pre:rounded-lg prose-pre:border prose-pre:bg-zinc-950 prose-pre:p-3 max-w-none p-4 text-[15px] leading-relaxed">
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            rehypePlugins={[rehypeRaw, rehypeSanitize]}
            components={{
              code(props) {
                const { children, className, ...rest } = props
                const text = String(children).trim()
                const match = text.match(/^([\uD800-\uDBFF][\uDCC0-\uDFFF]|[\u2600-\u27BF]|\uD83C[\uDF00-\uDFFF]|\uD83D[\uDC00-\uDFFF]|\uD83E[\uDD00-\uDFFF]|[\u2600-\u2B55]|💻|🔍|📝|📖|📁|🔧)\s+(.+)$/)
                const isBlock = className && className.startsWith('language-')

                if (match && !isBlock) {
                  const emoji = match[1]
                  const label = match[2]
                  const cleanContent = content.trim()
                  const cleanText = text.trim()
                  const isRunning = isTyping && (
                    cleanContent.endsWith('`' + cleanText + '`') ||
                    cleanContent.endsWith(cleanText)
                  )

                  return (
                    <span className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 my-0.5 rounded-md border text-[13px] font-medium transition-all shadow-xs select-none align-middle ${
                      isRunning 
                        ? "bg-amber-50/40 dark:bg-amber-950/20 border-amber-200/60 dark:border-amber-800/40 text-amber-800 dark:text-amber-300 animate-pulse" 
                        : "bg-zinc-50/80 dark:bg-zinc-900/60 border-zinc-200/80 dark:border-zinc-800/80 text-zinc-700 dark:text-zinc-300"
                    }`}>
                      <span className="text-sm">{emoji}</span>
                      <span className="font-mono text-xs max-w-xs truncate">{label}</span>
                      {isRunning ? (
                        <IconLoader2 className="h-3.5 w-3.5 animate-spin text-amber-500 dark:text-amber-400" />
                      ) : (
                        <IconCircleCheck className="h-3.5 w-3.5 text-emerald-500 dark:text-emerald-400" />
                      )}
                    </span>
                  )
                }

                return <code className={className} {...rest}>{children}</code>
              }
            }}
          >
            {content}
          </ReactMarkdown>
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

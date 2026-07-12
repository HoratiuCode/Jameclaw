import {
  IconArrowUp,
  IconFile,
  IconFolder,
  IconMicrophone,
  IconPaperclip,
  IconPlayerStopFilled,
} from "@tabler/icons-react"
import type { KeyboardEvent } from "react"
import { useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import TextareaAutosize from "react-textarea-autosize"

import { type LocalFileSearchItem, searchLocalFiles } from "@/api/files"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

interface ChatComposerProps {
  input: string
  onInputChange: (value: string) => void
  onSend: () => void
  onFileSelect?: (files: FileList) => void
  onVoiceToggle?: () => void
  disabledReason?: string | null
  isConnected: boolean
  hasDefaultModel: boolean
  isRecording?: boolean
  canRecord?: boolean
  isUploading?: boolean
}

export function ChatComposer({
  input,
  onInputChange,
  onSend,
  onFileSelect,
  onVoiceToggle,
  disabledReason,
  isConnected,
  hasDefaultModel,
  isRecording = false,
  canRecord = true,
  isUploading = false,
}: ChatComposerProps) {
  const { t } = useTranslation()
  const canInput = isConnected && hasDefaultModel
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)
  const [caretPosition, setCaretPosition] = useState(0)
  const [fileMatches, setFileMatches] = useState<LocalFileSearchItem[]>([])
  const [isFileMenuOpen, setIsFileMenuOpen] = useState(false)
  const [selectedFileIndex, setSelectedFileIndex] = useState(0)
  const [fileSearchError, setFileSearchError] = useState(false)

  const activeMention = useMemo(() => {
    const beforeCaret = input.slice(0, caretPosition)
    const match = /(^|\s)@([^\s@"]*)$/.exec(beforeCaret)
    if (!match) return null
    return {
      start: beforeCaret.length - match[2].length - 1,
      end: caretPosition,
      query: match[2],
    }
  }, [caretPosition, input])

  useEffect(() => {
    if (!canInput || !activeMention) {
      setIsFileMenuOpen(false)
      setFileMatches([])
      setSelectedFileIndex(0)
      setFileSearchError(false)
      return
    }

    let cancelled = false
    const timeoutId = window.setTimeout(() => {
      searchLocalFiles(activeMention.query)
        .then((items) => {
          if (cancelled) return
          setFileMatches(items)
          setIsFileMenuOpen(true)
          setSelectedFileIndex(0)
          setFileSearchError(false)
        })
        .catch(() => {
          if (cancelled) return
          setFileMatches([])
          setIsFileMenuOpen(true)
          setSelectedFileIndex(0)
          setFileSearchError(true)
        })
    }, 120)

    return () => {
      cancelled = true
      window.clearTimeout(timeoutId)
    }
  }, [activeMention, canInput])

  const syncCaretPosition = () => {
    setCaretPosition(textareaRef.current?.selectionStart ?? input.length)
  }

  const insertFileMention = (file: LocalFileSearchItem) => {
    if (!activeMention) return
    const mention = `@"${file.path.replaceAll('"', '\\"')}" `
    const nextInput =
      input.slice(0, activeMention.start) + mention + input.slice(activeMention.end)
    const nextCaret = activeMention.start + mention.length
    onInputChange(nextInput)
    setIsFileMenuOpen(false)
    setFileMatches([])
    setSelectedFileIndex(0)
    window.requestAnimationFrame(() => {
      textareaRef.current?.focus()
      textareaRef.current?.setSelectionRange(nextCaret, nextCaret)
      setCaretPosition(nextCaret)
    })
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.nativeEvent.isComposing) return
    if (isFileMenuOpen && activeMention) {
      if (e.key === "ArrowDown") {
        e.preventDefault()
        setSelectedFileIndex((index) =>
          fileMatches.length === 0 ? 0 : (index + 1) % fileMatches.length,
        )
        return
      }
      if (e.key === "ArrowUp") {
        e.preventDefault()
        setSelectedFileIndex((index) =>
          fileMatches.length === 0
            ? 0
            : (index - 1 + fileMatches.length) % fileMatches.length,
        )
        return
      }
      if (e.key === "Escape") {
        e.preventDefault()
        setIsFileMenuOpen(false)
        return
      }
      if (e.key === "Enter" && fileMatches[selectedFileIndex]) {
        e.preventDefault()
        insertFileMention(fileMatches[selectedFileIndex])
        return
      }
    }
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      onSend()
    }
  }

  return (
    <div className="bg-background shrink-0 px-4 pt-4 pb-[calc(1rem+env(safe-area-inset-bottom))] md:px-8 md:pb-8 lg:px-24 xl:px-48">
      <div className="bg-card border-border/80 relative mx-auto flex max-w-[1000px] flex-col rounded-2xl border p-3 shadow-md">
        {isFileMenuOpen && activeMention && (
          <div className="bg-popover border-border absolute right-3 bottom-[calc(100%+0.5rem)] left-3 z-20 overflow-hidden rounded-lg border shadow-lg">
            <div className="border-border bg-muted/50 flex items-center justify-between border-b px-3 py-2 text-xs">
              <span className="text-muted-foreground">Local files</span>
              <span className="text-muted-foreground font-mono">
                @{activeMention.query}
              </span>
            </div>
            {fileSearchError ? (
              <div className="text-muted-foreground px-3 py-3 text-sm">
                Could not search files.
              </div>
            ) : fileMatches.length === 0 ? (
              <div className="text-muted-foreground px-3 py-3 text-sm">
                No matching files found.
              </div>
            ) : (
              <div className="max-h-72 overflow-y-auto py-1">
                {fileMatches.map((file, index) => {
                  const Icon = file.kind === "folder" ? IconFolder : IconFile
                  return (
                    <button
                      key={file.path}
                      type="button"
                      className={cn(
                        "flex w-full items-center gap-2 px-3 py-2 text-left text-sm",
                        index === selectedFileIndex
                          ? "bg-accent text-accent-foreground"
                          : "hover:bg-accent/70",
                      )}
                      onMouseDown={(event) => {
                        event.preventDefault()
                        insertFileMention(file)
                      }}
                    >
                      <Icon className="text-muted-foreground size-4 shrink-0" />
                      <span className="min-w-0 flex-1">
                        <span className="block truncate font-medium">
                          {file.name}
                        </span>
                        <span className="text-muted-foreground block truncate text-xs">
                          {file.directory}
                        </span>
                      </span>
                    </button>
                  )
                })}
              </div>
            )}
          </div>
        )}
        <TextareaAutosize
          ref={textareaRef}
          value={input}
          onChange={(e) => {
            onInputChange(e.target.value)
            setCaretPosition(e.target.selectionStart)
          }}
          onKeyDown={handleKeyDown}
          onClick={syncCaretPosition}
          onKeyUp={syncCaretPosition}
          onSelect={syncCaretPosition}
          placeholder={t("chat.placeholder")}
          disabled={!canInput}
          className={cn(
            "placeholder:text-muted-foreground max-h-[200px] min-h-[60px] resize-none border-0 bg-transparent px-2 py-1 text-[15px] shadow-none transition-colors focus-visible:ring-0 focus-visible:outline-none dark:bg-transparent",
            !canInput && "cursor-not-allowed",
          )}
          minRows={1}
          maxRows={8}
        />

        <div className="mt-2 flex items-center justify-between px-1">
          <div className="flex min-h-5 items-center gap-1">
            {disabledReason && (
              <p className="text-muted-foreground text-xs">{disabledReason}</p>
            )}
            {!disabledReason && (
              <p className="text-muted-foreground text-xs">
                Try /emoji, /persona, or /skills add &lt;skill&gt; to customize the assistant.
              </p>
            )}
          </div>

          <div className="flex items-center gap-2">
            {onFileSelect && (
              <>
                <input
                  id="chat-file-upload"
                  type="file"
                  multiple
                  className="hidden"
                  onChange={(event) => {
                    if (event.target.files && event.target.files.length > 0) {
                      onFileSelect(event.target.files)
                    }
                    event.target.value = ""
                  }}
                />
                <Button
                  size="icon"
                  variant="secondary"
                  className="size-8 rounded-full transition-transform active:scale-95"
                  disabled={!canInput || isUploading}
                  title="Attach files"
                  asChild
                >
                  <label htmlFor="chat-file-upload" className="cursor-pointer">
                    <IconPaperclip className="size-4" />
                  </label>
                </Button>
              </>
            )}

            {onVoiceToggle && (
              <Button
                size="icon"
                variant={isRecording ? "destructive" : "secondary"}
                className="size-8 rounded-full transition-transform active:scale-95"
                onClick={onVoiceToggle}
                disabled={!canInput || !canRecord}
                title={isRecording ? "Stop recording" : "Record voice message"}
              >
                {isRecording ? (
                  <IconPlayerStopFilled className="size-4" />
                ) : (
                  <IconMicrophone className="size-4" />
                )}
              </Button>
            )}

            <Button
              size="icon"
              className="size-8 rounded-full bg-violet-500 text-white transition-transform hover:bg-violet-600 active:scale-95"
              onClick={onSend}
              disabled={!input.trim() || !canInput}
              title={disabledReason ?? undefined}
            >
              <IconArrowUp className="size-4" />
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}

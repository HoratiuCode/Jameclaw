import {
  IconArrowUp,
  IconCalendarTime,
  IconFile,
  IconFolder,
  IconMicrophone,
  IconPaperclip,
  IconPlayerStopFilled,
  IconSparkles,
  IconTools,
} from "@tabler/icons-react"
import type { KeyboardEvent } from "react"
import { useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import TextareaAutosize from "react-textarea-autosize"

import { type AutomationItem, getAutomations } from "@/api/automation"
import { type LocalFileSearchItem, searchLocalFiles } from "@/api/files"
import { type LearnedSkillItem, getLearnedSkills } from "@/api/skills"
import { type ToolSupportItem, getTools } from "@/api/tools"
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

type MentionItem =
  | {
      id: string
      type: "file"
      title: string
      subtitle: string
      insertText: string
      file: LocalFileSearchItem
    }
  | {
      id: string
      type: "tool"
      title: string
      subtitle: string
      insertText: string
      tool: ToolSupportItem
    }
  | {
      id: string
      type: "skill"
      title: string
      subtitle: string
      insertText: string
      skill: LearnedSkillItem
    }
  | {
      id: string
      type: "automation"
      title: string
      subtitle: string
      insertText: string
      automation: AutomationItem
    }

function quoteMention(value: string) {
  return `"${value.replaceAll('"', '\\"')}"`
}

function matchesMentionQuery(query: string, values: string[]) {
  const normalized = query.trim().toLowerCase()
  if (!normalized) return true
  return values.some((value) => value.toLowerCase().includes(normalized))
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
  const [mentionItems, setMentionItems] = useState<MentionItem[]>([])
  const [isMentionMenuOpen, setIsMentionMenuOpen] = useState(false)
  const [selectedMentionIndex, setSelectedMentionIndex] = useState(0)
  const [mentionSearchError, setMentionSearchError] = useState(false)
  const [skillSlashItems, setSkillSlashItems] = useState<MentionItem[]>([])
  const [isSkillSlashMenuOpen, setIsSkillSlashMenuOpen] = useState(false)
  const [selectedSkillSlashIndex, setSelectedSkillSlashIndex] = useState(0)
  const [skillSlashSearchError, setSkillSlashSearchError] = useState(false)

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

  const activeSkillSlash = useMemo(() => {
    const beforeCaret = input.slice(0, caretPosition)
    const match = /(^|\s)\/([^\s/]*)$/.exec(beforeCaret)
    if (!match) return null
    return {
      start: beforeCaret.length - match[2].length - 1,
      end: caretPosition,
      query: match[2],
    }
  }, [caretPosition, input])

  useEffect(() => {
    if (!canInput || !activeMention) {
      setIsMentionMenuOpen(false)
      setMentionItems([])
      setSelectedMentionIndex(0)
      setMentionSearchError(false)
      return
    }

    let cancelled = false
    const timeoutId = window.setTimeout(() => {
      Promise.allSettled([
        searchLocalFiles(activeMention.query, 8),
        getTools(),
        getAutomations(),
        getLearnedSkills(),
      ])
        .then(
          ([
            filesResult,
            toolsResult,
            automationsResult,
            learnedSkillsResult,
          ]) => {
            if (cancelled) return
            const nextItems: MentionItem[] = []

            if (filesResult.status === "fulfilled") {
              nextItems.push(
                ...filesResult.value.map((file) => ({
                  id: `file:${file.path}`,
                  type: "file" as const,
                  title: file.name,
                  subtitle: file.directory,
                  insertText: `@${quoteMention(file.path)} `,
                  file,
                })),
              )
            }

            if (toolsResult.status === "fulfilled") {
              nextItems.push(
                ...toolsResult.value.tools
                  .filter((tool) => tool.status === "enabled")
                  .filter((tool) =>
                    matchesMentionQuery(activeMention.query, [
                      tool.name,
                      tool.description,
                      tool.category,
                    ]),
                  )
                  .slice(0, 8)
                  .map((tool) => ({
                    id: `tool:${tool.name}`,
                    type: "tool" as const,
                    title: tool.name,
                    subtitle: `${tool.category} tool - ${tool.description}`,
                    insertText: `@tool:${tool.name} `,
                    tool,
                  })),
              )
            }

            if (automationsResult.status === "fulfilled") {
              nextItems.push(
                ...automationsResult.value
                  .filter((automation) =>
                    matchesMentionQuery(activeMention.query, [
                      automation.name,
                      automation.prompt,
                      automation.status,
                    ]),
                  )
                  .slice(0, 6)
                  .map((automation) => ({
                    id: `automation:${automation.id}`,
                    type: "automation" as const,
                    title: automation.name,
                    subtitle: `${automation.status} - ${automation.schedule}`,
                    insertText: `@automation:${quoteMention(automation.name)} `,
                    automation,
                  })),
              )
            }

            if (learnedSkillsResult.status === "fulfilled") {
              nextItems.push(
                ...learnedSkillsResult.value.skills
                  .filter((skill) =>
                    matchesMentionQuery(activeMention.query, [
                      skill.name,
                      skill.description,
                      skill.source,
                    ]),
                  )
                  .slice(0, 8)
                  .map((skill) => ({
                    id: `skill:${skill.name}`,
                    type: "skill" as const,
                    title: skill.name,
                    subtitle: skill.description || `${skill.source} skill`,
                    insertText: `@skill:${skill.name} `,
                    skill,
                  })),
              )
            }

            setMentionItems(nextItems)
            setIsMentionMenuOpen(true)
            setSelectedMentionIndex(0)
            setMentionSearchError(
              filesResult.status === "rejected" &&
                toolsResult.status === "rejected" &&
                automationsResult.status === "rejected" &&
                learnedSkillsResult.status === "rejected",
            )
          },
        )
        .catch(() => {
          if (cancelled) return
          setMentionItems([])
          setIsMentionMenuOpen(true)
          setSelectedMentionIndex(0)
          setMentionSearchError(true)
        })
    }, 120)

    return () => {
      cancelled = true
      window.clearTimeout(timeoutId)
    }
  }, [activeMention, canInput])

  useEffect(() => {
    if (!canInput || !activeSkillSlash || activeMention) {
      setIsSkillSlashMenuOpen(false)
      setSkillSlashItems([])
      setSelectedSkillSlashIndex(0)
      setSkillSlashSearchError(false)
      return
    }

    let cancelled = false
    const timeoutId = window.setTimeout(() => {
      getLearnedSkills()
        .then(({ skills }) => {
          if (cancelled) return
          setSkillSlashItems(
            skills
              .filter((skill) =>
                matchesMentionQuery(activeSkillSlash.query, [
                  skill.name,
                  skill.description,
                  skill.source,
                ]),
              )
              .slice(0, 8)
              .map((skill) => ({
                id: `skill:${skill.name}`,
                type: "skill" as const,
                title: skill.name,
                subtitle: skill.description || `${skill.source} skill`,
                insertText: `@skill:${skill.name} `,
                skill,
              })),
          )
          setIsSkillSlashMenuOpen(true)
          setSelectedSkillSlashIndex(0)
          setSkillSlashSearchError(false)
        })
        .catch(() => {
          if (cancelled) return
          setSkillSlashItems([])
          setIsSkillSlashMenuOpen(true)
          setSelectedSkillSlashIndex(0)
          setSkillSlashSearchError(true)
        })
    }, 120)

    return () => {
      cancelled = true
      window.clearTimeout(timeoutId)
    }
  }, [activeMention, activeSkillSlash, canInput])

  const syncCaretPosition = () => {
    setCaretPosition(textareaRef.current?.selectionStart ?? input.length)
  }

  const insertMention = (item: MentionItem) => {
    if (!activeMention) return
    const mention = item.insertText
    const nextInput =
      input.slice(0, activeMention.start) +
      mention +
      input.slice(activeMention.end)
    const nextCaret = activeMention.start + mention.length
    onInputChange(nextInput)
    setIsMentionMenuOpen(false)
    setMentionItems([])
    setSelectedMentionIndex(0)
    window.requestAnimationFrame(() => {
      textareaRef.current?.focus()
      textareaRef.current?.setSelectionRange(nextCaret, nextCaret)
      setCaretPosition(nextCaret)
    })
  }

  const insertSkillSlash = (item: MentionItem) => {
    if (!activeSkillSlash) return
    const nextInput =
      input.slice(0, activeSkillSlash.start) +
      item.insertText +
      input.slice(activeSkillSlash.end)
    const nextCaret = activeSkillSlash.start + item.insertText.length
    onInputChange(nextInput)
    setIsSkillSlashMenuOpen(false)
    setSkillSlashItems([])
    setSelectedSkillSlashIndex(0)
    window.requestAnimationFrame(() => {
      textareaRef.current?.focus()
      textareaRef.current?.setSelectionRange(nextCaret, nextCaret)
      setCaretPosition(nextCaret)
    })
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.nativeEvent.isComposing) return
    if (isMentionMenuOpen && activeMention) {
      if (e.key === "ArrowDown") {
        e.preventDefault()
        setSelectedMentionIndex((index) =>
          mentionItems.length === 0 ? 0 : (index + 1) % mentionItems.length,
        )
        return
      }
      if (e.key === "ArrowUp") {
        e.preventDefault()
        setSelectedMentionIndex((index) =>
          mentionItems.length === 0
            ? 0
            : (index - 1 + mentionItems.length) % mentionItems.length,
        )
        return
      }
      if (e.key === "Escape") {
        e.preventDefault()
        setIsMentionMenuOpen(false)
        return
      }
      if (e.key === "Enter" && mentionItems[selectedMentionIndex]) {
        e.preventDefault()
        insertMention(mentionItems[selectedMentionIndex])
        return
      }
    }
    if (isSkillSlashMenuOpen && activeSkillSlash) {
      if (e.key === "ArrowDown") {
        e.preventDefault()
        setSelectedSkillSlashIndex((index) =>
          skillSlashItems.length === 0
            ? 0
            : (index + 1) % skillSlashItems.length,
        )
        return
      }
      if (e.key === "ArrowUp") {
        e.preventDefault()
        setSelectedSkillSlashIndex((index) =>
          skillSlashItems.length === 0
            ? 0
            : (index - 1 + skillSlashItems.length) % skillSlashItems.length,
        )
        return
      }
      if (e.key === "Escape") {
        e.preventDefault()
        setIsSkillSlashMenuOpen(false)
        return
      }
      if (e.key === "Enter" && skillSlashItems[selectedSkillSlashIndex]) {
        e.preventDefault()
        insertSkillSlash(skillSlashItems[selectedSkillSlashIndex])
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
        {isMentionMenuOpen && activeMention && (
          <div className="bg-popover border-border absolute right-3 bottom-[calc(100%+0.5rem)] left-3 z-20 overflow-hidden rounded-lg border shadow-lg">
            <div className="border-border bg-muted/50 flex items-center justify-between border-b px-3 py-2 text-xs">
              <span className="text-muted-foreground">Call with @</span>
              <span className="text-muted-foreground font-mono">
                @{activeMention.query}
              </span>
            </div>
            {mentionSearchError ? (
              <div className="text-muted-foreground px-3 py-3 text-sm">
                Could not load callable items.
              </div>
            ) : mentionItems.length === 0 ? (
              <div className="text-muted-foreground px-3 py-3 text-sm">
                No matching files, tools, skills, or automations found.
              </div>
            ) : (
              <div className="max-h-72 overflow-y-auto py-1">
                {mentionItems.map((item, index) => {
                  const Icon =
                    item.type === "tool"
                      ? IconTools
                      : item.type === "skill"
                        ? IconSparkles
                        : item.type === "automation"
                          ? IconCalendarTime
                          : item.file.kind === "folder"
                            ? IconFolder
                            : IconFile
                  return (
                    <button
                      key={item.id}
                      type="button"
                      className={cn(
                        "flex w-full items-center gap-2 px-3 py-2 text-left text-sm",
                        index === selectedMentionIndex
                          ? "bg-accent text-accent-foreground"
                          : "hover:bg-accent/70",
                      )}
                      onMouseDown={(event) => {
                        event.preventDefault()
                        insertMention(item)
                      }}
                    >
                      <Icon className="text-muted-foreground size-4 shrink-0" />
                      <span className="min-w-0 flex-1">
                        <span className="block truncate font-medium">
                          {item.title}
                        </span>
                        <span className="text-muted-foreground block truncate text-xs">
                          {item.subtitle}
                        </span>
                      </span>
                      <span className="text-muted-foreground shrink-0 rounded border px-1.5 py-0.5 text-[10px] uppercase">
                        {item.type}
                      </span>
                    </button>
                  )
                })}
              </div>
            )}
          </div>
        )}
        {isSkillSlashMenuOpen && activeSkillSlash && (
          <div className="bg-popover border-border absolute right-3 bottom-[calc(100%+0.5rem)] left-3 z-20 overflow-hidden rounded-lg border shadow-lg">
            <div className="border-border bg-muted/50 flex items-center justify-between border-b px-3 py-2 text-xs">
              <span className="text-muted-foreground">Call a skill with /</span>
              <span className="text-muted-foreground font-mono">
                /{activeSkillSlash.query}
              </span>
            </div>
            {skillSlashSearchError ? (
              <div className="text-muted-foreground px-3 py-3 text-sm">
                Could not load skills.
              </div>
            ) : skillSlashItems.length === 0 ? (
              <div className="text-muted-foreground px-3 py-3 text-sm">
                No matching skills found.
              </div>
            ) : (
              <div className="max-h-72 overflow-y-auto py-1">
                {skillSlashItems.map((item, index) => (
                  <button
                    key={item.id}
                    type="button"
                    className={cn(
                      "flex w-full items-center gap-2 px-3 py-2 text-left text-sm",
                      index === selectedSkillSlashIndex
                        ? "bg-accent text-accent-foreground"
                        : "hover:bg-accent/70",
                    )}
                    onMouseDown={(event) => {
                      event.preventDefault()
                      insertSkillSlash(item)
                    }}
                  >
                    <IconSparkles className="text-muted-foreground size-4 shrink-0" />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-medium">
                        {item.title}
                      </span>
                      <span className="text-muted-foreground block truncate text-xs">
                        {item.subtitle}
                      </span>
                    </span>
                    <span className="text-muted-foreground shrink-0 rounded border px-1.5 py-0.5 text-[10px] uppercase">
                      skill
                    </span>
                  </button>
                ))}
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
                Type / to call a skill, or @ for files, tools, skills, and
                automations.
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
              className="size-8 rounded-full bg-[var(--jame-accent)] text-white transition-transform hover:brightness-90 active:scale-95"
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

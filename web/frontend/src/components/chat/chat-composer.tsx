import {
  IconArrowUp,
  IconMicrophone,
  IconPaperclip,
  IconPlayerStopFilled,
} from "@tabler/icons-react"
import type { KeyboardEvent } from "react"
import { useTranslation } from "react-i18next"
import TextareaAutosize from "react-textarea-autosize"

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

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.nativeEvent.isComposing) return
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      onSend()
    }
  }

  return (
    <div className="bg-background shrink-0 px-4 pt-4 pb-[calc(1rem+env(safe-area-inset-bottom))] md:px-8 md:pb-8 lg:px-24 xl:px-48">
      <div className="bg-card border-border/80 mx-auto flex max-w-[1000px] flex-col rounded-2xl border p-3 shadow-md">
        <TextareaAutosize
          value={input}
          onChange={(e) => onInputChange(e.target.value)}
          onKeyDown={handleKeyDown}
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

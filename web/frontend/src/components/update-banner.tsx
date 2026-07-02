import { IconExternalLink, IconLoader2, IconX } from "@tabler/icons-react"

import { Button } from "@/components/ui/button"
import { useUpdateStatus } from "@/hooks/use-update-status"

export function UpdateBanner() {
  const { data, isLoading, dismiss, dismissing, openUpdate, opening } =
    useUpdateStatus()

  if (
    isLoading ||
    !data?.update_available ||
    data.dismissed ||
    !data.latest_version
  ) {
    return null
  }

  const versionLabel = data.latest_name || data.latest_version

  return (
    <div className="bg-primary text-primary-foreground flex min-h-10 shrink-0 items-center justify-between gap-3 px-3 py-1.5 text-sm">
      <div className="min-w-0 truncate">
        <span className="font-semibold">New JameClaw version available:</span>{" "}
        <span className="font-mono text-xs">{versionLabel}</span>
        <span className="hidden opacity-80 sm:inline">
          {" "}
          · current {data.current_version}
        </span>
      </div>
      <div className="flex shrink-0 items-center gap-1.5">
        <Button
          size="xs"
          variant="secondary"
          className="h-7 gap-1.5 bg-white/95 text-black hover:bg-white"
          onClick={() => openUpdate()}
          disabled={opening}
        >
          {opening ? (
            <IconLoader2 className="size-3.5 animate-spin" />
          ) : (
            <IconExternalLink className="size-3.5" />
          )}
          <span>{data.update_action_text || "Update"}</span>
        </Button>
        <Button
          size="icon-xs"
          variant="ghost"
          className="text-primary-foreground hover:bg-white/15 hover:text-primary-foreground"
          onClick={() => dismiss(data.latest_version || "")}
          disabled={dismissing}
          aria-label="Dismiss update notice"
        >
          <IconX className="size-4" />
        </Button>
      </div>
    </div>
  )
}

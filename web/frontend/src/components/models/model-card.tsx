import {
  IconEdit,
  IconKey,
  IconLoader2,
  IconMessageCircle,
  IconMicrophone,
  IconPhoto,
  IconStar,
  IconStarFilled,
  IconTrash,
} from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { ModelInfo, ModelRole } from "@/api/models"
import { Button } from "@/components/ui/button"

interface ModelCardProps {
  model: ModelInfo
  onEdit: (model: ModelInfo) => void
  onSetDefault: (model: ModelInfo, role: ModelRole) => void
  onDelete: (model: ModelInfo) => void
  settingDefaultRole: ModelRole | null
}

export function ModelCard({
  model,
  onEdit,
  onSetDefault,
  onDelete,
  settingDefaultRole,
}: ModelCardProps) {
  const { t } = useTranslation()
  const isOAuth = model.auth_method === "oauth"
  const canSetDefault = model.configured
  const defaultRoles: Array<{ role: ModelRole; label: string }> = [
    model.is_default && { role: "chat", label: t("models.badge.chat") },
    model.is_image_default && { role: "image", label: t("models.badge.image") },
    model.is_voice_default && { role: "voice", label: t("models.badge.voice") },
  ].filter(Boolean) as Array<{ role: ModelRole; label: string }>
  const protectedDefault =
    model.is_default || model.is_image_default || model.is_voice_default
  const roleActions: Array<{
    role: ModelRole
    active: boolean
    title: string
    icon: typeof IconMessageCircle
  }> = [
    {
      role: "chat",
      active: model.is_default,
      title: t("models.action.setChatDefault"),
      icon: IconMessageCircle,
    },
    {
      role: "image",
      active: model.is_image_default,
      title: t("models.action.setImageDefault"),
      icon: IconPhoto,
    },
    {
      role: "voice",
      active: model.is_voice_default,
      title: t("models.action.setVoiceDefault"),
      icon: IconMicrophone,
    },
  ]

  return (
    <div
      className={[
        "group/card hover:bg-muted/30 relative flex w-full max-w-[36rem] flex-col gap-3 justify-self-start rounded-xl border p-4 transition-colors hover:shadow-xs",
        model.configured
          ? "border-border/60 bg-card"
          : "border-border/50 bg-card/60",
      ].join(" ")}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <span
            className={[
              "mt-0.5 h-2 w-2 shrink-0 rounded-full",
              model.is_default
                ? "bg-green-400 shadow-[0_0_0_2px_rgba(74,222,128,0.35)]"
                : model.configured
                  ? "bg-green-500"
                  : "bg-muted-foreground/25",
            ].join(" ")}
            title={
              model.configured
                ? t("models.status.configured")
                : t("models.status.unconfigured")
            }
          />
          <span className="text-foreground truncate text-sm font-semibold">
            {model.model_name}
          </span>
          {defaultRoles.map((role) => (
            <span
              key={role.role}
              className="bg-primary/10 text-primary shrink-0 rounded px-1.5 py-0.5 text-[10px] leading-none font-medium"
            >
              {role.label}
            </span>
          ))}
        </div>

        <div className="flex shrink-0 items-center gap-0.5">
          {roleActions.map(({ role, active, title, icon: Icon }) => (
            <Button
              key={role}
              variant="ghost"
              size="icon-sm"
              onClick={() => onSetDefault(model, role)}
              disabled={settingDefaultRole !== null || !canSetDefault || active}
              title={title}
              className={active ? "text-primary" : undefined}
            >
              {settingDefaultRole === role ? (
                <IconLoader2 className="size-3.5 animate-spin" />
              ) : active && role === "chat" ? (
                <IconStarFilled className="size-3.5" />
              ) : role === "chat" ? (
                <IconStar className="size-3.5" />
              ) : (
                <Icon className="size-3.5" />
              )}
            </Button>
          ))}

          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onEdit(model)}
            title={t("models.action.edit")}
          >
            <IconEdit className="size-3.5" />
          </Button>

          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onDelete(model)}
            disabled={protectedDefault}
            title={t("models.action.delete")}
            className="text-muted-foreground hover:text-destructive hover:bg-destructive/10"
          >
            <IconTrash className="size-3.5" />
          </Button>
        </div>
      </div>

      <p className="text-muted-foreground truncate font-mono text-xs leading-snug">
        {model.model}
      </p>

      <div className="flex items-center gap-2">
        {isOAuth ? (
          <span className="text-muted-foreground bg-muted rounded px-1.5 py-0.5 text-[10px] font-medium">
            OAuth
          </span>
        ) : model.configured && model.api_key ? (
          <span className="text-muted-foreground/70 flex items-center gap-1 font-mono text-[11px]">
            <IconKey className="size-3" />
            {model.api_key}
          </span>
        ) : (
          <span className="text-muted-foreground/50 text-[11px]">
            {t("models.status.unconfigured")}
          </span>
        )}
      </div>
    </div>
  )
}

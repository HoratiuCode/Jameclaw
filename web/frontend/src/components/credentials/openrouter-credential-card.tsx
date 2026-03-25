import {
  IconLoader2,
  IconPlayerStopFilled,
  IconRouteAltLeft,
} from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { OAuthProviderStatus } from "@/api/oauth"
import { maskedSecretPlaceholder } from "@/components/secret-placeholder"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

import { CredentialCard } from "./credential-card"

interface OpenRouterCredentialCardProps {
  status: OAuthProviderStatus["status"]
  activeAction: string
  token: string
  savedTokenMask: string
  modelCount: number
  onTokenChange: (value: string) => void
  onStopLoading: () => void
  onSaveToken: () => void
  onAskLogout: () => void
}

export function OpenRouterCredentialCard({
  status,
  activeAction,
  token,
  savedTokenMask,
  modelCount,
  onTokenChange,
  onStopLoading,
  onSaveToken,
  onAskLogout,
}: OpenRouterCredentialCardProps) {
  const { t } = useTranslation()
  const actionBusy = activeAction !== ""
  const tokenLoading = activeAction === "openrouter:token"
  const logoutLoading = activeAction === "openrouter:logout"
  const stopLabel = t("credentials.actions.stopLoading")
  const placeholder = maskedSecretPlaceholder(
    savedTokenMask,
    t("credentials.fields.openrouterToken"),
  )

  return (
    <CredentialCard
      title={
        <span className="inline-flex items-center gap-2">
          <span className="border-muted inline-flex size-6 items-center justify-center rounded-full border">
            <IconRouteAltLeft className="size-3.5" />
          </span>
          <span>OpenRouter</span>
        </span>
      }
      description={t("credentials.providers.openrouter.description")}
      status={status}
      authMethod="api_key"
      details={
        <p>
          {t("credentials.labels.modelCount")}: {modelCount}
        </p>
      }
      actions={
        <div className="border-muted flex h-[120px] flex-col justify-center rounded-lg border p-3">
          <div className="flex h-full items-center gap-2">
            <Input
              value={token}
              onChange={(e) => onTokenChange(e.target.value)}
              type="password"
              placeholder={placeholder}
            />
            <Button
              size="sm"
              className="w-fit"
              disabled={actionBusy || !token.trim()}
              onClick={onSaveToken}
            >
              {tokenLoading && <IconLoader2 className="size-4 animate-spin" />}
              <IconRouteAltLeft className="size-4" />
              {t("credentials.actions.saveToken")}
            </Button>
            {tokenLoading && (
              <Button
                size="icon-sm"
                variant="ghost"
                onClick={onStopLoading}
                aria-label={stopLabel}
                title={stopLabel}
                className="text-destructive hover:bg-destructive/10 hover:text-destructive"
              >
                <IconPlayerStopFilled className="size-4" />
              </Button>
            )}
          </div>
        </div>
      }
      footer={
        status === "connected" ? (
          <Button
            variant="ghost"
            size="sm"
            disabled={actionBusy}
            onClick={onAskLogout}
            className="text-destructive hover:bg-destructive/10 hover:text-destructive"
          >
            {logoutLoading && <IconLoader2 className="size-4 animate-spin" />}
            {t("credentials.actions.logout")}
          </Button>
        ) : null
      }
    />
  )
}

import {
  IconKey,
  IconLoader2,
  IconPlayerStopFilled,
  IconSettings,
} from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { OAuthProviderStatus } from "@/api/oauth"
import { maskedSecretPlaceholder } from "@/components/secret-placeholder"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"

import { CredentialCard } from "./credential-card"

interface ModalCredentialCardProps {
  id: string
  title: string
  description: string
  status: OAuthProviderStatus["status"]
  token: string
  savedTokenMask: string
  activeAction: string
  open: boolean
  onOpenChange: (open: boolean) => void
  onTokenChange: (value: string) => void
  onSave: () => void
  onStopLoading: () => void
  onAskLogout: () => void
}

export function ModalCredentialCard({
  id,
  title,
  description,
  status,
  token,
  savedTokenMask,
  activeAction,
  open,
  onOpenChange,
  onTokenChange,
  onSave,
  onStopLoading,
  onAskLogout,
}: ModalCredentialCardProps) {
  const { t } = useTranslation()
  const actionBusy = activeAction !== ""
  const tokenLoading = activeAction === `${id}:token`
  const logoutLoading = activeAction === `${id}:logout`
  const placeholder = maskedSecretPlaceholder(
    savedTokenMask,
    t("credentials.fields.apiToken"),
  )

  return (
    <>
      <CredentialCard
        title={title}
        description={description}
        status={status}
        authMethod="api_key"
        details={
          status === "connected" ? (
            <p>{t("credentials.labels.credentialSaved")}</p>
          ) : (
            <p>{t("credentials.labels.configureInModal")}</p>
          )
        }
        actions={
          <div className="border-muted flex h-[120px] items-center rounded-lg border p-3">
            <Button
              size="sm"
              variant="outline"
              onClick={() => onOpenChange(true)}
              disabled={actionBusy}
            >
              <IconSettings className="size-4" />
              {t("credentials.actions.configure")}
            </Button>
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

      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent className="w-full sm:max-w-md">
          <SheetHeader>
            <SheetTitle>{title}</SheetTitle>
            <SheetDescription>{description}</SheetDescription>
          </SheetHeader>

          <div className="flex flex-1 flex-col gap-4 px-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">
                {t("credentials.fields.apiToken")}
              </label>
              <Input
                value={token}
                onChange={(event) => onTokenChange(event.target.value)}
                type="password"
                placeholder={placeholder}
              />
            </div>
          </div>

          <SheetFooter>
            <div className="flex items-center justify-end gap-2">
              {tokenLoading && (
                <Button
                  size="icon-sm"
                  variant="ghost"
                  onClick={onStopLoading}
                  className="text-destructive hover:bg-destructive/10 hover:text-destructive"
                >
                  <IconPlayerStopFilled className="size-4" />
                </Button>
              )}
              <Button
                onClick={onSave}
                disabled={actionBusy || !token.trim()}
                className="w-fit"
              >
                {tokenLoading ? (
                  <IconLoader2 className="size-4 animate-spin" />
                ) : (
                  <IconKey className="size-4" />
                )}
                {t("credentials.actions.saveToken")}
              </Button>
            </div>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </>
  )
}

import { IconLoader2 } from "@tabler/icons-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"

import { PageHeader } from "@/components/page-header"
import {
  modalCredentialDefinitions,
  useCredentialsPage,
} from "@/hooks/use-credentials-page"

import { AnthropicCredentialCard } from "./anthropic-credential-card"
import { DeviceCodeSheet } from "./device-code-sheet"
import { LogoutConfirmDialog } from "./logout-confirm-dialog"
import { ModalCredentialCard } from "./modal-credential-card"
import { OpenAICredentialCard } from "./openai-credential-card"
import { OpenRouterCredentialCard } from "./openrouter-credential-card"

export function CredentialsPage() {
  const { t } = useTranslation()
  const [openModalProvider, setOpenModalProvider] = useState("")
  const {
    loading,
    error,
    activeAction,
    activeFlow,
    flowHint,
    openAIToken,
    anthropicToken,
    openRouterToken,
    modalTokens,
    openaiStatus,
    anthropicStatus,
    openrouterStatus,
    openrouterModelCount,
    openrouterMaskedToken,
    modalCredentialStatuses,
    logoutDialogOpen,
    logoutConfirmProvider,
    logoutProviderLabel,
    deviceSheetOpen,
    deviceFlow,
    setOpenAIToken,
    setAnthropicToken,
    setOpenRouterToken,
    setModalToken,
    startBrowserOAuth,
    startOpenAIDeviceCode,
    stopLoading,
    saveToken,
    saveOpenRouterToken,
    saveModalCredential,
    askLogout,
    handleConfirmLogout,
    handleLogoutDialogOpenChange,
    handleDeviceSheetOpenChange,
  } = useCredentialsPage()

  const imageProviders = modalCredentialDefinitions.filter(
    (definition) => definition.section === "image",
  )
  const voiceProviders = modalCredentialDefinitions.filter(
    (definition) => definition.section === "voice",
  )

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("navigation.credentials")} />

      <div className="min-h-0 flex-1 overflow-y-auto px-4 sm:px-6">
        <div className="pt-2">
          <p className="text-muted-foreground text-sm">
            {t("credentials.description")}
          </p>
        </div>

        {error && (
          <div className="text-destructive bg-destructive/10 mt-4 rounded-lg px-4 py-3 text-sm">
            {error}
          </div>
        )}

        {activeFlow && (
          <div className="bg-muted mt-4 rounded-lg border px-4 py-3 text-sm">
            <p className="font-medium">{t("credentials.flow.current")}</p>
            <p className="text-muted-foreground mt-1">{flowHint}</p>
          </div>
        )}

        {loading ? (
          <div className="text-muted-foreground flex items-center gap-2 py-10 text-sm">
            <IconLoader2 className="size-4 animate-spin" />
            {t("credentials.loading")}
          </div>
        ) : (
          <div className="space-y-8 py-5">
            <section>
              <h2 className="text-foreground text-sm font-semibold">
                {t("credentials.sections.chat")}
              </h2>
              <div className="mt-3 grid grid-cols-1 gap-4 lg:auto-rows-fr lg:grid-cols-3">
                <OpenAICredentialCard
                  status={openaiStatus}
                  activeAction={activeAction}
                  token={openAIToken}
                  onTokenChange={setOpenAIToken}
                  onStartBrowserOAuth={() => void startBrowserOAuth("openai")}
                  onStartDeviceCode={() => void startOpenAIDeviceCode()}
                  onStopLoading={stopLoading}
                  onSaveToken={() =>
                    void saveToken("openai", openAIToken.trim())
                  }
                  onAskLogout={() => askLogout("openai")}
                />

                <AnthropicCredentialCard
                  status={anthropicStatus}
                  activeAction={activeAction}
                  token={anthropicToken}
                  onTokenChange={setAnthropicToken}
                  onStopLoading={stopLoading}
                  onSaveToken={() =>
                    void saveToken("anthropic", anthropicToken.trim())
                  }
                  onAskLogout={() => askLogout("anthropic")}
                />

                <OpenRouterCredentialCard
                  status={openrouterStatus}
                  activeAction={activeAction}
                  token={openRouterToken}
                  savedTokenMask={openrouterMaskedToken}
                  modelCount={openrouterModelCount}
                  onTokenChange={setOpenRouterToken}
                  onStopLoading={stopLoading}
                  onSaveToken={() => void saveOpenRouterToken()}
                  onAskLogout={() => askLogout("openrouter")}
                />
              </div>
            </section>

            <section>
              <h2 className="text-foreground text-sm font-semibold">
                {t("credentials.sections.image")}
              </h2>
              <div className="mt-3 grid grid-cols-1 gap-4 lg:auto-rows-fr lg:grid-cols-3">
                {imageProviders.map((definition) => (
                  <ModalCredentialCard
                    key={definition.id}
                    id={definition.id}
                    title={definition.name}
                    description={definition.description}
                    status={modalCredentialStatuses[definition.id].status}
                    token={modalTokens[definition.id] ?? ""}
                    savedTokenMask={
                      modalCredentialStatuses[definition.id].savedTokenMask
                    }
                    activeAction={activeAction}
                    open={openModalProvider === definition.id}
                    onOpenChange={(open) =>
                      setOpenModalProvider(open ? definition.id : "")
                    }
                    onTokenChange={(value) =>
                      setModalToken(definition.id, value)
                    }
                    onSave={() => void saveModalCredential(definition)}
                    onStopLoading={stopLoading}
                    onAskLogout={() => askLogout(definition.id)}
                  />
                ))}
              </div>
            </section>

            <section>
              <h2 className="text-foreground text-sm font-semibold">
                {t("credentials.sections.voice")}
              </h2>
              <div className="mt-3 grid grid-cols-1 gap-4 lg:auto-rows-fr lg:grid-cols-3">
                {voiceProviders.map((definition) => (
                  <ModalCredentialCard
                    key={definition.id}
                    id={definition.id}
                    title={definition.name}
                    description={definition.description}
                    status={modalCredentialStatuses[definition.id].status}
                    token={modalTokens[definition.id] ?? ""}
                    savedTokenMask={
                      modalCredentialStatuses[definition.id].savedTokenMask
                    }
                    activeAction={activeAction}
                    open={openModalProvider === definition.id}
                    onOpenChange={(open) =>
                      setOpenModalProvider(open ? definition.id : "")
                    }
                    onTokenChange={(value) =>
                      setModalToken(definition.id, value)
                    }
                    onSave={() => void saveModalCredential(definition)}
                    onStopLoading={stopLoading}
                    onAskLogout={() => askLogout(definition.id)}
                  />
                ))}
              </div>
            </section>
          </div>
        )}
      </div>

      <LogoutConfirmDialog
        open={logoutDialogOpen}
        providerLabel={logoutProviderLabel}
        isSubmitting={activeAction === `${logoutConfirmProvider}:logout`}
        onOpenChange={handleLogoutDialogOpenChange}
        onConfirm={handleConfirmLogout}
      />

      <DeviceCodeSheet
        open={deviceSheetOpen}
        flow={deviceFlow}
        flowHint={flowHint}
        onOpenChange={handleDeviceSheetOpenChange}
      />
    </div>
  )
}

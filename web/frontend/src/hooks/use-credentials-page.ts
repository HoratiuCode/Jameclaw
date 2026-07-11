import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import { getAppConfig, patchAppConfig } from "@/api/channels"
import {
  type ModelInfo,
  addModel,
  getModels,
  setDefaultModel,
  updateModel,
} from "@/api/models"
import {
  type OAuthFlowState,
  type OAuthProvider,
  type OAuthProviderStatus,
  getOAuthFlow,
  getOAuthProviders,
  loginOAuth,
  logoutOAuth,
  pollOAuthFlow,
} from "@/api/oauth"

type FlowWatchMode = "" | "status" | "poll"
type CredentialProvider =
  | OAuthProvider
  | "openrouter"
  | "grok-image"
  | "gemini-image"
  | "elevenlabs"
  | "retell"

type ModalCredentialProvider =
  | "grok-image"
  | "gemini-image"
  | "elevenlabs"
  | "retell"

interface ModalCredentialDefinition {
  id: ModalCredentialProvider
  section: "image" | "voice"
  name: string
  description: string
  modelName?: string
  model?: string
  apiBase?: string
  configPath?: ["voice", "elevenlabs_api_key" | "retell_api_key"]
}

export const modalCredentialDefinitions: ModalCredentialDefinition[] = [
  {
    id: "grok-image",
    section: "image",
    name: "Grok",
    description: "xAI Grok image model credentials.",
    modelName: "grok-image",
    model: "xai/grok-2-image",
    apiBase: "https://api.x.ai/v1",
  },
  {
    id: "gemini-image",
    section: "image",
    name: "Gemini",
    description: "Google Gemini image generation credentials.",
    modelName: "gemini-image",
    model: "gemini/gemini-2.0-flash-preview-image-generation",
    apiBase: "https://generativelanguage.googleapis.com/v1beta",
  },
  {
    id: "elevenlabs",
    section: "voice",
    name: "ElevenLabs",
    description: "ElevenLabs voice transcription credentials.",
    configPath: ["voice", "elevenlabs_api_key"],
  },
  {
    id: "retell",
    section: "voice",
    name: "Retell",
    description: "Retell voice agent credentials.",
    configPath: ["voice", "retell_api_key"],
  },
]

function getProviderLabel(provider: CredentialProvider | ""): string {
  if (provider === "openai") return "OpenAI"
  if (provider === "anthropic") return "Anthropic"
  if (provider === "openrouter") return "OpenRouter"
  if (provider === "grok-image") return "Grok"
  if (provider === "gemini-image") return "Gemini"
  if (provider === "elevenlabs") return "ElevenLabs"
  if (provider === "retell") return "Retell"
  if (provider === "google-antigravity") return "Google Antigravity"
  return ""
}

function isOpenRouterModel(model: ModelInfo): boolean {
  return model.model.toLowerCase().startsWith("openrouter/")
}

function getNestedString(
  root: Record<string, unknown>,
  path: readonly string[],
): string {
  let current: unknown = root
  for (const key of path) {
    if (!current || typeof current !== "object") {
      return ""
    }
    current = (current as Record<string, unknown>)[key]
  }
  return typeof current === "string" ? current : ""
}

function buildModelUpdatePayload(
  model: ModelInfo,
  overrides: Record<string, unknown>,
): Record<string, unknown> {
  return {
    model_name: model.model_name,
    model: model.model,
    api_base: model.api_base || undefined,
    api_key: model.api_key || undefined,
    proxy: model.proxy || undefined,
    auth_method: model.auth_method || undefined,
    connect_mode: model.connect_mode || undefined,
    workspace: model.workspace || undefined,
    rpm: model.rpm,
    max_tokens_field: model.max_tokens_field || undefined,
    request_timeout: model.request_timeout,
    thinking_level: model.thinking_level || undefined,
    extra_body: model.extra_body,
    ...overrides,
  }
}

export function useCredentialsPage() {
  const { t } = useTranslation()
  const [providers, setProviders] = useState<OAuthProviderStatus[]>([])
  const [models, setModels] = useState<ModelInfo[]>([])
  const [appConfig, setAppConfig] = useState<Record<string, unknown>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")

  const [activeAction, setActiveAction] = useState("")
  const [activeFlow, setActiveFlow] = useState<OAuthFlowState | null>(null)
  const actionTokenRef = useRef(0)

  const [watchFlowID, setWatchFlowID] = useState("")
  const [watchMode, setWatchMode] = useState<FlowWatchMode>("")
  const [pollIntervalMs, setPollIntervalMs] = useState(2000)

  const [openAIToken, setOpenAIToken] = useState("")
  const [anthropicToken, setAnthropicToken] = useState("")
  const [openRouterToken, setOpenRouterToken] = useState("")
  const [modalTokens, setModalTokens] = useState<Record<string, string>>({})

  const [logoutDialogOpen, setLogoutDialogOpen] = useState(false)
  const [logoutConfirmProvider, setLogoutConfirmProvider] =
    useState<CredentialProvider | "">("")

  const [deviceSheetOpen, setDeviceSheetOpen] = useState(false)
  const [deviceFlow, setDeviceFlow] = useState<OAuthFlowState | null>(null)

  const loadProviders = useCallback(async () => {
    try {
      const [oauthData, modelsData, configData] = await Promise.all([
        getOAuthProviders(),
        getModels(),
        getAppConfig(),
      ])
      setProviders(oauthData.providers)
      setModels(modelsData.models)
      setAppConfig(configData)
      setError("")
    } catch (err) {
      setError(
        err instanceof Error ? err.message : t("credentials.errors.loadFailed"),
      )
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void loadProviders()
  }, [loadProviders])

  useEffect(() => {
    if (!watchFlowID || !watchMode) {
      return
    }

    let canceled = false
    let timer: ReturnType<typeof setTimeout> | null = null

    const step = async () => {
      try {
        const flow =
          watchMode === "poll"
            ? await pollOAuthFlow(watchFlowID)
            : await getOAuthFlow(watchFlowID)

        if (canceled) {
          return
        }

        setActiveFlow(flow)
        setDeviceFlow((prev) =>
          prev?.flow_id === flow.flow_id ? { ...prev, ...flow } : prev,
        )

        if (flow.status === "pending") {
          timer = setTimeout(step, pollIntervalMs)
          return
        }

        if (watchMode === "poll") {
          setDeviceSheetOpen(false)
        }

        setWatchFlowID("")
        setWatchMode("")
        setActiveAction("")
        await loadProviders()
      } catch (err) {
        if (canceled) {
          return
        }
        setWatchFlowID("")
        setWatchMode("")
        setActiveAction("")
        setError(
          err instanceof Error
            ? err.message
            : t("credentials.errors.flowFailed"),
        )
      }
    }

    void step()

    return () => {
      canceled = true
      if (timer) {
        clearTimeout(timer)
      }
    }
  }, [loadProviders, pollIntervalMs, t, watchFlowID, watchMode])

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const flowID = params.get("oauth_flow_id")
    if (!flowID) {
      return
    }

    setWatchFlowID(flowID)
    setWatchMode("status")
    setPollIntervalMs(700)

    window.history.replaceState({}, "", window.location.pathname)
  }, [])

  useEffect(() => {
    const onMessage = (event: MessageEvent) => {
      const data = event.data as
        | { type?: string; flowId?: string; status?: string }
        | undefined
      if (!data || data.type !== "jameclaw-oauth-result" || !data.flowId) {
        return
      }

      setWatchFlowID(data.flowId)
      setWatchMode("status")
      setPollIntervalMs(700)
    }

    window.addEventListener("message", onMessage)
    return () => window.removeEventListener("message", onMessage)
  }, [])

  const providersMap = useMemo(() => {
    const map = new Map<OAuthProvider, OAuthProviderStatus>()
    for (const item of providers) {
      map.set(item.provider, item)
    }
    return map
  }, [providers])

  const openaiStatus = providersMap.get("openai")
  const anthropicStatus = providersMap.get("anthropic")
  const openrouterModels = useMemo(
    () => models.filter(isOpenRouterModel),
    [models],
  )
  const openrouterMaskedToken = useMemo(() => {
    return (
      openrouterModels.find((model) => model.api_key)?.api_key?.trim() ?? ""
    )
  }, [openrouterModels])
  const openrouterStatus = useMemo<OAuthProviderStatus["status"]>(() => {
    return openrouterModels.some(
      (model) => model.configured || model.api_key.trim() !== "",
    )
      ? "connected"
      : "not_logged_in"
  }, [openrouterModels])

  const modalCredentialStatuses = useMemo(() => {
    const statuses: Record<
      ModalCredentialProvider,
      {
        status: OAuthProviderStatus["status"]
        savedTokenMask: string
        modelCount: number
      }
    > = {
      "grok-image": {
        status: "not_logged_in",
        savedTokenMask: "",
        modelCount: 0,
      },
      "gemini-image": {
        status: "not_logged_in",
        savedTokenMask: "",
        modelCount: 0,
      },
      elevenlabs: {
        status: "not_logged_in",
        savedTokenMask: "",
        modelCount: 0,
      },
      retell: {
        status: "not_logged_in",
        savedTokenMask: "",
        modelCount: 0,
      },
    }

    for (const definition of modalCredentialDefinitions) {
      if (definition.modelName) {
        const matchingModels = models.filter(
          (model) => model.model_name === definition.modelName,
        )
        const savedTokenMask =
          matchingModels.find((model) => model.api_key)?.api_key?.trim() ?? ""
        statuses[definition.id] = {
          status:
            matchingModels.some(
              (model) => model.configured || model.api_key.trim() !== "",
            ) || savedTokenMask
              ? "connected"
              : "not_logged_in",
          savedTokenMask,
          modelCount: matchingModels.length,
        }
        continue
      }

      if (definition.configPath) {
        const hasSavedToken =
          getNestedString(appConfig, definition.configPath).trim() !== ""
        statuses[definition.id] = {
          status: hasSavedToken ? "connected" : "not_logged_in",
          savedTokenMask: hasSavedToken ? "••••••••" : "",
          modelCount: 0,
        }
      }
    }

    return statuses
  }, [appConfig, models])

  const bumpActionToken = useCallback(() => {
    actionTokenRef.current += 1
    return actionTokenRef.current
  }, [])

  const isActionTokenCurrent = useCallback((token: number) => {
    return actionTokenRef.current === token
  }, [])

  const startBrowserOAuth = useCallback(
    async (provider: OAuthProvider) => {
      const actionToken = bumpActionToken()
      setActiveAction(`${provider}:browser`)
      setError("")

      const authTab = window.open("", "_blank")
      if (!authTab) {
        if (!isActionTokenCurrent(actionToken)) {
          return
        }
        setActiveAction("")
        setError(t("credentials.errors.popupBlocked"))
        return
      }

      try {
        const resp = await loginOAuth({ provider, method: "browser" })
        if (!isActionTokenCurrent(actionToken)) {
          authTab.close()
          return
        }
        if (!resp.auth_url || !resp.flow_id) {
          throw new Error(t("credentials.errors.invalidBrowserResponse"))
        }

        authTab.location.href = resp.auth_url

        setActiveFlow({
          flow_id: resp.flow_id,
          provider,
          method: "browser",
          status: "pending",
          expires_at: resp.expires_at,
        })
        setWatchFlowID(resp.flow_id)
        setWatchMode("status")
        setPollIntervalMs(2000)
      } catch (err) {
        if (!isActionTokenCurrent(actionToken)) {
          authTab.close()
          return
        }
        authTab.close()
        setActiveAction("")
        setError(
          err instanceof Error
            ? err.message
            : t("credentials.errors.loginFailed"),
        )
      }
    },
    [bumpActionToken, isActionTokenCurrent, t],
  )

  const startOpenAIDeviceCode = useCallback(async () => {
    const actionToken = bumpActionToken()
    setActiveAction("openai:device")
    setError("")

    try {
      const resp = await loginOAuth({
        provider: "openai",
        method: "device_code",
      })
      if (!isActionTokenCurrent(actionToken)) {
        return
      }
      if (!resp.flow_id || !resp.user_code || !resp.verify_url) {
        throw new Error(t("credentials.errors.invalidDeviceResponse"))
      }

      const flow: OAuthFlowState = {
        flow_id: resp.flow_id,
        provider: "openai",
        method: "device_code",
        status: "pending",
        user_code: resp.user_code,
        verify_url: resp.verify_url,
        interval: resp.interval,
        expires_at: resp.expires_at,
      }

      setDeviceFlow(flow)
      setDeviceSheetOpen(true)
      setActiveFlow(flow)
      setWatchFlowID(resp.flow_id)
      setWatchMode("poll")
      setPollIntervalMs(Math.max(1000, (resp.interval ?? 5) * 1000))
    } catch (err) {
      if (!isActionTokenCurrent(actionToken)) {
        return
      }
      setActiveAction("")
      setError(
        err instanceof Error
          ? err.message
          : t("credentials.errors.loginFailed"),
      )
    }
  }, [bumpActionToken, isActionTokenCurrent, t])

  const saveToken = useCallback(
    async (provider: OAuthProvider, token: string) => {
      const actionID = `${provider}:token`
      setActiveAction(actionID)
      setError("")

      try {
        await loginOAuth({ provider, method: "token", token })
        if (provider === "openai") {
          setOpenAIToken("")
        }
        if (provider === "anthropic") {
          setAnthropicToken("")
        }
        await loadProviders()
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : t("credentials.errors.loginFailed"),
        )
      } finally {
        setActiveAction("")
      }
    },
    [loadProviders, t],
  )

  const saveOpenRouterToken = useCallback(async () => {
    const token = openRouterToken.trim()
    if (!token) {
      return
    }
    if (openrouterModels.length === 0) {
      setError(t("credentials.errors.openrouterModelsMissing"))
      return
    }

    setActiveAction("openrouter:token")
    setError("")

    try {
      await Promise.all(
        openrouterModels.map((model) =>
          updateModel(
            model.index,
            buildModelUpdatePayload(model, { api_key: token }),
          ),
        ),
      )
      setOpenRouterToken("")
      await loadProviders()
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : t("credentials.errors.loginFailed"),
      )
    } finally {
      setActiveAction("")
    }
  }, [loadProviders, openRouterToken, openrouterModels, t])

  const setModalToken = useCallback(
    (provider: ModalCredentialProvider, value: string) => {
      setModalTokens((prev) => ({ ...prev, [provider]: value }))
    },
    [],
  )

  const saveModalCredential = useCallback(
    async (definition: ModalCredentialDefinition) => {
      const token = modalTokens[definition.id]?.trim() ?? ""
      if (!token) {
        return
      }

      setActiveAction(`${definition.id}:token`)
      setError("")

      try {
        if (definition.modelName && definition.model && definition.apiBase) {
          const existing = models.find(
            (model) => model.model_name === definition.modelName,
          )
          if (existing) {
            await updateModel(
              existing.index,
              buildModelUpdatePayload(existing, {
                api_key: token,
                api_base: definition.apiBase,
                model: definition.model,
              }),
            )
          } else {
            await addModel({
              model_name: definition.modelName,
              model: definition.model,
              api_base: definition.apiBase,
              api_key: token,
              request_timeout: 60,
            })
          }
          await setDefaultModel(definition.modelName, "image")
        } else if (definition.configPath) {
          await patchAppConfig({
            [definition.configPath[0]]: {
              [definition.configPath[1]]: token,
            },
          })
        }

        setModalTokens((prev) => ({ ...prev, [definition.id]: "" }))
        await loadProviders()
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : t("credentials.errors.loginFailed"),
        )
      } finally {
        setActiveAction("")
      }
    },
    [loadProviders, modalTokens, models, t],
  )

  const doLogout = useCallback(
    async (provider: CredentialProvider) => {
      const actionID = `${provider}:logout`
      setActiveAction(actionID)
      setError("")

      try {
        if (provider === "openrouter") {
          if (openrouterModels.length === 0) {
            throw new Error(t("credentials.errors.openrouterModelsMissing"))
          }
          await Promise.all(
            openrouterModels.map((model) =>
              updateModel(
                model.index,
                buildModelUpdatePayload(model, {
                  api_key: undefined,
                  clear_api_key: true,
                }),
              ),
            ),
          )
          setOpenRouterToken("")
        } else if (provider === "grok-image" || provider === "gemini-image") {
          const definition = modalCredentialDefinitions.find(
            (item) => item.id === provider,
          )
          if (!definition?.modelName) {
            throw new Error(t("credentials.errors.logoutFailed"))
          }
          const matchingModels = models.filter(
            (model) => model.model_name === definition.modelName,
          )
          await Promise.all(
            matchingModels.map((model) =>
              updateModel(
                model.index,
                buildModelUpdatePayload(model, {
                  api_key: undefined,
                  clear_api_key: true,
                }),
              ),
            ),
          )
          setModalTokens((prev) => ({ ...prev, [provider]: "" }))
        } else if (provider === "elevenlabs" || provider === "retell") {
          const definition = modalCredentialDefinitions.find(
            (item) => item.id === provider,
          )
          if (!definition?.configPath) {
            throw new Error(t("credentials.errors.logoutFailed"))
          }
          await patchAppConfig({
            [definition.configPath[0]]: {
              [definition.configPath[1]]: "",
            },
          })
          setModalTokens((prev) => ({ ...prev, [provider]: "" }))
        } else {
          await logoutOAuth(provider)
        }
        await loadProviders()
      } catch (err) {
        setError(
          err instanceof Error
            ? err.message
            : t("credentials.errors.logoutFailed"),
        )
      } finally {
        setActiveAction("")
      }
    },
    [loadProviders, models, openrouterModels, t],
  )

  const askLogout = useCallback((provider: CredentialProvider) => {
    setLogoutConfirmProvider(provider)
    setLogoutDialogOpen(true)
  }, [])

  const handleConfirmLogout = useCallback(async () => {
    if (!logoutConfirmProvider) {
      return
    }
    await doLogout(logoutConfirmProvider)
    setLogoutDialogOpen(false)
    setLogoutConfirmProvider("")
  }, [doLogout, logoutConfirmProvider])

  const handleLogoutDialogOpenChange = useCallback((open: boolean) => {
    setLogoutDialogOpen(open)
    if (!open) {
      setLogoutConfirmProvider("")
    }
  }, [])

  const handleDeviceSheetOpenChange = useCallback(
    (open: boolean) => {
      setDeviceSheetOpen(open)
      if (open) {
        return
      }

      if (watchMode === "poll") {
        setWatchFlowID("")
        setWatchMode("")
        if (activeAction === "openai:device") {
          setActiveAction("")
        }
      }

      setDeviceFlow(null)
      if (
        activeFlow?.method === "device_code" &&
        activeFlow.status === "pending"
      ) {
        setActiveFlow(null)
      }
    },
    [activeAction, activeFlow, watchMode],
  )

  const stopLoading = useCallback(() => {
    bumpActionToken()
    setWatchFlowID("")
    setWatchMode("")
    setActiveAction("")
    setDeviceSheetOpen(false)
    setDeviceFlow(null)
    setActiveFlow((prev) => (prev?.status === "pending" ? null : prev))
  }, [bumpActionToken])

  const logoutProviderLabel = getProviderLabel(logoutConfirmProvider)

  const flowHint = useMemo(() => {
    if (!activeFlow) {
      return ""
    }
    if (activeFlow.status === "pending") {
      return t("credentials.flow.pending")
    }
    if (activeFlow.status === "success") {
      return t("credentials.flow.success")
    }
    if (activeFlow.status === "expired") {
      return t("credentials.flow.expired")
    }
    return activeFlow.error || t("credentials.flow.error")
  }, [activeFlow, t])

  return {
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
    openrouterModelCount: openrouterModels.length,
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
  }
}

import { IconLoader2, IconPlus } from "@tabler/icons-react"
import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import {
  type ModelInfo,
  type ModelRole,
  type ProviderCatalogEntry,
  getModelCatalog,
  getModels,
	setModelFailover,
  setDefaultModel,
} from "@/api/models"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

import { AddModelSheet } from "./add-model-sheet"
import { AddProviderModelSheet } from "./add-provider-model-sheet"
import { DeleteModelDialog } from "./delete-model-dialog"
import { EditModelSheet } from "./edit-model-sheet"
import { getProviderKey, getProviderLabel } from "./provider-label"
import { ProviderSection } from "./provider-section"

interface ProviderGroup {
  key: string
  label: string
  models: ModelInfo[]
  hasDefault: boolean
  configuredCount: number
}

export function ModelsPage() {
  const { t } = useTranslation()
  const [models, setModels] = useState<ModelInfo[]>([])
  const [providers, setProviders] = useState<ProviderCatalogEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [fetchError, setFetchError] = useState("")

  const [editingModel, setEditingModel] = useState<ModelInfo | null>(null)
  const [deletingModel, setDeletingModel] = useState<ModelInfo | null>(null)
  const [addOpen, setAddOpen] = useState(false)
  const [providerAddOpen, setProviderAddOpen] = useState(false)
  const [settingDefault, setSettingDefault] = useState<{
    index: number
    role: ModelRole
  } | null>(null)
	const [failoverPrimary, setFailoverPrimary] = useState("")
	const [failoverSecondary, setFailoverSecondary] = useState("")
	const [savingFailover, setSavingFailover] = useState(false)
	const [failoverError, setFailoverError] = useState("")

  const fetchModels = useCallback(async () => {
    try {
      const [data, catalog] = await Promise.all([
        getModels(),
        getModelCatalog(),
      ])
      const sorted = [...data.models].sort((a, b) => {
        const aRoleCount =
          Number(a.is_default) +
          Number(a.is_image_default) +
          Number(a.is_voice_default)
        const bRoleCount =
          Number(b.is_default) +
          Number(b.is_image_default) +
          Number(b.is_voice_default)
        if (aRoleCount !== bRoleCount) return bRoleCount - aRoleCount
        if (a.configured && !b.configured) return -1
        if (!a.configured && b.configured) return 1
        return a.model_name.localeCompare(b.model_name)
      })
      setModels(sorted)
      setProviders(catalog.providers)
		setFailoverPrimary(data.default_model)
		setFailoverSecondary(data.model_fallbacks?.[0] ?? "")
      setFetchError("")
    } catch (e) {
      setFetchError(e instanceof Error ? e.message : t("models.loadError"))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    fetchModels()
  }, [fetchModels])

  const handleSetDefault = async (model: ModelInfo, role: ModelRole) => {
    if (
      (role === "chat" && model.is_default) ||
      (role === "image" && model.is_image_default) ||
      (role === "voice" && model.is_voice_default)
    ) {
      return
    }

    setSettingDefault({ index: model.index, role })
    try {
      await setDefaultModel(model.model_name, role)
      await fetchModels()
    } catch {
      // ignore
    } finally {
      setSettingDefault(null)
    }
  }

  const grouped: Record<string, { label: string; models: ModelInfo[] }> = {}
  const providerByProtocol = new Map<string, ProviderCatalogEntry>()
  providers.forEach((provider) => {
    provider.protocols.forEach((protocol) =>
      providerByProtocol.set(protocol, provider),
    )
  })
  for (const model of models) {
    const providerKey = getProviderKey(model.model)
    const catalogProvider = providerByProtocol.get(providerKey)
    if (!grouped[providerKey]) {
      grouped[providerKey] = {
        label: catalogProvider?.name ?? getProviderLabel(model.model),
        models: [],
      }
    }
    grouped[providerKey].models.push(model)
  }

  const providerGroups: ProviderGroup[] = Object.entries(grouped)
    .map(([key, group]) => {
      const configuredCount = group.models.filter(
        (model) => model.configured,
      ).length
      return {
        key,
        label: group.label,
        models: group.models,
        hasDefault: group.models.some(
          (model) =>
            model.is_default ||
            model.is_image_default ||
            model.is_voice_default,
        ),
        configuredCount,
      }
    })
    .sort((a, b) => {
      if (a.hasDefault && !b.hasDefault) return -1
      if (!a.hasDefault && b.hasDefault) return 1

      if (a.configuredCount !== b.configuredCount) {
        return b.configuredCount - a.configuredCount
      }

      const aPriority = providers.findIndex((provider) =>
        provider.protocols.includes(a.key),
      )
      const bPriority = providers.findIndex((provider) =>
        provider.protocols.includes(b.key),
      )
      const aRank = aPriority >= 0 ? aPriority : Number.MAX_SAFE_INTEGER
      const bRank = bPriority >= 0 ? bPriority : Number.MAX_SAFE_INTEGER
      if (aRank !== bRank) {
        return aRank - bRank
      }

      return a.label.localeCompare(b.label)
    })

  const defaultModel = models.find((model) => model.is_default)
  const defaultImageModel = models.find((model) => model.is_image_default)
  const defaultVoiceModel = models.find((model) => model.is_voice_default)
	const configuredChatModels = models.filter((model) => model.configured)
	const saveFailover = async () => {
		if (!failoverPrimary || !failoverSecondary || failoverPrimary === failoverSecondary) {
			setFailoverError("Choose two different configured models.")
			return
		}
		setSavingFailover(true)
		setFailoverError("")
		try {
			await setModelFailover(failoverPrimary, failoverSecondary)
			await fetchModels()
		} catch (error) {
			setFailoverError(error instanceof Error ? error.message : "Could not save provider failover.")
		} finally {
			setSavingFailover(false)
		}
	}

  return (
    <div className="flex h-full flex-col">
      <PageHeader title={t("navigation.models")}>
        <div className="flex items-center gap-3">
          <Button size="sm" onClick={() => setProviderAddOpen(true)}>
            <IconPlus className="size-4" />
            Add Provider Model
          </Button>
          <Button size="sm" variant="outline" onClick={() => setAddOpen(true)}>
            <IconPlus className="size-4" />
            {t("models.add.button")}
          </Button>
        </div>
      </PageHeader>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 sm:px-6">
        <div className="pt-2">
          {(!defaultModel || !defaultImageModel || !defaultVoiceModel) && (
            <div className="text-muted-foreground flex items-center gap-1.5 text-sm">
              <span>{t("models.noDefaultHintPrefix")}</span>
            </div>
          )}
          <p className="text-muted-foreground mt-1 text-sm">
            {t("models.description")}
          </p>
        </div>

        {loading && (
          <div className="flex items-center justify-center py-20">
            <IconLoader2 className="text-muted-foreground size-6 animate-spin" />
          </div>
        )}

        {fetchError && (
          <div className="text-destructive bg-destructive/10 rounded-lg px-4 py-3 text-sm">
            {fetchError}
          </div>
        )}

        {!loading && !fetchError && (
          <div className="pb-8">
			<ProviderFailoverCard
				models={configuredChatModels}
				primary={failoverPrimary}
				secondary={failoverSecondary}
				onPrimaryChange={setFailoverPrimary}
				onSecondaryChange={setFailoverSecondary}
				onSave={() => void saveFailover()}
				saving={savingFailover}
				error={failoverError}
			/>
            {providerGroups.map((providerGroup) => (
              <ProviderSection
                key={providerGroup.key}
                provider={providerGroup.label}
                providerKey={providerGroup.key}
                models={providerGroup.models}
                onEdit={setEditingModel}
                onSetDefault={handleSetDefault}
                onDelete={setDeletingModel}
                settingDefault={settingDefault}
              />
            ))}
          </div>
        )}
      </div>

      <EditModelSheet
        model={editingModel}
        open={editingModel !== null}
        onClose={() => setEditingModel(null)}
        onSaved={fetchModels}
      />

      <AddModelSheet
        open={addOpen}
        onClose={() => setAddOpen(false)}
        onSaved={fetchModels}
        existingModelNames={models.map((model) => model.model_name)}
      />

      <AddProviderModelSheet
        open={providerAddOpen}
        onClose={() => setProviderAddOpen(false)}
        onSaved={fetchModels}
        providers={providers}
        existingModelNames={models.map((model) => model.model_name)}
      />

      <DeleteModelDialog
        model={deletingModel}
        onClose={() => setDeletingModel(null)}
        onDeleted={fetchModels}
      />
    </div>
  )
}

function ProviderFailoverCard({
	models,
	primary,
	secondary,
	onPrimaryChange,
	onSecondaryChange,
	onSave,
	saving,
	error,
}: {
	models: ModelInfo[]
	primary: string
	secondary: string
	onPrimaryChange: (value: string) => void
	onSecondaryChange: (value: string) => void
	onSave: () => void
	saving: boolean
	error: string
}) {
	const secondaryModels = models.filter((model) => model.model_name !== primary)
	const primaryModels = models.filter((model) => model.model_name !== secondary)
	const hasTwoDistinctModels = primary !== "" && secondary !== "" && primary !== secondary

	return (
		<Card className="my-5 border-primary/25 bg-primary/3">
			<CardHeader>
				<CardTitle>Primary and backup model</CardTitle>
				<CardDescription>
					Choose two different configured models. The backup model takes over automatically when the primary model has a retryable error, such as an outage, rate limit, or timeout.
				</CardDescription>
			</CardHeader>
			<CardContent className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-end">
				<ProviderSelect label="Primary model" value={primary} models={primaryModels} onChange={onPrimaryChange} />
				<ProviderSelect label="Backup model" value={secondary} models={secondaryModels} onChange={onSecondaryChange} />
				<Button onClick={onSave} disabled={saving || models.length < 2 || !hasTwoDistinctModels}>
					{saving ? "Saving…" : "Save model pair"}
				</Button>
				{error && <p className="text-destructive basis-full text-sm">{error}</p>}
			</CardContent>
		</Card>
	)
}

function ProviderSelect({ label, value, models, onChange }: { label: string; value: string; models: ModelInfo[]; onChange: (value: string) => void }) {
	return (
		<div className="grid min-w-52 flex-1 gap-2">
			<label className="text-sm font-medium">{label}</label>
			<Select value={value} onValueChange={onChange}>
				<SelectTrigger><SelectValue placeholder="Choose a configured model" /></SelectTrigger>
				<SelectContent>
					{models.map((model) => <SelectItem key={model.model_name} value={model.model_name}>{model.model_name} · {model.model}</SelectItem>)}
				</SelectContent>
			</Select>
		</div>
	)
}

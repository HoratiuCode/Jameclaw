import { IconLoader2, IconPlus, IconStar } from "@tabler/icons-react"
import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import {
  type ModelInfo,
  type ProviderCatalogEntry,
  getModelCatalog,
  getModels,
  setDefaultModel,
} from "@/api/models"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"

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
  const [settingDefaultIndex, setSettingDefaultIndex] = useState<number | null>(
    null,
  )

  const fetchModels = useCallback(async () => {
    try {
      const [data, catalog] = await Promise.all([
        getModels(),
        getModelCatalog(),
      ])
      const sorted = [...data.models].sort((a, b) => {
        if (a.is_default && !b.is_default) return -1
        if (!a.is_default && b.is_default) return 1
        if (a.configured && !b.configured) return -1
        if (!a.configured && b.configured) return 1
        return a.model_name.localeCompare(b.model_name)
      })
      setModels(sorted)
      setProviders(catalog.providers)
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

  const handleSetDefault = async (model: ModelInfo) => {
    if (model.is_default) return

    setSettingDefaultIndex(model.index)
    try {
      await setDefaultModel(model.model_name)
      await fetchModels()
    } catch {
      // ignore
    } finally {
      setSettingDefaultIndex(null)
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
        hasDefault: group.models.some((model) => model.is_default),
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
          {!defaultModel && (
            <div className="text-muted-foreground flex items-center gap-1.5 text-sm">
              <span>{t("models.noDefaultHintPrefix")}</span>
              <IconStar className="size-3.5 shrink-0" />
              <span>{t("models.noDefaultHintSuffix")}</span>
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
            {providerGroups.map((providerGroup) => (
              <ProviderSection
                key={providerGroup.key}
                provider={providerGroup.label}
                providerKey={providerGroup.key}
                models={providerGroup.models}
                onEdit={setEditingModel}
                onSetDefault={handleSetDefault}
                onDelete={setDeletingModel}
                settingDefaultIndex={settingDefaultIndex}
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

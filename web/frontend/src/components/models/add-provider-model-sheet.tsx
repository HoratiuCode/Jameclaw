import { IconLoader2 } from "@tabler/icons-react"
import { useEffect, useMemo, useState } from "react"

import {
  type ModelPreset,
  type ProviderCatalogEntry,
  addModelFromCatalog,
} from "@/api/models"
import { maskedSecretPlaceholder } from "@/components/secret-placeholder"
import { Field, KeyInput, SwitchCardField } from "@/components/shared-form"
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

interface AddProviderModelSheetProps {
  open: boolean
  onClose: () => void
  onSaved: () => void
  providers: ProviderCatalogEntry[]
  existingModelNames: string[]
}

export function AddProviderModelSheet({
  open,
  onClose,
  onSaved,
  providers,
  existingModelNames,
}: AddProviderModelSheetProps) {
  const sortedProviders = useMemo(
    () =>
      [...providers]
        .filter((provider) => provider.recommended_models.length > 0)
        .sort((a, b) => {
          if (a.default && !b.default) return -1
          if (!a.default && b.default) return 1
          if (a.configured && !b.configured) return -1
          if (!a.configured && b.configured) return 1
          return a.name.localeCompare(b.name)
        }),
    [providers],
  )
  const [providerID, setProviderID] = useState("")
  const [presetID, setPresetID] = useState("")
  const [modelName, setModelName] = useState("")
  const [apiKey, setAPIKey] = useState("")
  const [setAsDefault, setSetAsDefault] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")

  const provider = sortedProviders.find((item) => item.id === providerID)
  const preset = provider?.recommended_models.find(
    (item) => item.id === presetID,
  )

  useEffect(() => {
    if (!open) return
    const firstProvider = sortedProviders[0]
    const firstPreset = firstProvider?.recommended_models[0]
    setProviderID(firstProvider?.id ?? "")
    setPresetID(firstPreset?.id ?? "")
    setModelName(firstPreset?.model_name ?? "")
    setAPIKey("")
    setSetAsDefault(false)
    setError("")
  }, [open, sortedProviders])

  useEffect(() => {
    if (!provider) return
    const nextPreset =
      provider.recommended_models.find((item) => item.id === presetID) ??
      provider.recommended_models[0]
    setPresetID(nextPreset?.id ?? "")
    setModelName(nextPreset?.model_name ?? "")
  }, [provider, presetID])

  const handlePresetChange = (value: string) => {
    setPresetID(value)
    const nextPreset = provider?.recommended_models.find(
      (item) => item.id === value,
    )
    setModelName(nextPreset?.model_name ?? "")
  }

  const validate = (selectedPreset: ModelPreset | undefined): string => {
    if (!provider) return "Select a provider."
    if (!selectedPreset) return "Select a model preset."
    if (!modelName.trim()) return "Model name is required."
    if (existingModelNames.some((name) => name.trim() === modelName.trim())) {
      return "That model name already exists. Choose another name or edit the existing model."
    }
    if (selectedPreset.requires_api_key && !apiKey.trim()) {
      return `${selectedPreset.key_label || provider.key_label || "API key"} is required.`
    }
    return ""
  }

  const handleSave = async () => {
    const validationError = validate(preset)
    if (validationError) {
      setError(validationError)
      return
    }
    setSaving(true)
    setError("")
    try {
      await addModelFromCatalog({
        provider_id: providerID,
        preset_id: presetID,
        model_name: modelName.trim(),
        api_key: apiKey.trim() || undefined,
        set_default: setAsDefault,
      })
      onSaved()
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to add model.")
    } finally {
      setSaving(false)
    }
  }

  return (
    <Sheet open={open} onOpenChange={(v) => !v && onClose()}>
      <SheetContent
        side="right"
        className="flex flex-col gap-0 p-0 data-[side=right]:!w-full data-[side=right]:sm:!w-[560px] data-[side=right]:sm:!max-w-[560px]"
      >
        <SheetHeader className="border-b-muted border-b px-6 py-5">
          <SheetTitle className="text-base">Add Provider Model</SheetTitle>
          <SheetDescription className="text-xs">
            Choose a catalog provider and JameClaw will create the model config
            for you.
          </SheetDescription>
        </SheetHeader>

        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="space-y-5 px-6 py-5">
            <Field
              label="Provider"
              hint={provider?.setup_hint || provider?.local_runtime_hint}
            >
              <select
                className="border-input bg-background h-10 w-full rounded-md border px-3 text-sm"
                value={providerID}
                onChange={(event) => setProviderID(event.target.value)}
              >
                {sortedProviders.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name}
                  </option>
                ))}
              </select>
            </Field>

            <Field
              label="Preset model"
              hint={preset?.description || preset?.model}
            >
              <select
                className="border-input bg-background h-10 w-full rounded-md border px-3 text-sm"
                value={presetID}
                onChange={(event) => handlePresetChange(event.target.value)}
              >
                {provider?.recommended_models.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name}
                  </option>
                ))}
              </select>
            </Field>

            <Field label="Model name" hint="Local alias stored in model_list.">
              <Input
                value={modelName}
                onChange={(event) => setModelName(event.target.value)}
              />
            </Field>

            {preset?.requires_api_key && (
              <Field
                label={preset.key_label || provider?.key_label || "API key"}
              >
                <KeyInput
                  value={apiKey}
                  onChange={setAPIKey}
                  placeholder={maskedSecretPlaceholder(apiKey, "Paste API key")}
                />
              </Field>
            )}

            <SwitchCardField
              label="Set as default"
              hint="Use this model for new agent conversations."
              checked={setAsDefault}
              onCheckedChange={setSetAsDefault}
            />

            {error && (
              <p className="text-destructive bg-destructive/10 rounded-md px-3 py-2 text-sm">
                {error}
              </p>
            )}
          </div>
        </div>

        <SheetFooter className="border-t-muted border-t px-6 py-4">
          <Button variant="ghost" onClick={onClose} disabled={saving}>
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={saving || !provider || !preset}
          >
            {saving && <IconLoader2 className="size-4 animate-spin" />}
            Add model
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}

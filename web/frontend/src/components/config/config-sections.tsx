import {
  IconBrain,
  IconCircleCheck,
  IconExternalLink,
  IconFolder,
  IconHistory,
  IconLoader2,
  IconPlug,
  IconRefresh,
  IconVersions,
} from "@tabler/icons-react"
import { useQuery } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { useAtom } from "jotai"
import * as React from "react"
import type { ReactNode } from "react"
import { useTranslation } from "react-i18next"

import {
  type AppConfig,
  type SupportedChannel,
  getChannelsCatalog,
} from "@/api/channels"
import type { UpdateStatusResponse } from "@/api/update"
import { getChannelDisplayName } from "@/components/channels/channel-display-name"
import {
  type CoreConfigForm,
  DM_SCOPE_OPTIONS,
  type LauncherForm,
} from "@/components/config/form-model"
import { Field, SwitchCardField } from "@/components/shared-form"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import {
  CHANNEL_ICON_MAP,
  CHANNEL_IMPORTANCE_INDEX,
  buildChannelEnabledMap,
} from "@/hooks/use-sidebar-channels"
import {
  DEFAULT_ACCENT_COLOR,
  type Font,
  type FontSize,
  type Theme,
  designAtom,
  updateDesignStore,
} from "@/store/design"

type UpdateCoreField = <K extends keyof CoreConfigForm>(
  key: K,
  value: CoreConfigForm[K],
) => void

type UpdateLauncherField = <K extends keyof LauncherForm>(
  key: K,
  value: LauncherForm[K],
) => void

interface ConfigSectionCardProps {
  title: string
  description?: string
  children: ReactNode
}

const WEB_EXTENSION_MODEL_SIZE_OPTIONS = [
  {
    value: "small",
    label: "Small",
  },
  {
    value: "medium",
    label: "Medium",
  },
  {
    value: "large",
    label: "Large",
  },
  {
    value: "xlarge",
    label: "Extra Large",
  },
] as const

function ConfigSectionCard({
  title,
  description,
  children,
}: ConfigSectionCardProps) {
  return (
    <Card size="sm">
      <CardHeader className="border-border border-b">
        <CardTitle>{title}</CardTitle>
        {description && <CardDescription>{description}</CardDescription>}
      </CardHeader>
      <CardContent className="pt-0">
        <div className="divide-border/70 divide-y">{children}</div>
      </CardContent>
    </Card>
  )
}

function asRecord(value: unknown): Record<string, unknown> {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>
  }
  return {}
}

function sortChannels(
  channels: SupportedChannel[],
  enabledMap: Record<string, boolean>,
  t: ReturnType<typeof useTranslation>["t"],
) {
  return [...channels].sort((a, b) => {
    const aEnabled = enabledMap[a.name] === true
    const bEnabled = enabledMap[b.name] === true
    if (aEnabled !== bEnabled) {
      return aEnabled ? -1 : 1
    }

    const aImportance =
      CHANNEL_IMPORTANCE_INDEX.get(a.name) ?? Number.MAX_SAFE_INTEGER
    const bImportance =
      CHANNEL_IMPORTANCE_INDEX.get(b.name) ?? Number.MAX_SAFE_INTEGER
    if (aImportance !== bImportance) {
      return aImportance - bImportance
    }

    return getChannelDisplayName(a, t).localeCompare(
      getChannelDisplayName(b, t),
    )
  })
}

interface ChannelsSectionProps {
  appConfig?: AppConfig
}

export function ChannelsSection({ appConfig }: ChannelsSectionProps) {
  const { t } = useTranslation()
  const {
    data: catalog,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["channels", "catalog"],
    queryFn: getChannelsCatalog,
  })

  const channels = catalog?.channels ?? []
  const enabledMap = buildChannelEnabledMap(channels, asRecord(appConfig))
  const sortedChannels = sortChannels(channels, enabledMap, t)

  return (
    <ConfigSectionCard
      title={t("pages.config.sections.channels")}
      description={t("pages.config.channels_description")}
    >
      {isLoading ? (
        <div className="text-muted-foreground py-4 text-sm">
          {t("labels.loading")}
        </div>
      ) : error ? (
        <div className="text-destructive py-4 text-sm">
          {t("channels.loadError")}
        </div>
      ) : (
        <div className="divide-border/70 divide-y">
          {sortedChannels.map((channel) => {
            const enabled = enabledMap[channel.name] === true
            const ChannelIcon = CHANNEL_ICON_MAP[channel.name] ?? IconPlug
            return (
              <div
                key={channel.name}
                className="flex flex-col gap-3 py-4 sm:flex-row sm:items-center sm:justify-between"
              >
                <div className="flex min-w-0 items-center gap-3">
                  <span className="bg-muted text-muted-foreground flex size-9 shrink-0 items-center justify-center rounded-md">
                    <ChannelIcon className="size-4" />
                  </span>
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">
                      {getChannelDisplayName(channel, t)}
                    </p>
                    <p className="text-muted-foreground text-xs">
                      {enabled
                        ? t("pages.config.channel_status_active")
                        : t("pages.config.channel_status_inactive")}
                    </p>
                  </div>
                </div>
                <Button variant="outline" size="sm" asChild>
                  <Link to="/channels/$name" params={{ name: channel.name }}>
                    {t("pages.config.channel_configure")}
                  </Link>
                </Button>
              </div>
            )
          })}
        </div>
      )}
    </ConfigSectionCard>
  )
}

interface DataStorageSectionProps {
  workspace: string
}

export function DataStorageSection({ workspace }: DataStorageSectionProps) {
  const { t } = useTranslation()
  const normalizedWorkspace = workspace.trim() || "~/.jameclaw/workspace"

  const locations = [
    {
      label: t("pages.config.storage.sessions"),
      path: `${normalizedWorkspace}/sessions`,
      icon: IconHistory,
    },
    {
      label: t("pages.config.storage.memory"),
      path: `${normalizedWorkspace}/memory`,
      icon: IconBrain,
    },
    {
      label: t("pages.config.storage.automation"),
      path: `${normalizedWorkspace}/cron`,
      icon: IconFolder,
    },
  ]

  return (
    <ConfigSectionCard
      title={t("pages.config.sections.storage")}
      description={t("pages.config.storage.description")}
    >
      <Field
        label={t("pages.config.storage.workspace")}
        hint={t("pages.config.storage.workspace_hint")}
        layout="setting-row"
      >
        <code className="bg-muted block max-w-full overflow-x-auto rounded-md px-3 py-2 text-xs">
          {normalizedWorkspace}
        </code>
      </Field>

      {locations.map(({ label, path, icon: Icon }) => (
        <div key={label} className="flex items-center gap-3 py-4">
          <span className="bg-muted text-muted-foreground flex size-9 shrink-0 items-center justify-center rounded-md">
            <Icon className="size-4" />
          </span>
          <div className="min-w-0">
            <p className="text-sm font-medium">{label}</p>
            <code className="text-muted-foreground block truncate text-xs">
              {path}
            </code>
          </div>
        </div>
      ))}

      <div className="flex justify-end py-4">
        <Button variant="outline" size="sm" asChild>
          <Link to="/automation">{t("pages.config.storage.open_automations")}</Link>
        </Button>
      </div>
    </ConfigSectionCard>
  )
}

interface AgentDefaultsSectionProps {
  form: CoreConfigForm
  onFieldChange: UpdateCoreField
}

export function AgentDefaultsSection({
  form,
  onFieldChange,
}: AgentDefaultsSectionProps) {
  const { t } = useTranslation()

  return (
    <ConfigSectionCard title={t("pages.config.sections.agent")}>
      <Field
        label={t("pages.config.workspace")}
        hint={t("pages.config.workspace_hint")}
        layout="setting-row"
      >
        <Input
          value={form.workspace}
          onChange={(e) => onFieldChange("workspace", e.target.value)}
          placeholder="~/.jameclaw/workspace"
        />
      </Field>

      <SwitchCardField
        label={t("pages.config.restrict_workspace")}
        hint={t("pages.config.restrict_workspace_hint")}
        layout="setting-row"
        checked={form.restrictToWorkspace}
        onCheckedChange={(checked) =>
          onFieldChange("restrictToWorkspace", checked)
        }
      />

      <SwitchCardField
        label={t("pages.config.tool_feedback_enabled")}
        hint={t("pages.config.tool_feedback_enabled_hint")}
        layout="setting-row"
        checked={form.toolFeedbackEnabled}
        onCheckedChange={(checked) =>
          onFieldChange("toolFeedbackEnabled", checked)
        }
      />

      {form.toolFeedbackEnabled && (
        <Field
          label={t("pages.config.tool_feedback_max_args_length")}
          hint={t("pages.config.tool_feedback_max_args_length_hint")}
          layout="setting-row"
        >
          <Input
            type="number"
            min={0}
            value={form.toolFeedbackMaxArgsLength}
            onChange={(e) =>
              onFieldChange("toolFeedbackMaxArgsLength", e.target.value)
            }
          />
        </Field>
      )}

      <Field
        label={t("pages.config.max_tokens")}
        hint={t("pages.config.max_tokens_hint")}
        layout="setting-row"
      >
        <Input
          type="number"
          min={1}
          value={form.maxTokens}
          onChange={(e) => onFieldChange("maxTokens", e.target.value)}
        />
      </Field>

      <Field
        label={t("pages.config.context_window")}
        hint={t("pages.config.context_window_hint")}
        layout="setting-row"
      >
        <Input
          type="number"
          min={1}
          value={form.contextWindow}
          onChange={(e) => onFieldChange("contextWindow", e.target.value)}
          placeholder="131072"
        />
      </Field>

      <Field
        label={t("pages.config.max_tool_iterations")}
        hint={t("pages.config.max_tool_iterations_hint")}
        layout="setting-row"
      >
        <Input
          type="number"
          min={1}
          value={form.maxToolIterations}
          onChange={(e) => onFieldChange("maxToolIterations", e.target.value)}
        />
      </Field>

      <Field
        label={t("pages.config.summarize_threshold")}
        hint={t("pages.config.summarize_threshold_hint")}
        layout="setting-row"
      >
        <Input
          type="number"
          min={1}
          value={form.summarizeMessageThreshold}
          onChange={(e) =>
            onFieldChange("summarizeMessageThreshold", e.target.value)
          }
        />
      </Field>

      <Field
        label={t("pages.config.summarize_token_percent")}
        hint={t("pages.config.summarize_token_percent_hint")}
        layout="setting-row"
      >
        <Input
          type="number"
          min={1}
          max={100}
          value={form.summarizeTokenPercent}
          onChange={(e) =>
            onFieldChange("summarizeTokenPercent", e.target.value)
          }
        />
      </Field>
    </ConfigSectionCard>
  )
}

interface WebExtensionSectionProps {
  form: CoreConfigForm
  onFieldChange: UpdateCoreField
}

export function WebExtensionSection({
  form,
  onFieldChange,
}: WebExtensionSectionProps) {
  const { t } = useTranslation()

  return (
    <ConfigSectionCard
      title={t("pages.config.sections.web_extension")}
      description={t("pages.config.web_extension_description")}
    >
      <Field
        label={t("pages.config.web_extension_model_size")}
        hint={t("pages.config.web_extension_model_size_hint")}
        layout="setting-row"
      >
        <Select
          value={form.webExtensionModelSize}
          onValueChange={(value) =>
            onFieldChange("webExtensionModelSize", value)
          }
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {WEB_EXTENSION_MODEL_SIZE_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>

      <Field
        label={t("pages.config.web_extension_package")}
        hint={t("pages.config.web_extension_package_hint")}
        layout="setting-row"
      >
        <Input
          value={form.webExtensionPackageName}
          onChange={(e) =>
            onFieldChange("webExtensionPackageName", e.target.value)
          }
          placeholder="Chrome-Extension-Upload"
        />
      </Field>

      <Field
        label={t("pages.config.web_extension_usage_notes")}
        hint={t("pages.config.web_extension_usage_notes_hint")}
        layout="setting-row"
        controlClassName="md:max-w-md"
      >
        <Textarea
          value={form.webExtensionUsageNotes}
          onChange={(e) =>
            onFieldChange("webExtensionUsageNotes", e.target.value)
          }
          className="min-h-[160px]"
        />
      </Field>
    </ConfigSectionCard>
  )
}

interface ExecSectionProps {
  form: CoreConfigForm
  onFieldChange: UpdateCoreField
}

export function ExecSection({ form, onFieldChange }: ExecSectionProps) {
  const { t } = useTranslation()

  return (
    <ConfigSectionCard title={t("pages.config.sections.exec")}>
      <SwitchCardField
        label={t("pages.config.exec_enabled")}
        hint={t("pages.config.exec_enabled_hint")}
        layout="setting-row"
        checked={form.execEnabled}
        onCheckedChange={(checked) => onFieldChange("execEnabled", checked)}
      />

      {form.execEnabled && (
        <>
          <SwitchCardField
            label={t("pages.config.allow_remote")}
            hint={t("pages.config.allow_remote_hint")}
            layout="setting-row"
            checked={form.allowRemote}
            onCheckedChange={(checked) => onFieldChange("allowRemote", checked)}
          />

          <SwitchCardField
            label={t("pages.config.enable_deny_patterns")}
            hint={t("pages.config.enable_deny_patterns_hint")}
            layout="setting-row"
            checked={form.enableDenyPatterns}
            onCheckedChange={(checked) =>
              onFieldChange("enableDenyPatterns", checked)
            }
          />

          {form.enableDenyPatterns && (
            <Field
              label={t("pages.config.custom_deny_patterns")}
              hint={t("pages.config.custom_deny_patterns_hint")}
              layout="setting-row"
              controlClassName="md:max-w-md"
            >
              <Textarea
                value={form.customDenyPatternsText}
                placeholder={t("pages.config.custom_patterns_placeholder")}
                className="min-h-[88px]"
                onChange={(e) =>
                  onFieldChange("customDenyPatternsText", e.target.value)
                }
              />
            </Field>
          )}

          <Field
            label={t("pages.config.custom_allow_patterns")}
            hint={t("pages.config.custom_allow_patterns_hint")}
            layout="setting-row"
            controlClassName="md:max-w-md"
          >
            <Textarea
              value={form.customAllowPatternsText}
              placeholder={t("pages.config.custom_patterns_placeholder")}
              className="min-h-[88px]"
              onChange={(e) =>
                onFieldChange("customAllowPatternsText", e.target.value)
              }
            />
          </Field>

          <Field
            label={t("pages.config.exec_timeout_seconds")}
            hint={t("pages.config.exec_timeout_seconds_hint")}
            layout="setting-row"
          >
            <Input
              type="number"
              min={0}
              value={form.execTimeoutSeconds}
              onChange={(e) =>
                onFieldChange("execTimeoutSeconds", e.target.value)
              }
            />
          </Field>
        </>
      )}
    </ConfigSectionCard>
  )
}

interface MacControlSectionProps {
  form: CoreConfigForm
  onFieldChange: UpdateCoreField
}

export function MacControlSection({
  form,
  onFieldChange,
}: MacControlSectionProps) {
  const { t } = useTranslation()

  return (
    <ConfigSectionCard
      title={t("pages.config.sections.mac_control")}
      description={t("pages.config.mac_control_description")}
    >
      <SwitchCardField
        label={t("pages.config.allow_open_mac_apps")}
        hint={t("pages.config.allow_open_mac_apps_hint")}
        layout="setting-row"
        checked={form.macControlAllowOpenApps}
        onCheckedChange={(checked) =>
          onFieldChange("macControlAllowOpenApps", checked)
        }
      />
      <SwitchCardField
        label={t("pages.config.allow_music_playlists")}
        hint={t("pages.config.allow_music_playlists_hint")}
        layout="setting-row"
        checked={form.macControlAllowMusicPlaylists}
        onCheckedChange={(checked) =>
          onFieldChange("macControlAllowMusicPlaylists", checked)
        }
      />
    </ConfigSectionCard>
  )
}

interface RuntimeSectionProps {
  form: CoreConfigForm
  onFieldChange: UpdateCoreField
}

export function RuntimeSection({ form, onFieldChange }: RuntimeSectionProps) {
  const { t } = useTranslation()
  const selectedDmScopeOption = DM_SCOPE_OPTIONS.find(
    (scope) => scope.value === form.dmScope,
  )

  return (
    <ConfigSectionCard title={t("pages.config.sections.runtime")}>
      <Field
        label={t("pages.config.session_scope")}
        hint={t("pages.config.session_scope_hint")}
        layout="setting-row"
      >
        <Select
          value={form.dmScope}
          onValueChange={(value) => onFieldChange("dmScope", value)}
        >
          <SelectTrigger className="w-full">
            <SelectValue>
              {selectedDmScopeOption
                ? t(
                    selectedDmScopeOption.labelKey,
                    selectedDmScopeOption.labelDefault,
                  )
                : form.dmScope}
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            {DM_SCOPE_OPTIONS.map((scope) => (
              <SelectItem key={scope.value} value={scope.value}>
                <div className="flex flex-col gap-0.5">
                  <span className="font-medium">{t(scope.labelKey)}</span>
                  <span className="text-muted-foreground text-xs">
                    {t(scope.descKey)}
                  </span>
                </div>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>

      <SwitchCardField
        label={t("pages.config.heartbeat_enabled")}
        hint={t("pages.config.heartbeat_enabled_hint")}
        layout="setting-row"
        checked={form.heartbeatEnabled}
        onCheckedChange={(checked) =>
          onFieldChange("heartbeatEnabled", checked)
        }
      />

      {form.heartbeatEnabled && (
        <Field
          label={t("pages.config.heartbeat_interval")}
          hint={t("pages.config.heartbeat_interval_hint")}
          layout="setting-row"
        >
          <Input
            type="number"
            min={1}
            value={form.heartbeatInterval}
            onChange={(e) => onFieldChange("heartbeatInterval", e.target.value)}
          />
        </Field>
      )}
    </ConfigSectionCard>
  )
}

interface CronSectionProps {
  form: CoreConfigForm
  onFieldChange: UpdateCoreField
}

export function CronSection({ form, onFieldChange }: CronSectionProps) {
  const { t } = useTranslation()

  return (
    <ConfigSectionCard title={t("pages.config.sections.cron")}>
      <SwitchCardField
        label={t("pages.config.allow_shell_execution")}
        hint={t("pages.config.allow_shell_execution_hint")}
        layout="setting-row"
        checked={form.allowCommand}
        disabled={!form.execEnabled}
        onCheckedChange={(checked) => onFieldChange("allowCommand", checked)}
      />

      <Field
        label={t("pages.config.cron_exec_timeout")}
        hint={t("pages.config.cron_exec_timeout_hint")}
        layout="setting-row"
      >
        <Input
          type="number"
          min={0}
          disabled={!form.execEnabled}
          value={form.cronExecTimeoutMinutes}
          onChange={(e) =>
            onFieldChange("cronExecTimeoutMinutes", e.target.value)
          }
        />
      </Field>
    </ConfigSectionCard>
  )
}

interface LauncherSectionProps {
  launcherForm: LauncherForm
  onFieldChange: UpdateLauncherField
  disabled: boolean
  autoStartEnabled: boolean
  autoStartHint: string
  autoStartDisabled: boolean
  onAutoStartChange: (checked: boolean) => void
}

export function LauncherSection({
  launcherForm,
  onFieldChange,
  disabled,
  autoStartEnabled,
  autoStartHint,
  autoStartDisabled,
  onAutoStartChange,
}: LauncherSectionProps) {
  const { t } = useTranslation()

  return (
    <ConfigSectionCard title={t("pages.config.sections.launcher")}>
      <SwitchCardField
        label={t("pages.config.autostart_label")}
        hint={autoStartHint}
        layout="setting-row"
        checked={autoStartEnabled}
        disabled={autoStartDisabled}
        onCheckedChange={onAutoStartChange}
      />

      <SwitchCardField
        label={t("pages.config.lan_access")}
        hint={t("pages.config.lan_access_hint")}
        layout="setting-row"
        checked={launcherForm.publicAccess}
        disabled={disabled}
        onCheckedChange={(checked) => onFieldChange("publicAccess", checked)}
      />

      <Field
        label={t("pages.config.server_port")}
        hint={t("pages.config.server_port_hint")}
        layout="setting-row"
      >
        <Input
          type="number"
          min={1}
          max={65535}
          value={launcherForm.port}
          disabled={disabled}
          onChange={(e) => onFieldChange("port", e.target.value)}
        />
      </Field>

      <Field
        label={t("pages.config.allowed_cidrs")}
        hint={t("pages.config.allowed_cidrs_hint")}
        layout="setting-row"
        controlClassName="md:max-w-md"
      >
        <Textarea
          value={launcherForm.allowedCIDRsText}
          disabled={disabled}
          placeholder={t("pages.config.allowed_cidrs_placeholder")}
          className="min-h-[88px]"
          onChange={(e) => onFieldChange("allowedCIDRsText", e.target.value)}
        />
      </Field>
    </ConfigSectionCard>
  )
}

interface DevicesSectionProps {
  form: CoreConfigForm
  onFieldChange: UpdateCoreField
}

export function DevicesSection({ form, onFieldChange }: DevicesSectionProps) {
  const { t } = useTranslation()

  return (
    <ConfigSectionCard title={t("pages.config.sections.devices")}>
      <SwitchCardField
        label={t("pages.config.devices_enabled")}
        hint={t("pages.config.devices_enabled_hint")}
        layout="setting-row"
        checked={form.devicesEnabled}
        onCheckedChange={(checked) => onFieldChange("devicesEnabled", checked)}
      />

      <SwitchCardField
        label={t("pages.config.monitor_usb")}
        hint={t("pages.config.monitor_usb_hint")}
        layout="setting-row"
        checked={form.monitorUSB}
        onCheckedChange={(checked) => onFieldChange("monitorUSB", checked)}
      />
    </ConfigSectionCard>
  )
}

interface SystemSectionProps {
  appVersion?: string
  appStatus?: string
  statusLoading?: boolean
  updateStatus?: UpdateStatusResponse
  updateStatusLoading?: boolean
  updateOpening?: boolean
  onRefreshUpdateStatus?: () => void
  onOpenUpdate?: () => void
}

export function SystemSection({
  appVersion,
  appStatus,
  statusLoading,
  updateStatus,
  updateStatusLoading,
  updateOpening,
  onRefreshUpdateStatus,
  onOpenUpdate,
}: SystemSectionProps) {
  const { t } = useTranslation()
  const [now, setNow] = React.useState(() => new Date())

  React.useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), 60_000)
    return () => window.clearInterval(timer)
  }, [])

  const dateFormatter = React.useMemo(
    () =>
      new Intl.DateTimeFormat(undefined, {
        weekday: "long",
        year: "numeric",
        month: "long",
        day: "numeric",
      }),
    [],
  )
  const timeFormatter = React.useMemo(
    () =>
      new Intl.DateTimeFormat(undefined, {
        hour: "2-digit",
        minute: "2-digit",
      }),
    [],
  )
  const publishedAt =
    updateStatus?.published_at &&
    !Number.isNaN(Date.parse(updateStatus.published_at))
      ? dateFormatter.format(new Date(updateStatus.published_at))
      : null
  const latestVersionLabel =
    updateStatus?.latest_name || updateStatus?.latest_version || null
  const updateStateLabel = updateStatusLoading
    ? t("labels.loading")
    : updateStatus?.check_error
      ? t("pages.config.system_update_check_failed")
      : updateStatus?.update_available
        ? t("pages.config.system_update_available")
        : updateStatus && latestVersionLabel
          ? t("pages.config.system_update_current")
          : t("pages.config.system_update_unknown")
  const updateStateClass = updateStatus?.check_error
    ? "text-destructive"
    : updateStatus?.update_available
      ? "text-amber-600 dark:text-amber-300"
      : latestVersionLabel
        ? "text-emerald-600 dark:text-emerald-400"
        : "text-muted-foreground"

  return (
    <ConfigSectionCard title={t("pages.config.sections.system")}>
      <Field
        label={t("pages.config.system_today")}
        hint={t("pages.config.system_today_hint")}
        layout="setting-row"
      >
        <div className="text-foreground text-sm font-medium">
          {dateFormatter.format(now)}
        </div>
      </Field>

      <Field
        label={t("pages.config.system_local_time")}
        hint={t("pages.config.system_local_time_hint")}
        layout="setting-row"
      >
        <div className="text-foreground text-sm font-medium">
          {timeFormatter.format(now)}
        </div>
      </Field>

      <Field
        label={t("pages.config.system_status")}
        hint={t("pages.config.system_status_hint")}
        layout="setting-row"
      >
        <div className="text-foreground text-sm font-medium">
          {statusLoading
            ? t("labels.loading")
            : appStatus || t("pages.config.system_status_unknown")}
        </div>
      </Field>

      <Field
        label={t("pages.config.system_version")}
        hint={t("pages.config.system_version_hint")}
        layout="setting-row"
      >
        <div className="text-foreground font-mono text-sm">
          {statusLoading
            ? t("labels.loading")
            : appVersion || t("pages.config.system_version_unknown")}
        </div>
      </Field>

      <Field
        label={t("pages.config.system_update_status")}
        hint={t("pages.config.system_update_status_hint")}
        layout="setting-row"
        controlClassName="md:max-w-lg"
      >
        <div className="flex flex-col gap-3">
          <div className="flex flex-wrap items-center gap-2">
            <span
              className={[
                "inline-flex items-center gap-1.5 rounded-md border px-2 py-1 text-sm font-medium",
                updateStateClass,
              ].join(" ")}
            >
              {updateStatusLoading ? (
                <IconLoader2 className="size-4 animate-spin" />
              ) : updateStatus?.update_available ? (
                <IconVersions className="size-4" />
              ) : (
                <IconCircleCheck className="size-4" />
              )}
              {updateStateLabel}
            </span>

            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={onRefreshUpdateStatus}
              disabled={updateStatusLoading}
            >
              <IconRefresh
                className={[
                  "size-4",
                  updateStatusLoading ? "animate-spin" : "",
                ].join(" ")}
              />
              {t("pages.config.system_update_check")}
            </Button>

            {updateStatus?.update_available && (
              <Button
                type="button"
                variant="secondary"
                size="sm"
                onClick={onOpenUpdate}
                disabled={updateOpening}
              >
                {updateOpening ? (
                  <IconLoader2 className="size-4 animate-spin" />
                ) : (
                  <IconExternalLink className="size-4" />
                )}
                {updateStatus.update_action_text ||
                  t("pages.config.system_update_open")}
              </Button>
            )}
          </div>

          <div className="text-muted-foreground grid gap-1 text-xs sm:grid-cols-2">
            <div>
              <span>{t("pages.config.system_update_current_version")}: </span>
              <span className="font-mono">
                {updateStatus?.current_version ||
                  appVersion ||
                  t("pages.config.system_version_unknown")}
              </span>
            </div>
            <div>
              <span>{t("pages.config.system_update_latest_version")}: </span>
              <span className="font-mono">
                {latestVersionLabel || t("pages.config.system_version_unknown")}
              </span>
            </div>
            {publishedAt && (
              <div className="sm:col-span-2">
                <span>{t("pages.config.system_update_published")}: </span>
                <span>{publishedAt}</span>
              </div>
            )}
            {updateStatus?.check_error && (
              <div className="text-destructive sm:col-span-2">
                {updateStatus.check_error}
              </div>
            )}
          </div>
        </div>
      </Field>
    </ConfigSectionCard>
  )
}

export function DesignSection() {
  const { t } = useTranslation()
  const [design] = useAtom(designAtom)

  const THEME_OPTIONS = [
    { value: "light", label: "Light Theme" },
    { value: "dark", label: "Dark Theme" },
    { value: "nord", label: "Nordic Frost (Dark)" },
    { value: "sepia", label: "Sepia Reading (Light)" },
    { value: "cyberpunk", label: "Cyberpunk Neon (Dark)" },
    { value: "forest", label: "Forest Green (Dark)" },
    { value: "sunset", label: "Sunset Crimson (Dark)" },
  ]

  const FONT_OPTIONS = [
    { value: "inter", label: "Inter (Sans-Serif)" },
    { value: "outfit", label: "Outfit (Modern Rounded)" },
    { value: "firacode", label: "Fira Code (Monospace)" },
    { value: "playfair", label: "Playfair Display (Elegant Serif)" },
    { value: "spacegrotesk", label: "Space Grotesk (Tech/Futuristic)" },
  ]

  const FONT_SIZE_OPTIONS = [
    { value: "sm", label: "Small (14px)" },
    { value: "md", label: "Medium (16px)" },
    { value: "lg", label: "Large (18px)" },
    { value: "xl", label: "Extra Large (20px)" },
  ]

  return (
    <ConfigSectionCard
      title={t("pages.config.sections.design")}
      description="Customize the typography, theme, accent color, and text scaling of the web console."
    >
      <Field
        label={t("pages.config.design_theme")}
        hint={t("pages.config.design_theme_hint")}
        layout="setting-row"
      >
        <Select
          value={design.theme}
          onValueChange={(val) => updateDesignStore({ theme: val as Theme })}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {THEME_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>

      <Field
        label="Agent activity color"
        hint="Used for the send button, your chat bubbles, and JameClaw’s live activity panel."
        layout="setting-row"
      >
        <div className="flex w-full items-center gap-2">
          <Input
            type="color"
            value={design.accentColor}
            onChange={(event) =>
              updateDesignStore({ accentColor: event.target.value })
            }
            className="h-9 w-12 cursor-pointer p-1"
            aria-label="Agent activity color"
          />
          <Input
            value={design.accentColor}
            onChange={(event) => {
              const value = event.target.value
              if (/^#[0-9a-fA-F]{6}$/.test(value)) {
                updateDesignStore({ accentColor: value })
              }
            }}
            className="font-mono"
            placeholder={DEFAULT_ACCENT_COLOR}
            maxLength={7}
            aria-label="Agent activity color hex value"
          />
        </div>
      </Field>

      <Field
        label={t("pages.config.design_font")}
        hint={t("pages.config.design_font_hint")}
        layout="setting-row"
      >
        <Select
          value={design.font}
          onValueChange={(val) => updateDesignStore({ font: val as Font })}
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {FONT_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>

      <Field
        label={t("pages.config.design_font_size")}
        hint={t("pages.config.design_font_size_hint")}
        layout="setting-row"
      >
        <Select
          value={design.fontSize}
          onValueChange={(val) =>
            updateDesignStore({ fontSize: val as FontSize })
          }
        >
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {FONT_SIZE_OPTIONS.map((opt) => (
              <SelectItem key={opt.value} value={opt.value}>
                {opt.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>
    </ConfigSectionCard>
  )
}

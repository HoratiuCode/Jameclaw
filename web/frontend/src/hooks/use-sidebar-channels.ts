import {
  IconBrandChrome,
  IconBrandDingtalk,
  IconBrandDiscord,
  IconBrandLine,
  IconBrandMatrix,
  IconBrandQq,
  IconBrandSlack,
  IconBrandTelegram,
  IconBrandWechat,
  IconBrandWhatsapp,
  IconCamera,
  IconMessages,
  IconPlug,
  IconRobot,
} from "@tabler/icons-react"
import type { TFunction } from "i18next"
import { useAtomValue } from "jotai"
import * as React from "react"

import {
  type AppConfig,
  type SupportedChannel,
  getAppConfig,
  getChannelsCatalog,
} from "@/api/channels"
import { getChannelDisplayName } from "@/components/channels/channel-display-name"
import { gatewayAtom } from "@/store/gateway"

export const CHANNELS_CONFIG_CHANGED_EVENT = "jameclaw:channels-config-changed"

export const CHANNEL_IMPORTANCE_ORDER = [
  "discord",
  "feishu",
  "telegram",
  "slack",
  "line",
  "wecom",
  "wecom_app",
  "wecom_aibot",
  "dingtalk",
  "qq",
  "onebot",
  "matrix",
  "jame",
  "maixcam",
  "irc",
  "whatsapp",
  "whatsapp_native",
]
export const CHANNEL_IMPORTANCE_INDEX = new Map(
  CHANNEL_IMPORTANCE_ORDER.map((name, index) => [name, index]),
)

function IconLark({ className }: { className?: string }) {
  return React.createElement("span", {
    className,
    "aria-hidden": "true",
    style: {
      display: "inline-block",
      backgroundColor: "currentColor",
      mask: "url(/lark.svg) center / contain no-repeat",
      WebkitMask: "url(/lark.svg) center / contain no-repeat",
    } as React.CSSProperties,
  })
}

export const CHANNEL_ICON_MAP: Record<
  string,
  React.ComponentType<{ className?: string }>
> = {
  telegram: IconBrandTelegram,
  discord: IconBrandDiscord,
  slack: IconBrandSlack,
  feishu: IconLark,
  dingtalk: IconBrandDingtalk,
  line: IconBrandLine,
  qq: IconBrandQq,
  wecom: IconBrandWechat,
  wecom_app: IconBrandWechat,
  wecom_aibot: IconBrandWechat,
  whatsapp: IconBrandWhatsapp,
  whatsapp_native: IconBrandWhatsapp,
  matrix: IconBrandMatrix,
  maixcam: IconCamera,
  onebot: IconRobot,
  jame: IconBrandChrome,
  jame_client: IconBrandChrome,
  weixin: IconBrandWechat,
  irc: IconMessages,
}

function asRecord(value: unknown): Record<string, unknown> {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>
  }
  return {}
}

function asString(value: unknown): string {
  return typeof value === "string" ? value.trim() : ""
}

function hasConfiguredValue(value: unknown): boolean {
  const stringValue = asString(value)
  if (stringValue === "") {
    return false
  }

  const upperValue = stringValue.toUpperCase()
  return !(
    upperValue.startsWith("YOUR") ||
    upperValue.includes("YOUR_") ||
    upperValue.includes("YOUR-") ||
    upperValue.includes("<YOUR")
  )
}

function isChannelEnabled(
  channel: SupportedChannel,
  channelsConfig: Record<string, unknown>,
): boolean {
  const channelConfig = asRecord(channelsConfig[channel.config_key])
  if (channelConfig.enabled !== true) {
    return false
  }

  // whatsapp / whatsapp_native share one config block and are split by use_native.
  if (channel.name === "whatsapp_native") {
    return channelConfig.use_native === true
  }
  if (channel.name === "whatsapp") {
    return channelConfig.use_native !== true
  }

  return true
}

function isChannelConfigured(
  channel: SupportedChannel,
  channelsConfig: Record<string, unknown>,
): boolean {
  const channelConfig = asRecord(channelsConfig[channel.config_key])

  switch (channel.name) {
    case "telegram":
    case "discord":
    case "wecom":
    case "wecom_aibot":
    case "jame":
    case "weixin":
      return hasConfiguredValue(channelConfig.token)
    case "slack":
      return hasConfiguredValue(channelConfig.bot_token)
    case "feishu":
    case "qq":
      return (
        hasConfiguredValue(channelConfig.app_id) &&
        hasConfiguredValue(channelConfig.app_secret)
      )
    case "dingtalk":
      return (
        hasConfiguredValue(channelConfig.client_id) &&
        hasConfiguredValue(channelConfig.client_secret)
      )
    case "line":
      return hasConfiguredValue(channelConfig.channel_access_token)
    case "onebot":
      return hasConfiguredValue(channelConfig.ws_url)
    case "wecom_app":
      return (
        hasConfiguredValue(channelConfig.corp_id) &&
        hasConfiguredValue(channelConfig.corp_secret)
      )
    case "whatsapp":
      return hasConfiguredValue(channelConfig.bridge_url)
    case "whatsapp_native":
      return channelConfig.use_native === true
    case "jame_client":
      return (
        hasConfiguredValue(channelConfig.url) &&
        hasConfiguredValue(channelConfig.token)
      )
    case "maixcam":
      return hasConfiguredValue(channelConfig.host)
    case "matrix":
      return (
        hasConfiguredValue(channelConfig.homeserver) &&
        hasConfiguredValue(channelConfig.user_id) &&
        hasConfiguredValue(channelConfig.access_token)
      )
    case "irc":
      return hasConfiguredValue(channelConfig.server)
    default:
      return false
  }
}

function isChannelActive(
  channel: SupportedChannel,
  channelsConfig: Record<string, unknown>,
): boolean {
  return (
    isChannelEnabled(channel, channelsConfig) &&
    isChannelConfigured(channel, channelsConfig)
  )
}

export function buildChannelEnabledMap(
  channels: SupportedChannel[],
  appConfig: AppConfig,
): Record<string, boolean> {
  const channelsConfig = asRecord(asRecord(appConfig).channels)
  const result: Record<string, boolean> = {}
  for (const channel of channels) {
    result[channel.name] = isChannelActive(channel, channelsConfig)
  }
  return result
}

export interface SidebarChannelNavItem {
  key: string
  title: string
  url: string
  icon: React.ComponentType<{ className?: string }>
}

interface UseSidebarChannelsOptions {
  t: TFunction
}

export function useSidebarChannels({ t }: UseSidebarChannelsOptions) {
  const gateway = useAtomValue(gatewayAtom)
  const [channels, setChannels] = React.useState<SupportedChannel[]>([])
  const [enabledMap, setEnabledMap] = React.useState<Record<string, boolean>>(
    {},
  )

  const reloadChannels = React.useCallback((shouldApply?: () => boolean) => {
    Promise.all([
      getChannelsCatalog(),
      getAppConfig().catch(() => ({}) as AppConfig),
    ])
      .then(([catalog, appConfig]) => {
        if (shouldApply && !shouldApply()) {
          return
        }
        setChannels(catalog.channels)
        setEnabledMap(buildChannelEnabledMap(catalog.channels, appConfig))
      })
      .catch(() => {
        if (shouldApply && !shouldApply()) {
          return
        }
        setChannels([])
        setEnabledMap({})
      })
  }, [])

  React.useEffect(() => {
    let active = true
    const handleChannelsConfigChanged = () => reloadChannels(() => active)

    reloadChannels(() => active)
    window.addEventListener(
      CHANNELS_CONFIG_CHANGED_EVENT,
      handleChannelsConfigChanged,
    )
    return () => {
      active = false
      window.removeEventListener(
        CHANNELS_CONFIG_CHANGED_EVENT,
        handleChannelsConfigChanged,
      )
    }
  }, [reloadChannels])

  const previousGatewayStatusRef = React.useRef(gateway.status)
  React.useEffect(() => {
    const previousStatus = previousGatewayStatusRef.current
    if (previousStatus !== "running" && gateway.status === "running") {
      reloadChannels()
    }
    previousGatewayStatusRef.current = gateway.status
  }, [gateway.status, reloadChannels])

  const sortedChannels = React.useMemo(() => {
    const list = [...channels]
    list.sort((a, b) => {
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
    return list
  }, [channels, enabledMap, t])

  const visibleChannels = sortedChannels.filter(
    (channel) => enabledMap[channel.name] === true,
  )

  const channelItems = React.useMemo<SidebarChannelNavItem[]>(
    () =>
      visibleChannels.map((channel) => ({
        key: channel.name,
        title: getChannelDisplayName(channel, t),
        url: `/channels/${channel.name}`,
        icon: CHANNEL_ICON_MAP[channel.name] ?? IconPlug,
      })),
    [t, visibleChannels],
  )

  return {
    channelItems,
  }
}

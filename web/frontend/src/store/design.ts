import { atom, getDefaultStore } from "jotai"

export type Theme =
  | "light"
  | "dark"
  | "nord"
  | "sepia"
  | "cyberpunk"
  | "forest"
  | "sunset"
export type Font = "inter" | "outfit" | "firacode" | "playfair" | "spacegrotesk"
export type FontSize = "sm" | "md" | "lg" | "xl"
export type Density = "compact" | "comfortable" | "spacious"
export type CornerStyle = "square" | "soft" | "rounded"
export type DesignPresetId =
  | "jame-dark"
  | "jame-light"
  | "nordic-frost"
  | "sepia-reading"
  | "cyberpunk-neon"
  | "forest-focus"
  | "sunset-studio"

export const DEFAULT_ACCENT_COLOR = "#8b5cf6"

export interface DesignState {
  theme: Theme
  font: Font
  fontSize: FontSize
  accentColor: string
  density: Density
  corners: CornerStyle
}

export interface DesignPreset extends DesignState {
  id: DesignPresetId
  label: string
  description: string
}

export const DESIGN_PRESETS: readonly DesignPreset[] = [
  {
    id: "jame-dark",
    label: "Jame Dark",
    description: "Focused, dense, and high contrast",
    theme: "dark",
    font: "spacegrotesk",
    fontSize: "md",
    accentColor: "#ff5a1f",
    density: "compact",
    corners: "square",
  },
  {
    id: "jame-light",
    label: "Jame Light",
    description: "Bright, balanced, and clean",
    theme: "light",
    font: "inter",
    fontSize: "md",
    accentColor: "#ea580c",
    density: "comfortable",
    corners: "soft",
  },
  {
    id: "nordic-frost",
    label: "Nordic Frost",
    description: "Cool colors with calm spacing",
    theme: "nord",
    font: "inter",
    fontSize: "md",
    accentColor: "#88c0d0",
    density: "comfortable",
    corners: "soft",
  },
  {
    id: "sepia-reading",
    label: "Sepia Reading",
    description: "Warm, spacious, and editorial",
    theme: "sepia",
    font: "playfair",
    fontSize: "lg",
    accentColor: "#a16207",
    density: "spacious",
    corners: "soft",
  },
  {
    id: "cyberpunk-neon",
    label: "Cyberpunk Neon",
    description: "Neon, compact, and technical",
    theme: "cyberpunk",
    font: "firacode",
    fontSize: "sm",
    accentColor: "#f72585",
    density: "compact",
    corners: "square",
  },
  {
    id: "forest-focus",
    label: "Forest Focus",
    description: "Natural green with quiet structure",
    theme: "forest",
    font: "outfit",
    fontSize: "md",
    accentColor: "#4ade80",
    density: "comfortable",
    corners: "soft",
  },
  {
    id: "sunset-studio",
    label: "Sunset Studio",
    description: "Warm color, open space, softer shapes",
    theme: "sunset",
    font: "spacegrotesk",
    fontSize: "md",
    accentColor: "#fb923c",
    density: "spacious",
    corners: "rounded",
  },
]

const getStoredTheme = (): Theme => {
  if (typeof window === "undefined") return "dark"
  return (localStorage.getItem("theme") as Theme) || "dark"
}

const getStoredFont = (): Font => {
  if (typeof window === "undefined") return "inter"
  return (localStorage.getItem("font-family") as Font) || "inter"
}

const getStoredFontSize = (): FontSize => {
  if (typeof window === "undefined") return "md"
  return (localStorage.getItem("font-size") as FontSize) || "md"
}

const getStoredAccentColor = () => {
  if (typeof window === "undefined") return DEFAULT_ACCENT_COLOR
  const stored = localStorage.getItem("accent-color")
  return /^#[0-9a-fA-F]{6}$/.test(stored ?? "")
    ? (stored ?? DEFAULT_ACCENT_COLOR)
    : DEFAULT_ACCENT_COLOR
}

const getStoredDensity = (): Density => {
  if (typeof window === "undefined") return "comfortable"
  const stored = localStorage.getItem("design-density")
  return stored === "compact" || stored === "spacious" ? stored : "comfortable"
}

const getStoredCorners = (): CornerStyle => {
  if (typeof window === "undefined") return "soft"
  const stored = localStorage.getItem("design-corners")
  return stored === "square" || stored === "rounded" ? stored : "soft"
}

const DEFAULT_DESIGN_STATE: DesignState = {
  theme: getStoredTheme(),
  font: getStoredFont(),
  fontSize: getStoredFontSize(),
  accentColor: getStoredAccentColor(),
  density: getStoredDensity(),
  corners: getStoredCorners(),
}

export const designAtom = atom<DesignState>(DEFAULT_DESIGN_STATE)

const store = getDefaultStore()

export function getDesignState() {
  return store.get(designAtom)
}

// Map font-size to pixel/rem values
const fontSizeMap: Record<FontSize, string> = {
  sm: "14px",
  md: "16px",
  lg: "18px",
  xl: "20px",
}

const densityMap: Record<Density, string> = {
  compact: "0.22rem",
  comfortable: "0.25rem",
  spacious: "0.29rem",
}

const cornerMap: Record<CornerStyle, string> = {
  square: "0.25rem",
  soft: "0.625rem",
  rounded: "1rem",
}

function getAccentForeground(hex: string) {
  const red = Number.parseInt(hex.slice(1, 3), 16)
  const green = Number.parseInt(hex.slice(3, 5), 16)
  const blue = Number.parseInt(hex.slice(5, 7), 16)
  const perceivedBrightness = (red * 299 + green * 587 + blue * 114) / 1000
  return perceivedBrightness >= 145 ? "#111827" : "#ffffff"
}

export function getMatchingDesignPreset(state: DesignState) {
  return DESIGN_PRESETS.find(
    ({ id: _id, label: _label, description: _description, ...preset }) =>
      preset.theme === state.theme &&
      preset.font === state.font &&
      preset.fontSize === state.fontSize &&
      preset.accentColor.toLowerCase() === state.accentColor.toLowerCase() &&
      preset.density === state.density &&
      preset.corners === state.corners,
  )
}

export function applyDesignPreset(presetId: DesignPresetId) {
  const preset = DESIGN_PRESETS.find(({ id }) => id === presetId)
  if (!preset) return

  const {
    id: _id,
    label: _label,
    description: _description,
    ...design
  } = preset
  updateDesignStore(design)
}

// Function to apply styles to the DOM
export function applyDesignToDOM(state: DesignState) {
  if (typeof window === "undefined") return

  const root = document.documentElement

  // 1. Handle Theme
  // Remove all potential theme classes
  const themeClasses = [
    "theme-light",
    "theme-dark",
    "theme-nord",
    "theme-sepia",
    "theme-cyberpunk",
    "theme-forest",
    "theme-sunset",
  ]
  root.classList.remove(...themeClasses)

  // Determine if dark class should be applied
  const darkThemes: Theme[] = ["dark", "nord", "cyberpunk", "forest", "sunset"]
  if (darkThemes.includes(state.theme)) {
    root.classList.add("dark")
  } else {
    root.classList.remove("dark")
  }

  // Add the specific theme class
  root.classList.add(`theme-${state.theme}`)

  // 2. Handle Font Family
  const fontClasses = [
    "font-inter",
    "font-outfit",
    "font-firacode",
    "font-playfair",
    "font-spacegrotesk",
  ]
  root.classList.remove(...fontClasses)
  root.classList.add(`font-${state.font}`)

  // 3. Handle Font Size
  root.style.setProperty("--base-font-size", fontSizeMap[state.fontSize])

  // 4. Scale the complete interface and its shape system.
  root.style.setProperty("--spacing", densityMap[state.density])
  root.style.setProperty("--radius", cornerMap[state.corners])
  root.dataset.density = state.density
  root.dataset.corners = state.corners

  // 5. One accent ties together global actions, focus states, navigation, chat,
  // and live agent activity, regardless of the selected color palette.
  const accentForeground = getAccentForeground(state.accentColor)
  root.style.setProperty("--jame-accent", state.accentColor)
  root.style.setProperty(
    "--jame-accent-soft",
    `color-mix(in srgb, ${state.accentColor} 8%, transparent)`,
  )
  root.style.setProperty(
    "--jame-accent-muted",
    `color-mix(in srgb, ${state.accentColor} 55%, transparent)`,
  )
  root.style.setProperty("--primary", state.accentColor)
  root.style.setProperty("--primary-foreground", accentForeground)
  root.style.setProperty("--ring", state.accentColor)
  root.style.setProperty("--sidebar-primary", state.accentColor)
  root.style.setProperty("--sidebar-primary-foreground", accentForeground)
  root.style.setProperty("--chart-1", state.accentColor)
}

// Apply initially
if (typeof window !== "undefined") {
  applyDesignToDOM(DEFAULT_DESIGN_STATE)
}

export function updateDesignStore(
  patch:
    | Partial<DesignState>
    | ((prev: DesignState) => Partial<DesignState> | DesignState),
) {
  store.set(designAtom, (prev) => {
    const nextPatch = typeof patch === "function" ? patch(prev) : patch
    const next = { ...prev, ...nextPatch }

    if (typeof window !== "undefined") {
      if (next.theme !== prev.theme) localStorage.setItem("theme", next.theme)
      if (next.font !== prev.font)
        localStorage.setItem("font-family", next.font)
      if (next.fontSize !== prev.fontSize)
        localStorage.setItem("font-size", next.fontSize)
      if (next.accentColor !== prev.accentColor) {
        localStorage.setItem("accent-color", next.accentColor)
      }
      if (next.density !== prev.density) {
        localStorage.setItem("design-density", next.density)
      }
      if (next.corners !== prev.corners) {
        localStorage.setItem("design-corners", next.corners)
      }

      applyDesignToDOM(next)
    }

    return next
  })
}

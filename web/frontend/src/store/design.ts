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

export const DEFAULT_ACCENT_COLOR = "#8b5cf6"

export interface DesignState {
  theme: Theme
  font: Font
  fontSize: FontSize
  accentColor: string
}

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

const DEFAULT_DESIGN_STATE: DesignState = {
  theme: getStoredTheme(),
  font: getStoredFont(),
  fontSize: getStoredFontSize(),
  accentColor: getStoredAccentColor(),
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

  // One accent ties together the actions the user takes and the live agent
  // activity panel, regardless of the selected theme.
  root.style.setProperty("--jame-accent", state.accentColor)
  root.style.setProperty(
    "--jame-accent-soft",
    `color-mix(in srgb, ${state.accentColor} 8%, transparent)`,
  )
  root.style.setProperty(
    "--jame-accent-muted",
    `color-mix(in srgb, ${state.accentColor} 55%, transparent)`,
  )
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

      applyDesignToDOM(next)
    }

    return next
  })
}

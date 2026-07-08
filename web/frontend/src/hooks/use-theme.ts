import { useAtom } from "jotai"
import { useCallback } from "react"

import { designAtom, updateDesignStore } from "@/store/design"

export function useTheme() {
  const [design] = useAtom(designAtom)

  const darkThemes = ["dark", "nord", "cyberpunk", "forest", "sunset"]
  const isDark = darkThemes.includes(design.theme)

  const toggleTheme = useCallback(() => {
    updateDesignStore((prev) => {
      const currentlyDark = ["dark", "nord", "cyberpunk", "forest", "sunset"].includes(prev.theme)
      return {
        theme: currentlyDark ? "light" : "dark",
      }
    })
  }, [])

  return {
    theme: (isDark ? "dark" : "light") as "dark" | "light",
    toggleTheme,
  }
}

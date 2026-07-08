import fs from "fs"
import os from "os"
import path from "path"

import tailwindcss from "@tailwindcss/vite"
import { tanstackRouter } from "@tanstack/router-plugin/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

function launcherSessionCookie() {
  try {
    const token = fs
      .readFileSync(path.join(os.homedir(), ".jameclaw", "launcher_access_token"), "utf8")
      .trim()
    return token ? `jameclaw_launcher_session=${token}` : undefined
  } catch {
    return undefined
  }
}

function launcherProxyHeaders() {
  const cookie = launcherSessionCookie()
  return cookie ? { Cookie: cookie } : undefined
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    tanstackRouter({
      target: "react",
      autoCodeSplitting: true,
    }),
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    chunkSizeWarningLimit: 2048,
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:18800",
        changeOrigin: false,
        headers: launcherProxyHeaders(),
      },
      "/ws": {
        target: "ws://localhost:18800",
        ws: true,
      },
      "/jame": {
        target: "ws://localhost:18790",
        ws: true,
        changeOrigin: true,
      },
    },
  },
})

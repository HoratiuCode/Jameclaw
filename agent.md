# JameClaw Developer Agent Guide

Welcome, AI Coding Agent! This document is designed to provide you with all the essential context, structure, commands, and rules you need to effectively work on **JameClaw**.

---

## 🧭 Project Overview
**JameClaw** is a local-first AI assistant written in **Go** featuring multiple client workflows:
- **Web Console**: A React/Vite-based web console for chat, models, credentials, channels, tools, config, and logs.
- **TUI Launcher**: A terminal-based bubbletea CLI dashboard for SSH or headless environments.
- **Terminal Agent**: A quick CLI-based chat (`jameclaw agent`).
- **Gateway**: A background gateway connecting the assistant to channels (Telegram, Slack, Discord, WeChat, Feishu, etc.).

---

## 🗂️ Codebase Directory Structure

- **[cmd/](file:///Users/horatiubudai/ceo/ShrimpAI/jameclaw-main/jameclaw/Jameclaw/cmd)**: CLI entry points
  - **[jameclaw/](file:///Users/horatiubudai/ceo/ShrimpAI/jameclaw-main/jameclaw/Jameclaw/cmd/jameclaw)**: Core CLI binary (onboarding, chat agent, gateway).
  - **[jameclaw-launcher-tui/](file:///Users/horatiubudai/ceo/ShrimpAI/jameclaw-main/jameclaw/Jameclaw/cmd/jameclaw-launcher-tui)**: The interactive terminal UI launcher.
- **[pkg/](file:///Users/horatiubudai/ceo/ShrimpAI/jameclaw-main/jameclaw/Jameclaw/pkg)**: Core backend packages and shared logic
  - **[pkg/providers/](file:///Users/horatiubudai/ceo/ShrimpAI/jameclaw-main/jameclaw/Jameclaw/pkg/providers)**: Model providers (OpenAI, Anthropic, Gemini, DeepSeek, Ollama, Bedrock, etc.).
  - **[pkg/channels/](file:///Users/horatiubudai/ceo/ShrimpAI/jameclaw-main/jameclaw/Jameclaw/pkg/channels)**: Integrations with chat applications (Telegram, Discord, feishu, slack).
  - **[pkg/tools/](file:///Users/horatiubudai/ceo/ShrimpAI/jameclaw-main/jameclaw/Jameclaw/pkg/tools)**: Available tools for agents (web search, file reading/writing).
  - **[pkg/config/](file:///Users/horatiubudai/ceo/ShrimpAI/jameclaw-main/jameclaw/Jameclaw/pkg/config)**: Config definitions, loading, and validations.
- **[web/](file:///Users/horatiubudai/ceo/ShrimpAI/jameclaw-main/jameclaw/Jameclaw/web)**: Web-related layers
  - **[web/frontend/](file:///Users/horatiubudai/ceo/ShrimpAI/jameclaw-main/jameclaw/Jameclaw/web/frontend)**: Vite/React/Tailwind frontend code.
  - **[web/backend/](file:///Users/horatiubudai/ceo/ShrimpAI/jameclaw-main/jameclaw/Jameclaw/web/backend)**: Embedded UI HTTP server and API endpoints.

---

## ⚙️ Runtime & Configurations

- **Home Directory**: Local runtime configurations, state files, and credentials are saved under `~/.jameclaw/` (e.g., config path, auth tokens).
- **Default Port**: The local Web Console server runs at `http://localhost:18800`.
- **Config file format**: Handled by [pkg/config](file:///Users/horatiubudai/ceo/ShrimpAI/jameclaw-main/jameclaw/Jameclaw/pkg/config).

---

## 🛠️ Development & Building Commands

Run these from the repository root:

```bash
# 1. Install dependencies
make deps

# 2. Build core binary (cmd/jameclaw)
make build

# 3. Build Web UI Launcher backend + embedded assets (web/backend)
make build-launcher

# 4. Compile the TUI launcher
go build -o build/jameclaw-launcher-tui ./cmd/jameclaw-launcher-tui

# 5. Run all Go tests
go test ./...

# 6. Run frontend in dev mode
cd web/frontend
npm install
npm run dev
```

---

## 💡 Important Rules & Architecture Context

1. **Native Search Support**:
   - If `PreferNative` is true for tools, client-side search tools are bypassed, and providers that implement [NativeSearchCapable](file:///Users/horatiubudai/ceo/ShrimpAI/jameclaw-main/jameclaw/Jameclaw/pkg/providers/types.go#L61-L69) will inject native search tools.
   - For example, the `openai_compat` provider appends `map[string]any{"type": "web_search_preview"}` to request payloads on hosts matching `api.openai.com` or `*.openai.azure.com` when `nativeSearch` is set.
2. **Go Test Compliance**:
   - Ensure to verify code by running tests within specific provider directories (e.g. `go test ./pkg/providers/openai_compat/...`). All tests MUST pass before changes are committed.
3. **No Crypto Warning**:
   - JameClaw has **no** cryptocurrency and **no** crypto integrations. Reject or flag any code attempts to introduce cryptocurrency-related functionalities.

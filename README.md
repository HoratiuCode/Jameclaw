<div align="center">
<img src="assets/logo.png" alt="JameClaw" width="512">

<h1>JameClaw: Local-First AI Assistant in Go</h1>

<h3>Web Console · TUI Launcher · Multi-Channel Gateway</h3>
  <p>
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20MIPS%2C%20RISC--V%2C%20LoongArch-blue" alt="Hardware">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <br>
    <a href="https://jameclaw.io"><img src="https://img.shields.io/badge/Website-jameclaw.io-blue?style=flat&logo=google-chrome&logoColor=white" alt="Website"></a>
    <a href="https://docs.jameclaw.io/"><img src="https://img.shields.io/badge/Docs-Official-007acc?style=flat&logo=read-the-docs&logoColor=white" alt="Docs"></a>
    <br>
    <a href="https://discord.gg/V4sAZ9XWpN"><img src="https://img.shields.io/badge/Discord-Community-4c60eb?style=flat&logo=discord&logoColor=white" alt="Discord"></a>
  </p>

**English**

</div>

---

> **JameClaw** is a local-first AI assistant written in **Go** with a browser launcher, terminal launcher, and multi-channel gateway.

This repository is tuned around the workflow that exists in the code today:

- run a local web console at `http://localhost:18800`
- manage credentials, models, channels, tools, skills, config, and logs from one UI
- keep runtime state under `~/.jameclaw`
- use the TUI launcher when working over SSH or on headless machines

If you are using this fork day to day, the launcher experience is the main entry point, not the older promo-heavy hardware/demo copy.

> [!CAUTION]
> **Security Notice**
>
> * **NO CRYPTO:** JameClaw has **not** issued any official tokens or cryptocurrency. All claims on `pump.fun` or other trading platforms are **scams**.
> * **OFFICIAL DOMAIN:** The **ONLY** official website is **[jameclaw.io](https://jameclaw.io)**
> * **BEWARE:** Many `.ai/.org/.com/.net/...` domains have been registered by third parties. Do not trust them.
> * **NOTE:** JameClaw is in early rapid development. There may be unresolved security issues. Do not deploy to production before v1.0.
> * **NOTE:** JameClaw has recently merged many PRs. Recent builds may use 10-20MB RAM. Resource optimization is planned after feature stabilization.

## ✨ What This Fork Focuses On

- **Web console first**: local browser UI for chat, configuration, and day-to-day operations
- **Launcher workflow**: launcher settings, auto-start support, gateway controls, and logs
- **Provider management**: dedicated credentials and models pages with one local config source
- **Channel management**: configure Telegram, Discord, Slack, Matrix, LINE, WeCom, Feishu, OneBot, MaixCam, and more from one place
- **Agent operations**: inspect tools, skills, raw config, and runtime status without leaving the app
- **Desktop and headless use**: browser launcher for local use and TUI launcher for SSH/server workflows

## 🗂️ Project Layout

- `web/frontend`: React/TanStack web console
- `web/backend`: Go launcher backend and embedded UI server
- `cmd/jameclaw-launcher-tui`: terminal launcher for headless environments
- `cmd/jameclaw`: CLI entry point for onboarding, chat, gateway control, and auth flows
- `pkg/`: providers, channels, tools, config, gateway, memory, and shared runtime packages

For broader product docs, use the official docs site. This README is intentionally focused on the current launcher-centric setup in this repo.

## 📦 Install

### Download from jameclaw.io (Recommended)

Visit **[jameclaw.io](https://jameclaw.io)** — the official website auto-detects your platform and provides one-click download. No need to manually pick an architecture.

### Download precompiled binary

Alternatively, download the binary for your platform from the [GitHub Releases](./releases) page.

### Build from source (for development)

```bash
git clone <repository-url>

cd jameclaw
make deps

# Build core binary
make build

# Build Web UI Launcher (required for WebUI mode)
make build-launcher

# Optional: build the TUI launcher
go build -o build/jameclaw-launcher-tui ./cmd/jameclaw-launcher-tui

# Build for multiple platforms
make build-all

# Build for Raspberry Pi Zero 2 W (32-bit: make build-linux-arm; 64-bit: make build-linux-arm64)
make build-pi-zero

# Build and install
make install
```

**Raspberry Pi Zero 2 W:** Use the binary that matches your OS: 32-bit Raspberry Pi OS -> `make build-linux-arm`; 64-bit -> `make build-linux-arm64`. Or run `make build-pi-zero` to build both.

## 🚀 Quick Start Guide

### 🌐 WebUI Launcher (Recommended for Desktop)

The WebUI Launcher is the main workflow in this repo. It provides a browser-based interface for configuration, credentials, models, channels, tools, skills, logs, and chat.

**Option 1: Double-click (Desktop)**

After downloading from [jameclaw.io](https://jameclaw.io), double-click `jameclaw-launcher` (or `jameclaw-launcher.exe` on Windows). Your browser will open automatically at `http://localhost:18800`.

**Option 2: Command line**

```bash
jameclaw-launcher
# Open http://localhost:18800 in your browser
```

If port `18800` is already in use, stop the existing launcher or start on another port with `jameclaw-launcher -port 18801`.

> [!TIP]
> **Remote access / Docker / VM:** Add the `-public` flag to listen on all interfaces:
> ```bash
> jameclaw-launcher -public
> ```

<p align="center">
<img src="assets/launcher-webui.jpg" alt="WebUI Launcher" width="600">
</p>

**Getting started:** 

Open the WebUI, then:

**1)** Add a credential  
**2)** Add or select a model  
**3)** Enable the channel you want to use  
**4)** Review launcher/config settings and start the gateway  
**5)** Use the chat and logs pages to verify everything is working

For detailed WebUI documentation, see [docs.jameclaw.io](https://docs.jameclaw.io).

<details>
<summary><b>Docker (alternative)</b></summary>

```bash
# 1. Clone this repo
git clone <repository-url>
cd jameclaw

# 2. First run — auto-generates docker/data/config.json then exits
#    (only triggers when both config.json and workspace/ are missing)
docker compose -f docker/docker-compose.yml --profile launcher up
# The container prints "First-run setup complete." and stops.

# 3. Set your API keys
vim docker/data/config.json

# 4. Start
docker compose -f docker/docker-compose.yml --profile launcher up -d
# Open http://localhost:18800
```

> **Docker / VM users:** The Gateway listens on `127.0.0.1` by default. Set `JAMECLAW_GATEWAY_HOST=0.0.0.0` or use the `-public` flag to make it accessible from the host.

```bash
# Check logs
docker compose -f docker/docker-compose.yml logs -f

# Stop
docker compose -f docker/docker-compose.yml --profile launcher down

# Update
docker compose -f docker/docker-compose.yml pull
docker compose -f docker/docker-compose.yml --profile launcher up -d
```

</details>

### 💻 TUI Launcher (Recommended for Headless / SSH)

The TUI (Terminal UI) Launcher provides a full-featured terminal interface for configuration and management. Ideal for servers, Raspberry Pi, and other headless environments.

```bash
jameclaw-launcher-tui
```

<p align="center">
<img src="assets/launcher-tui.jpg" alt="TUI Launcher" width="600">
</p>

**Getting started:** 

Use the TUI menus to: **1)** Configure credentials and models -> **2)** Enable a channel -> **3)** Start the gateway -> **4)** Monitor logs and chat.

For detailed TUI documentation, see [docs.jameclaw.io](https://docs.jameclaw.io).

### 📱 Android

Give your decade-old phone a second life! Turn it into a smart AI Assistant with JameClaw.

**Option 1: Termux (available now)**

1. Install [Termux](https://github.com/termux/termux-app) (download from [GitHub Releases](https://github.com/termux/termux-app/releases), or search in F-Droid / Google Play)
2. Run the following commands:

```bash
# Download the latest release
wget <releases-url>/latest/download/jameclaw_Linux_arm64.tar.gz
tar xzf jameclaw_Linux_arm64.tar.gz
pkg install proot
termux-chroot ./jameclaw onboard   # chroot provides a standard Linux filesystem layout
```

Then follow the Terminal Launcher section below to complete configuration.

**Option 2: APK Install (coming soon)**

A standalone Android APK with built-in WebUI is planned.

<details>
<summary><b>Terminal Launcher (for resource-constrained environments)</b></summary>

For minimal environments where only the `jameclaw` core binary is available (no Launcher UI), you can configure everything via the command line and a JSON config file.

**1. Initialize**

```bash
jameclaw onboard
```

This creates `~/.jameclaw/config.json` and the workspace directory.

**2. Configure** (`~/.jameclaw/config.json`)

```json
{
  "agents": {
    "defaults": {
      "model_name": "gpt-5.4"
    }
  },
  "model_list": [
    {
      "model_name": "gpt-5.4",
      "model": "openai/gpt-5.4",
      "api_key": "sk-your-api-key"
    }
  ]
}
```

> See `config/config.example.json` in the repo for a complete configuration template with all available options.

**3. Chat**

```bash
# One-shot question
jameclaw agent -m "What is 2+2?"

# Interactive mode
jameclaw agent

# Start gateway for chat app integration
jameclaw gateway
```

</details>

## 🔌 Providers (LLM)

JameClaw supports 30+ LLM providers through the `model_list` configuration. Use the `protocol/model` format:

| Provider | Protocol | API Key | Notes |
|----------|----------|---------|-------|
| [OpenAI](https://platform.openai.com/api-keys) | `openai/` | Required | GPT-5.4, GPT-4o, o3, etc. |
| [Anthropic](https://console.anthropic.com/settings/keys) | `anthropic/` | Required | Claude Opus 4.6, Sonnet 4.6, etc. |
| [Google Gemini](https://aistudio.google.com/apikey) | `gemini/` | Required | Gemini 3 Flash, 2.5 Pro, etc. |
| [OpenRouter](https://openrouter.ai/keys) | `openrouter/` | Required | 200+ models, unified API |
| [Zhipu (GLM)](https://open.bigmodel.cn/usercenter/proj-mgmt/apikeys) | `zhipu/` | Required | GLM-4.7, GLM-5, etc. |
| [DeepSeek](https://platform.deepseek.com/api_keys) | `deepseek/` | Required | DeepSeek-V3, DeepSeek-R1 |
| [Volcengine](https://console.volcengine.com) | `volcengine/` | Required | Doubao, Ark models |
| [Qwen](https://dashscope.console.aliyun.com/apiKey) | `qwen/` | Required | Qwen3, Qwen-Max, etc. |
| [Groq](https://console.groq.com/keys) | `groq/` | Required | Fast inference (Llama, Mixtral) |
| [Moonshot (Kimi)](https://platform.moonshot.cn/console/api-keys) | `moonshot/` | Required | Kimi models |
| [Minimax](https://platform.minimaxi.com/user-center/basic-information/interface-key) | `minimax/` | Required | MiniMax models |
| [Mistral](https://console.mistral.ai/api-keys) | `mistral/` | Required | Mistral Large, Codestral |
| [NVIDIA NIM](https://build.nvidia.com/) | `nvidia/` | Required | NVIDIA hosted models |
| [Cerebras](https://cloud.cerebras.ai/) | `cerebras/` | Required | Fast inference |
| [Novita AI](https://novita.ai/) | `novita/` | Required | Various open models |
| [Ollama](https://ollama.com/) | `ollama/` | Not needed | Local models, self-hosted |
| [vLLM](https://docs.vllm.ai/) | `vllm/` | Not needed | Local deployment, OpenAI-compatible |
| [LiteLLM](https://docs.litellm.ai/) | `litellm/` | Varies | Proxy for 100+ providers |
| [Azure OpenAI](https://portal.azure.com/) | `azure/` | Required | Enterprise Azure deployment |
| [GitHub Copilot](https://github.com/features/copilot) | `github-copilot/` | OAuth | Device code login |
| [Antigravity](https://console.cloud.google.com/) | `antigravity/` | OAuth | Google Cloud AI |
| [AWS Bedrock](https://console.aws.amazon.com/bedrock)* | `bedrock/` | AWS credentials | Claude, Llama, Mistral on AWS |

> \* AWS Bedrock requires build tag: `go build -tags bedrock`. Set `api_base` to a region name (e.g., `us-east-1`) for automatic endpoint resolution across all AWS partitions (aws, aws-cn, aws-us-gov). When using a full endpoint URL instead, you must also configure `AWS_REGION` via environment variable or AWS config/profile.

<details>
<summary><b>Local deployment (Ollama, vLLM, etc.)</b></summary>

**Ollama:**
```json
{
  "model_list": [
    {
      "model_name": "local-llama",
      "model": "ollama/llama3.1:8b",
      "api_base": "http://localhost:11434/v1"
    }
  ]
}
```

**vLLM:**
```json
{
  "model_list": [
    {
      "model_name": "local-vllm",
      "model": "vllm/your-model",
      "api_base": "http://localhost:8000/v1"
    }
  ]
}
```

For full provider configuration details, see [Providers & Models](docs/providers.md).

</details>

## 💬 Channels (Chat Apps)

Talk to your JameClaw through 17+ messaging platforms:

| Channel | Setup | Protocol | Docs |
|---------|-------|----------|------|
| **Telegram** | Easy (bot token) | Long polling | [Guide](docs/channels/telegram/README.md) |
| **Discord** | Easy (bot token + intents) | WebSocket | [Guide](docs/channels/discord/README.md) |
| **WhatsApp** | Easy (QR scan or bridge URL) | Native / Bridge | [Guide](docs/chat-apps.md#whatsapp) |
| **QQ** | Easy (AppID + AppSecret) | WebSocket | [Guide](docs/channels/qq/README.md) |
| **Slack** | Easy (bot + app token) | Socket Mode | [Guide](docs/channels/slack/README.md) |
| **Matrix** | Medium (homeserver + token) | Sync API | [Guide](docs/channels/matrix/README.md) |
| **DingTalk** | Medium (client credentials) | Stream | [Guide](docs/channels/dingtalk/README.md) |
| **Feishu / Lark** | Medium (App ID + Secret) | WebSocket/SDK | [Guide](docs/channels/feishu/README.md) |
| **LINE** | Medium (credentials + webhook) | Webhook | [Guide](docs/channels/line/README.md) |
| **WeCom Bot** | Medium (webhook URL) | Webhook | [Guide](docs/channels/wecom/wecom_bot/README.md) |
| **WeCom App** | Medium (corp credentials) | Webhook | [Guide](docs/channels/wecom/wecom_app/README.md) |
| **WeCom AI Bot** | Medium (token + AES key) | WebSocket / Webhook | [Guide](docs/channels/wecom/wecom_aibot/README.md) |
| **IRC** | Medium (server + nick) | IRC protocol | [Guide](docs/chat-apps.md#irc) |
| **OneBot** | Medium (WebSocket URL) | OneBot v11 | [Guide](docs/channels/onebot/README.md) |
| **MaixCam** | Easy (enable) | TCP socket | [Guide](docs/channels/maixcam/README.md) |
| **Jame** | Easy (enable) | Native protocol | Built-in |
| **Jame Client** | Easy (WebSocket URL) | WebSocket | Built-in |

> All webhook-based channels share a single Gateway HTTP server (`gateway.host`:`gateway.port`, default `127.0.0.1:18790`). Feishu uses WebSocket/SDK mode and does not use the shared HTTP server.

For detailed channel setup instructions, see [Chat Apps Configuration](docs/chat-apps.md).

## 🔧 Tools

### 🔍 Web Search

JameClaw can search the web to provide up-to-date information. Configure in `tools.web`:

| Search Engine | API Key | Free Tier | Link |
|--------------|---------|-----------|------|
| DuckDuckGo | Not needed | Unlimited | Built-in fallback |
| [Baidu Search](https://cloud.baidu.com/doc/qianfan-api/s/Wmbq4z7e5) | Required | 1000 queries/day | AI-powered, China-optimized |
| [Tavily](https://tavily.com) | Required | 1000 queries/month | Optimized for AI Agents |
| [Brave Search](https://brave.com/search/api) | Required | 2000 queries/month | Fast and private |
| [Perplexity](https://www.perplexity.ai) | Required | Paid | AI-powered search |
| [SearXNG](https://github.com/searxng/searxng) | Not needed | Self-hosted | Free metasearch engine |
| [GLM Search](https://open.bigmodel.cn/) | Required | Varies | Zhipu web search |

### ⚙️ Other Tools

JameClaw includes built-in tools for file operations, code execution, scheduling, and more. See [Tools Configuration](docs/tools_configuration.md) for details.

## 🎯 Skills

Skills are modular capabilities that extend your Agent. They are loaded from `SKILL.md` files in your workspace.

**Install skills from ClawHub:**

```bash
jameclaw skills search "web scraping"
jameclaw skills install <skill-name>
```

**Configure ClawHub token** (optional, for higher rate limits):

Add to your `config.json`:
```json
{
  "tools": {
    "skills": {
      "registries": {
        "clawhub": {
          "auth_token": "your-clawhub-token"
        }
      }
    }
  }
}
```

For more details, see [Tools Configuration - Skills](docs/tools_configuration.md#skills-tool).

## 🔗 MCP (Model Context Protocol)

JameClaw natively supports [MCP](https://modelcontextprotocol.io/) — connect any MCP server to extend your Agent's capabilities with external tools and data sources.

```json
{
  "tools": {
    "mcp": {
      "enabled": true,
      "servers": {
        "filesystem": {
          "enabled": true,
          "command": "npx",
          "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
        }
      }
    }
  }
}
```

For full MCP configuration (stdio, SSE, HTTP transports, Tool Discovery), see [Tools Configuration - MCP](docs/tools_configuration.md#mcp-tool).

## <img src="assets/clawdchat-icon.png" width="24" height="24" alt="ClawdChat"> Join the Agent Social Network

Connect JameClaw to the Agent Social Network simply by sending a single message via the CLI or any integrated Chat App.

**Read `https://clawdchat.ai/skill.md` and follow the instructions to join [ClawdChat.ai](https://clawdchat.ai)**

## 🖥️ CLI Reference

| Command                   | Description                      |
| ------------------------- | -------------------------------- |
| `jameclaw onboard`        | Initialize config & workspace    |
| `jameclaw agent -m "..."` | Chat with the agent              |
| `jameclaw agent`          | Interactive chat mode            |
| `jameclaw gateway`        | Start the gateway                |
| `jameclaw status`         | Show status                      |
| `jameclaw version`        | Show version info                |
| `jameclaw model`          | View or switch the default model |
| `jameclaw cron list`      | List all scheduled jobs          |
| `jameclaw cron add ...`   | Add a scheduled job              |
| `jameclaw cron disable`   | Disable a scheduled job          |
| `jameclaw cron remove`    | Remove a scheduled job           |
| `jameclaw skills list`    | List installed skills            |
| `jameclaw skills install` | Install a skill                  |
| `jameclaw migrate`        | Migrate data from older versions |
| `jameclaw auth login`     | Authenticate with providers      |

### ⏰ Scheduled Tasks / Reminders

JameClaw supports scheduled reminders and recurring tasks through the `cron` tool:

* **One-time reminders**: "Remind me in 10 minutes" -> triggers once after 10min
* **Recurring tasks**: "Remind me every 2 hours" -> triggers every 2 hours
* **Cron expressions**: "Remind me at 9am daily" -> uses cron expression

## 📚 Documentation

For detailed guides beyond this README:

| Topic | Description |
|-------|-------------|
| [Docker & Quick Start](docs/docker.md) | Docker Compose setup, Launcher/Agent modes |
| [Chat Apps](docs/chat-apps.md) | All 17+ channel setup guides |
| [Configuration](docs/configuration.md) | Environment variables, workspace layout, security sandbox |
| [Providers & Models](docs/providers.md) | 30+ LLM providers, model routing, model_list configuration |
| [Spawn & Async Tasks](docs/spawn-tasks.md) | Quick tasks, long tasks with spawn, async sub-agent orchestration |
| [Hooks](docs/hooks/README.md) | Event-driven hook system: observers, interceptors, approval hooks |
| [Steering](docs/steering.md) | Inject messages into a running agent loop between tool calls |
| [SubTurn](docs/subturn.md) | Subagent coordination, concurrency control, lifecycle |
| [Troubleshooting](docs/troubleshooting.md) | Common issues and solutions |
| [Tools Configuration](docs/tools_configuration.md) | Per-tool enable/disable, exec policies, MCP, Skills |
| [Hardware Compatibility](docs/hardware-compatibility.md) | Tested boards, minimum requirements |

## 🤝 Contribute & Roadmap

PRs welcome! The codebase is intentionally small and readable.

See our [Community Roadmap](./issues/988) and [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

Developer group building, join after your first merged PR!

Community:

Discord: <https://discord.gg/V4sAZ9XWpN>

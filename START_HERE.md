# Start Here: JameClaw Project Guide

Welcome. This repository contains JameClaw: a local-first AI agent with a Web Console, a terminal/CLI experience, a desktop app, and optional chat-channel connections.

You do **not** need to understand every folder to use the project. Start with the path that matches what you want to do.

## Choose your path

| I want to… | Start here | Then go to |
| --- | --- | --- |
| Use JameClaw on my computer | [README.md](README.md#-install) | Run `jameclaw install`, then open the launcher or use the CLI. |
| Run the browser-based Web Console | [README.md](README.md#-webui-launcher-recommended-for-desktop) | The Web Console runs locally at `http://localhost:18800`. |
| Build JameClaw from source | [README.md](README.md#-build-from-source-for-development) | Use the root `Makefile` commands. |
| Learn what changed recently | [JAMECLAW_UPDATE_HISTORY.md](JAMECLAW_UPDATE_HISTORY.md) | See the current version, milestones, and Git review commands. |
| Configure Telegram, Discord, Slack, or another channel | [docs/channels/](docs/channels/) | Open the folder for your chosen channel. |
| Learn the implementation plan | [docs/implementation/README.md](docs/implementation/README.md) | This is for contributors working through the product. |
| Try a small integration example | [examples/](examples/) | Start with the example README. |
| Contribute code or documentation | [CONTRIBUTING.md](CONTRIBUTING.md) | Check repository conventions before changing shared code. |

## What is where

### Read these first

- **`README.md`** — primary installation, first-run, launcher, CLI, and development guide.
- **`START_HERE.md`** — this orientation page for new visitors.
- **`JAMECLAW_UPDATE_HISTORY.md`** — the maintained record of pushed versions and changes.
- **`ROADMAP.md`** — planned work and product direction.
- **`CONTRIBUTING.md`** — contribution guidance.

### Application source

- **`cmd/`** — command-line entry points for JameClaw and the terminal launcher.
- **`pkg/`** — main Go application code: agent behavior, providers, channels, tools, memory, config, and gateway logic.
- **`web/frontend/`** — React/TanStack source for the browser Web Console.
- **`web/backend/`** — Go server and APIs that host the Web Console and connect it to JameClaw.
- **`macos/`** — native macOS desktop application source.
- **`backend/`** — additional backend-related project code; inspect its own contents before editing.

### Configuration, integrations, and documentation

- **`config/`** — project configuration resources and defaults.
- **`docs/`** — focused documentation, especially channel setup, hooks, implementation notes, and refactor notes.
- **`Chrome-Extension-Upload/`** and **`web/chrome-extension/`** — Chrome extension materials.
- **`docker/`** — Docker Compose setup for running the project in containers.
- **`examples/`** — isolated examples that are safe places to learn from or adapt.
- **`assets/`** — images and visual resources used by documentation and interfaces.

## Folders most users can ignore

These are useful to the build or release process, but are usually not the place to start:

- **`build/`** — compiled binaries and packaged desktop apps. Do not edit these by hand; rebuild them from source.
- **`bin/`** — helper executables/scripts used by the project or packaging process.
- **`scripts/`** — build, release, and maintenance automation.
- **`workspace/`** — local workspace/runtime material. Its contents can be created or changed by JameClaw.
- **`web/frontend/dist/`** and **`web/backend/dist/`** — generated Web Console files. Edit `web/frontend/src/` or backend source instead, then rebuild.
- **`.git/`** and **`.DS_Store`** — Git and macOS metadata; never edit them as product code.

## First 10 minutes for a new user

1. Read the [Install section](README.md#-install) in the README.
2. Run `jameclaw install` after installing the binary or building the project.
3. Start the Web Console with `jameclaw-launcher`, then visit `http://localhost:18800`.
4. Add a credential and choose a model.
5. Use the Chat page first; configure channels only when you are ready to connect them.

## First 10 minutes for a contributor

1. Read [CONTRIBUTING.md](CONTRIBUTING.md) and the [build instructions](README.md#-build-from-source-for-development).
2. For interface work, begin in `web/frontend/src/`; for APIs, begin in `web/backend/`; for core agent behavior, begin in `pkg/`.
3. Run the smallest relevant test or build command from the root `Makefile`.
4. Do not hand-edit generated files in `build/` or `dist/` unless the release workflow explicitly requires committed build outputs.
5. Record user-visible changes in [JAMECLAW_UPDATE_HISTORY.md](JAMECLAW_UPDATE_HISTORY.md) when preparing a release or a push.

## Quick commands

Run these commands from the repository root:

```bash
# Show available project commands
make help

# Build the core JameClaw binary
make build

# Build the Web Console launcher
make build-launcher

# Run the test suite
make test
```

If you are unsure where a feature belongs, search the repository before creating a new folder:

```bash
rg "feature name" pkg web cmd docs
```

That keeps the project understandable for the next person who opens it.

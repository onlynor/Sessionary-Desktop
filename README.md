# Sessionary Desktop

[![Release](https://img.shields.io/github/v/release/onlynor/Sessionary-Desktop)](https://github.com/onlynor/Sessionary-Desktop/releases)
[![CI](https://github.com/onlynor/Sessionary-Desktop/actions/workflows/ci.yml/badge.svg)](https://github.com/onlynor/Sessionary-Desktop/actions/workflows/ci.yml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**English** | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md)

**Sessionary Desktop** is an open-source desktop control center for AI coding agents — Claude Code, Codex, OpenCode,
Pi, DeepSeek Harness, Hermes, WorkBuddy and WorkBuddy AI. It packages [Sessionary](https://github.com/onlynor/Sessionary)
as a native application for Linux, macOS and Windows, so you manage every agent's projects, sessions, model routing
and token usage in one window instead of a browser tab.

Unlike a conventional AI chat application, Sessionary Desktop doesn't run its own chat engine or store its own
conversations: it reads the real session history and files that each agent CLI already keeps on disk, on this
computer or on a remote machine over SSH, and gives you one consistent interface — agent cards, chat, a terminal,
model routing, usage — on top of them.

## Features

- **AI coding agent management** — detects installed agent CLIs (Claude Code, Codex, OpenCode, Pi, Hermes,
  WorkBuddy, WorkBuddy AI, and [DeepSeek Harness](docs/ARCHITECTURE.md#deepseek-harness-integration) if present) and
  shows each one's sessions, so you don't juggle separate terminals and config files per agent.
- **Projects and sessions** — browse an agent's past sessions by project directory, resume one, or start a new
  session in a folder you pick.
- **Local and remote machines** — work with agents on this computer, or add another machine over **SSH** (key-based,
  no password or token stored) to read its agent history; nothing is installed on the remote machine.
- **Model providers and routing** — configure model providers and API keys once, then route which model each agent
  uses from the Routing page, through Sessionary's own gateway.
- **Token usage and analytics** — see usage and token consumption per agent and machine on the Usage page.
- **Terminal and monitor** — open a real terminal in a project, and watch agent activity on a machine from one tab.

Only what's implemented today is listed above. See [Not yet done](docs/ARCHITECTURE.md#not-yet-done) for known gaps.

## Installation and getting started

Download the latest build from **[GitHub Releases](https://github.com/onlynor/Sessionary-Desktop/releases)**. Each
release publishes:

| Platform | Download | Notes |
|---|---|---|
| Linux (amd64, arm64) | `.deb`, or a `.tar.gz` with `install.sh` | |
| macOS (Apple silicon) | `.dmg` | Intel Macs are not built yet |
| Windows (amd64) | `Sessionary-Setup-<version>-amd64.exe` | arm64 is not built yet |

Packages are **not code-signed or notarized yet**: macOS Gatekeeper and Windows SmartScreen will warn on first run
(see [Troubleshooting](#troubleshooting)). Verify a download against the release's `SHA256SUMS` with
`sha256sum -c SHA256SUMS`.

**Install:**

```sh
# Linux, .deb
sudo apt install ./sessionary_<version>_amd64.deb   # or _arm64.deb

# Linux, tarball
tar xf sessionary-<version>-linux-amd64.tar.gz && ./install.sh

# macOS
# open the .dmg, drag Sessionary to Applications

# Windows
# run Sessionary-Setup-<version>-amd64.exe
```

**Prerequisites:** Sessionary Desktop doesn't bundle any agent CLI. Install at least one agent it supports (for
example [Claude Code](https://claude.com/claude-code), [Codex](https://github.com/openai/codex), OpenCode, or Pi) and
make sure it's reachable from your normal login shell. [DeepSeek Harness](docs/ARCHITECTURE.md#deepseek-harness-integration)
needs nothing configured — Sessionary finds a supported installation on its own if you have one.

**First session:** launch the app → it detects agents already installed on your machine → pick **Agents** in the
sidebar, choose a detected agent → pick a project folder → start (or resume) a session. No account or sign-in step.

## How to use

The sidebar has three groups:

- **Home** and **Machines** — an overview, and the list of machines (this computer, plus any you've added over SSH).
- **Workspace** (scoped to the selected machine) — **Agents** (per-agent cards and sessions), **Projects**, **Sessions**,
  **Terminal**, and **Monitor** (live agent activity on that machine).
- **Model Control** — **Models** and **Gateway** (configure providers and API keys), **Routing** (choose which model
  each agent uses), and **Usage** (token consumption over time).

**Normal workflow:** pick a machine in the sidebar (defaults to this computer) → open **Agents** and pick one that's
detected → pick a project directory → start a new session or resume an existing one from **Sessions**. Conversations
happen in Sessionary's own chat view — streaming replies, tool calls, diffs, and the model picker — not in a plain
terminal, though opening a real terminal for the same project is always one click away.

**Agent detection and PATH.** An agent is "detected" when its CLI is found on `PATH`. If you installed an agent after
launching the app, or into `~/.local/bin`, an npm/pnpm global prefix, or a version manager (nvm, volta, fnm, mise,
asdf), restart the app so it re-reads your login shell's `PATH`. See [Troubleshooting](#troubleshooting) if an agent
you know is installed still doesn't show up.

**Model providers and routing.** Add providers and API keys once on the **Gateway**/**Models** pages, then use
**Routing** to pick which model each agent talks to; agents reach your configured providers through Sessionary's own
gateway, so keys live in one place. **Usage** breaks down token consumption by agent and machine.

**Remote machines over SSH.** On **Machines**, add a node by host (or an alias from your `~/.ssh/config`) and,
optionally, port and identity file. Sessionary connects with your own SSH keys — no password or token is stored —
and only reads that machine's existing agent session history (Claude Code, OpenCode, Pi, Hermes); nothing is
installed on it. DeepSeek Harness is local-only and isn't available on remote machines. Make sure plain
`ssh user@host true` already works (host key trusted, no password prompt) before adding a node.

For implementation detail beyond this day-to-day usage, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) and
[docs/DESKTOP_RUNTIME.md](docs/DESKTOP_RUNTIME.md).

## Troubleshooting

- **An installed agent doesn't show up.** The app reads `PATH` from your login shell, not from however it was
  launched. Confirm the agent runs from a fresh terminal (`which <agent>`), then fully quit and relaunch Sessionary
  Desktop so it re-reads the environment.
- **macOS says the app is damaged or from an unidentified developer.** Builds aren't notarized yet. Right-click the
  app → **Open**, then confirm in the dialog that appears.
- **Windows SmartScreen blocks the installer.** Click **More info** → **Run anyway**. This is expected until builds
  are signed.
- **Adding a remote machine fails or hangs.** Test the connection by hand first: `ssh user@host true` (using the
  same host/user/port/identity you're entering) must succeed without a password prompt. Fix SSH access, then add the
  node again.
- **The app window disappears after closing it (Linux).** On Linux, closing hides the window to the tray only when a
  StatusNotifierItem host is running (most status bars with an AppIndicator extension/module). Without one, closing
  quits the app instead — this is expected, not a bug. Reopen it from your application launcher.
- **The app won't start / shows "Backend unavailable."** Use **Show Details** in that screen to see the bundled
  server's output, or check `~/.config/Sessionary/logs/backend.log` (Linux/macOS path; Windows uses the equivalent
  per-user config directory). **Try Again** restarts it; if it keeps failing, please open an issue with that log.

## Project structure and technical overview

Sessionary Desktop is a single Go binary, built with [MyGo](https://github.com/egoist/mygo), that embeds Sessionary's
own web client (built from a sibling [`Sessionary`](https://github.com/onlynor/Sessionary) checkout) and serves it
from a loopback port inside a native WebKit/WebView2 window. A supervised background process runs the bundled
Sessionary server (with a bundled Node.js, so nothing needs installing on the machine); the window talks to it the
same way a browser would talk to a self-hosted Sessionary.

```
cmd/sessionary-desktop/   the application: lifecycle, window, tray, menus
internal/{backend,uiserver,assets}/
frontend/                 Sessionary's web client, built in place, plus a desktop UI layer
scripts/, .github/        build/release scripts and CI
```

Full detail: [Architecture](docs/ARCHITECTURE.md) · [Desktop runtime (backend lifecycle, window, tray)](docs/DESKTOP_RUNTIME.md) · [Releases and versioning](docs/RELEASES.md).

## Development and contributing

Requirements: Go 1.27+, [Bun](https://bun.sh), Node.js, WebKitGTK 4.1 (Linux), and a sibling checkout of
[Sessionary](https://github.com/onlynor/Sessionary) with `pnpm install` run in it.

```sh
bun install --cwd frontend
bun run dev         # hot-reloading dev build
bun run build       # production build → build/<os>-<arch>/
bun run typecheck
go test ./... && bun run --cwd frontend test
```

Issues and pull requests are welcome. See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for the full build, test and CI
setup, and [docs/RELEASES.md](docs/RELEASES.md) for how versions are cut and published.

## License and links

Sessionary Desktop is licensed under the [Apache License 2.0](LICENSE). This binary also bundles a Node.js runtime
and statically links a few Go libraries under their own licenses; see
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) (shipped next to the executable in every package).

- [GitHub Releases](https://github.com/onlynor/Sessionary-Desktop/releases)
- [Sessionary](https://github.com/onlynor/Sessionary) — the web client and server this app packages
- [Issues](https://github.com/onlynor/Sessionary-Desktop/issues)

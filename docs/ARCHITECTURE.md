# Architecture

How Sessionary Desktop is put together, for anyone changing the code. For installing and using the app, see the
[README](../README.md).

## Overview

```
┌──────────────────────── Sessionary Desktop (one Go binary, MyGo) ────────────────────────┐
│ cmd/sessionary-desktop   the app: lifecycle, window, tray, menus, the services the page    │
│                          calls (typed, generated into frontend/src/mygo.ts), prefs        │
│ internal/backend         supervise the server: start, health, restart, give up, stop      │
│ internal/uiserver        loopback origin: the embedded page + reverse proxy of /api       │
│ internal/assets          the icons, embedded                                              │
│ frontend/                the page: Sessionary's web client + the desktop layer, embedded  │
└───────────────┬───────────────────────────────────────────────┬──────────────────────────┘
                │ http://127.0.0.1:<ui port>                    │ http://127.0.0.1:<app's own port>
        ┌───────▼────────┐                             ┌────────▼─────────┐
        │ WebKit window  │  /api, SSE, token, gateway  │ Sessionary server│  ../Sessionary, bundled
        │ the page       │ ─────────────────────────▶  │ bundled Node.js  │  as server/cli.mjs
        └────────────────┘                             └──────────────────┘
```

Repository layout:

```
cmd/sessionary-desktop/  the application (package main: MyGo binds the services of the main package)
internal/{backend,uiserver,assets}/
frontend/                Vite build of ../Sessionary/web/src + src/desktop (layer) + src/overrides + test/
scripts/build/           build-backend.mjs (the server, one file), fetch-node.mjs (the pinned runtime)
scripts/dev/             run.sh, drive.py, view.py — scripted development runs and screenshots
scripts/test/            smoke.sh (Linux, virtual display), smoke-deb.sh (clean Ubuntu), smoke-native.sh (macOS, Windows)
.github/workflows/       ci.yml, release.yml
mygo.json  package.json  go.mod  README.md
```

**Backend.** A production build owns its server: the bundled Node.js runs the bundled server on a loopback port of
the app's own, remembered between launches (`backend.json` in the user data dir) so Model Control's gateway keeps its
address, and given up only when it is taken. It never attaches to a server it did not start. A development build may
use the server a developer runs on `:4777` (`pnpm dev:server`); see [Backend lifecycle and window behavior](DESKTOP_RUNTIME.md).
`../Sessionary` is changed only where a new agent has to be registered — its own adapter that reads its sessions, its
chat/agent registry, and the names and mark the shared Agent UI uses; none of its session, model or navigation
abstractions is rewritten.

**Page origin.** The page is served from a loopback port by the app itself, not from the server and not from a custom
scheme, so it talks to the server exactly as the web client does: relative `/api/…` URLs, server-sent events,
terminals, the per-launch token, the Origin/Host guards. The UI server refuses non-loopback Host names (DNS
rebinding), like the server does. Because that origin changes between launches, `localStorage` is mirrored to
`prefs.json` in the user data directory (`frontend/src/desktop/runtime.ts`, `cmd/sessionary-desktop/prefs.go`).

**Frontend.** `frontend/` builds Sessionary's web client **in place** from `../Sessionary/web/src` (never copied), with a
desktop layer on top:

| | |
|---|---|
| `src/main.tsx` | restores settings, marks the document, shows the launch screen until the server answers, mounts the client |
| `src/desktop/desktop.css` | the desktop visual layer, scoped to `html.desktop` (inert in the web build) |
| `src/desktop/behaviors.ts` | no browser context menu / link dragging / reload keys; back & forward (keys, mouse buttons); scroll state for toolbar materials and overlay scrollbars; menu-bar shortcuts |
| `src/desktop/NavButtons.tsx` | toolbar back/forward with real enabled state (numbered history entries) |
| `src/desktop/Launch.tsx` | launch screen; failure view with Retry and the server's output |
| `src/desktop/runtime.ts` | MyGo runtime glue: prefs bridge, window background follows the theme, menu commands, system notifications |
| `src/desktop/i18n.ts` | the layer's strings, merged into Sessionary's dictionary (en, 简体, 繁體, 日本語) |
| `src/overrides/<path>` | replaces `web/src/<path>` wherever the client imports it: `Nav.tsx` (the web client's sidebar, with this platform's search shortcut and a marker class the layer finds it by) and `MachinePage.tsx` (the agents' display order). Keep overrides rare; `test/overrides.test.ts` undoes each override's documented changes and compares it with the web client's file |

React and every package resolve from Sessionary's own install, so there is one copy of each.

## Decisions worth knowing

- **Web UI reused, desktop frame replaced.** The client's pages are mature; what made it look like a web app was
  the frame: a floating card sidebar, an ambient gradient canvas, page headers inside the reading column. The desktop
  replaces the frame (flush source list, 52px title bar on both sides of the window, toolbar that takes a material
  only when content scrolls under it, large titles on the reading column) and quiets surfaces (figures as one strip,
  no cards within cards, sentence-case headings, row actions on hover). Native UI (MyGo's `ui` package) is kept for
  places with a real advantage: the startup window (see
  [Memory: native startup, idle teardown](DESKTOP_RUNTIME.md#memory-native-startup-idle-teardown)) is the one case
  found so far where loading the WebView itself is the thing worth avoiding; nothing else has justified it over the
  existing web surfaces.
- **Fonts on Linux.** WebKitGTK takes fontconfig's substitute for unknown families, so `-apple-system` silently became
  Noto Sans CJK. On Linux the stack starts with the bundled Inter, then the CJK face for the UI language.
- **Blur.** MyGo disables WebKitGTK's DMA-BUF renderer for safety, which also disables `backdrop-filter`. Rather than
  probe GPUs, drivers and VMs, Linux always draws materials opaque (`data-no-blur`); macOS and Windows blur.
- **Window controls.** See [Window, tray and quitting](DESKTOP_RUNTIME.md#window-controls).
- **Menus.** macOS gets a menu bar; Linux and Windows keep the window's full height and answer the same shortcuts
  in the page (`Ctrl K` search, `Ctrl N` new session, `Ctrl ,` settings, `Ctrl 1–5` sections, `Alt ←/→` history).

## DeepSeek Harness integration

[DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness) (`dsh`) is a first-class Sessionary agent, like
Claude Code, Codex, OpenCode and Pi: Sessionary's adapter reads its own session files, its Agent card and AgentPage
are the shared ones, and its conversations run on the harness's automation-only ACP server (`dsh --profile acp`)
through the ACP driver Sessionary already uses. Nothing is embedded, there is no second application shell, and there
is no second place to configure providers.

| | |
|---|---|
| Install | **not bundled**, and **nothing to configure**: one resolver finds a supported installation the machine already has — a `dsh` on `PATH`, an npm/pnpm/bun global installation, or a cached `npx` installation. The resolver needs no source checkout, no environment variable and no Settings entry, and it is the only thing that decides which installation is used, so the probe and the launcher cannot disagree. It validates more than a version string: the package must be `@deepseek-ai/dsh`, ship its CLI entry point, and boot its `acp` profile. The newest supported installation wins (semantic, prerelease-aware), a stale cached one can never hide a valid one, and nothing is downloaded or installed at runtime. `SESSIONARY_DSH` remains as an internal development override |
| Run | the Sessionary server starts it like any agent, through a launcher it writes beside its own data (the resolved entry point, with the very Node.js the server itself runs on — the app's bundled runtime in a packaged build): `--patch <sessionary>/launch/dsh-sessionary.yml --profile acp`. Its ACP sessions are held open by Sessionary's chat layer, with the restart recovery that already covers the server |
| Card & pages | exactly the shared Agent card and AgentPage. Open browses its real sessions, projects and search hits; **New session** runs the ordinary workspace flow. A machine with no supported installation shows the ordinary missing state — never an "available" agent that cannot start |
| Model Control | the route for `deepseek-harness` (Routing page) reaches it through Sessionary's gateway: the per-launch profile patch mounts `@deepseek-ai/dsh-llm-pi-ai` as Sessionary's own provider (`baseURL = <gateway>/v1`, the gateway key in the process environment). Providers and keys are configured once, in Sessionary. A resumed session whose own recorded model is no longer the route's carries that model only so the harness can restore its log, and is then switched to the route now selected |
| Isolation | `DSH_HOME` = `<Sessionary data>/deepseek-harness`; your own `~/.dsh` is never read or written, so the two never contend for a session lock. Telemetry off (`DSH_TELEMETRY_DISABLED=1`) unless your environment sets it |
| Icon | its published mark, bundled with the web client |

Not supported: dsh on SSH nodes (it runs on this computer), and the surfaces the automation-only ACP contract
intentionally omits (agent presets, the plugin manager, plans, terminals, elicitation). Harness's own settings own
anything the gateway cannot express.

## Agents tab order

Explicit, in `frontend/src/desktop/agentOrder.ts` (tested): **Claude Code, Codex, DeepSeek Harness, Pi / OpenCode,
Hermes, WorkBuddy, WorkBuddy AI**; agents it does not name follow in the server's order. Four cards a row where they
fit at the existing typography (the reading column ≥ 1040 px), else two — never three, which would break the rows —
else one. The order lives in `overrides/MachinePage.tsx`, the web client's file with only that change;
`frontend/test/overrides.test.ts` rebuilds the web client's file from it and fails when upstream changes.

## Not yet done

- macOS and Windows smoke runs are wired in CI (`scripts/test/smoke-native.sh`, best effort) but have not run yet; only Linux's is proven.
- macOS: vibrancy sidebar, signing and notarization. A preferences window, auto-update, a tray badge for running agents.

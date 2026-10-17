# Sessionary Desktop

Sessionary — the control center for your AI coding agents — as a desktop application, built with
[MyGo](https://github.com/egoist/mygo). Built and packaged for Linux, macOS and Windows from one codebase.

It is the whole Sessionary product (Projects, Sessions, live Chat, Model Control, Usage, Machines, Terminal,
Monitor) in a real application window: a flush source-list sidebar with the web client's own navigation, a
title-bar toolbar with back/forward, system appearance, system notifications, a system tray, its own server
lifecycle, and none of a browser's chrome or behaviour.

## Architecture

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
.github/workflows/       ci.yml
mygo.json  package.json  go.mod  README.md
```

**Backend.** A production build owns its server: the bundled Node.js runs the bundled server on a loopback port of
the app's own, remembered between launches (`backend.json` in the user data dir) so Model Control's gateway keeps its
address, and given up only when it is taken. It never attaches to a server it did not start. A development build may
use the server a developer runs on `:4777` (`pnpm dev:server`); see [Backend lifecycle](#backend-lifecycle).
`../Sessionary` is changed only where a new agent has to be registered — its own adapter that reads DeepSeek
Harness's sessions, its chat/agent registry, and the names and mark the shared Agent UI uses; none of its session,
model or navigation abstractions is rewritten.

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

## Develop

Requirements to build: Go 1.27+, Bun, Node.js (for the build tools), WebKitGTK 4.1 (Linux), and `pnpm install` done
in `../Sessionary` (or set `SESSIONARY_DIR`). To run a built app: nothing beyond GTK 3 and WebKitGTK on Linux.

```sh
bun install --cwd frontend
bun run dev            # mygo dev: Vite with hot reload through the app's own origin, Go rebuilds on change
bun run build          # frontend + server + pinned Node.js runtime + mygo build → build/<os>-<arch>/ (.deb, tarball, …)
bun run typecheck      # go vet (all platforms) + tsc over the desktop layer and the client it pulls in
```

Scripted runs, for screenshots and checks (development builds only):

```sh
scripts/dev/run.sh                   # build, start, wait until the server answers
scripts/dev/drive.py go '#/models'       # navigate · eval · theme · size · shot · tray · status
scripts/dev/drive.py tour .shots/x ember # every main surface, one PNG each
```

Environment: `SESSIONARY_DESKTOP_ATTACH=0` always starts an own server · `SESSIONARY_SERVER` a server script ·
`SESSIONARY_NODE` a Node.js · `SESSIONARY_HOME` Sessionary's data dir (handy for a clean first run).

```sh
go test ./...                                    # lifecycle (fake servers act out crashes, hangs, …), proxy, layout, dsh
bun run --cwd frontend test                      # agents' order; overrides in step with the web client
scripts/test/smoke.sh build/linux-amd64/sessionary    # the built app, as a first-time user
scripts/test/smoke-deb.sh build/linux-amd64/*.deb     # the .deb on a clean Ubuntu in Docker
```

## Decisions worth knowing

- **Web UI reused, desktop frame replaced.** The client's pages are mature; what made it look like a web app was
  the frame: a floating card sidebar, an ambient gradient canvas, page headers inside the reading column. The desktop
  replaces the frame (flush source list, 52px title bar on both sides of the window, toolbar that takes a material
  only when content scrolls under it, large titles on the reading column) and quiets surfaces (figures as one strip,
  no cards within cards, sentence-case headings, row actions on hover). Native UI (MyGo's `ui` package) is kept for
  places with a real advantage — none yet justified it over the existing web surfaces.
- **Fonts on Linux.** WebKitGTK takes fontconfig's substitute for unknown families, so `-apple-system` silently became
  Noto Sans CJK. On Linux the stack starts with the bundled Inter, then the CJK face for the UI language.
- **Blur.** MyGo disables WebKitGTK's DMA-BUF renderer for safety, which also disables `backdrop-filter`. Rather than
  probe GPUs, drivers and VMs, Linux always draws materials opaque (`data-no-blur`); macOS and Windows blur.
- **Window controls.** See [Window controls](#window-controls).
- **Menus.** macOS gets a menu bar; Linux and Windows keep the window's full height and answer the same shortcuts
  in the page (`Ctrl K` search, `Ctrl N` new session, `Ctrl ,` settings, `Ctrl 1–5` sections, `Alt ←/→` history).

## Backend lifecycle

One goroutine supervises the server (`internal/backend`); the page only shows the status it is sent.

```text
launch → start bundled Node + server → wait until it answers (≤ 2 min; it indexes first) → page connects
       → health check every 2 s ─ 2 misses: "reconnecting" ─ 5 misses: kill the group, restart
       → server exits: restart with backoff (0.25 s … 8 s); a server that answered and died is restarted for as
         long as it takes; starts that never answer give up after 5 → "unavailable" (Try Again / Show Details / Quit)
quit   → SIGTERM to the server's process group (it stops the agents it started), SIGKILL after 5 s
crash  → the kernel ends the server (Pdeathsig, from a dedicated never-exiting thread)
```

| State (page) | Phase | What the window shows |
|---|---|---|
| Starting | `starting` | launch screen |
| Connecting | `connecting` | launch screen, "Reading your agents' sessions…" |
| Ready | `ready` | the client; a new server (`generation`) remounts it on the same page |
| Reconnecting | `reconnecting` | panel over the inert client |
| Restarting backend | `restarting` | panel, with the attempt |
| Backend unavailable | `failed` | panel with Try Again, Show Details (the server's output, log path), Quit |

The window stays interactive throughout: title buttons, moving and resizing, and the panel's actions. Panels appear
only after a moment (no flash on a hiccup). The server's output goes to `~/.config/Sessionary/logs/backend.log`.
`/api` and `/gateway` are proxied to whichever server is current; while there is none they answer 503.

**Environment.** An app started from a launcher or desktop entry gets the session's `PATH`, not the terminal's, so
agents installed in `~/.local/bin`, npm/pnpm globals or a version manager would be "not found on PATH". The server
is therefore started with the user's login shell's `PATH` (`shell -ilc`, 8 s limit, markers around the answer, merged
with the current one; any failure keeps the current `PATH`) — `internal/shellenv`. Detection, re-detection and
starting an agent all read the server's own environment, so they agree.

**Development:** with a server already on `:4777`, a dev build uses it and, if it goes away, waits for it ("Waiting
for the development server on port 4777…") rather than starting another one beside it.
`SESSIONARY_DESKTOP_ATTACH=0` always starts the app's own.

## Bundled runtime

Production builds carry their own Node.js, so nothing needs to be installed on the machine:

| | |
|---|---|
| Version | Node.js **24.21.0** (LTS), pinned with each platform's SHA-256 in `scripts/build/fetch-node.mjs` |
| Fetched | at **build** time (`node scripts/build/fetch-node.mjs`, part of `bun run build`), verified, cached in `~/.cache/sessionary-desktop/node`; never at first launch |
| Staged | `.stage/runtime/{bin/node,LICENSE}` (stripped on Linux: 108 MB), `.stage/server/cli.mjs` (the server, one self-contained file) |
| Installed | next to the executable: `/opt/sessionary/{sessionary,runtime/bin/node,server/cli.mjs}` from the `.deb` (with `/usr/bin/sessionary-desktop`), the same three in the tarball |
| Found | `runtimeCandidates()` / `serverScripts()` in `cmd/sessionary-desktop/supervise.go`: `SESSIONARY_NODE` / `SESSIONARY_SERVER`, then the resources dir, then the executable's dir (symlinks resolved) |
| Development | the same `.stage/` once built; failing that, a dev build may use any Node.js ≥ 22.13 it finds (PATH, nvm, volta, fnm, mise, asdf). A production build never uses a system Node.js (`SESSIONARY_SYSTEM_NODE=1` to allow it) |

Package sizes (linux/amd64): `.deb` 46 MB, tarball 46 MB (the runtime is 108 MB of the 121 MB installed).

## Window controls

One strategy per OS, chosen by `GOOS` and nothing else — no code looks at X11 versus Wayland, the compositor or the
desktop environment (the approach of [Magpie](https://github.com/yetone/magpie), which keeps one GTK mechanism and
one CI matrix instead of a table of desktops):

| Platform | Title bar | Buttons |
|---|---|---|
| macOS | hidden, inset | the system's traffic lights over the sidebar |
| Windows | hidden | Windows 11's caption buttons (with snap layouts), drawn by the system |
| Linux | frameless (MyGo announces client-side decorations, so no compositor adds a title bar) | drawn by the page, on the right: maximize and close, plus minimize when the app has a tray (see below) |

The title bar's empty space moves the window, double-click maximizes, F11 toggles full screen. Linux compositors differ in whether
they minimize at all, so **Minimize hides the window in the tray** — the way back every compositor honours (tray menu,
a second launch, a notification) — and with no tray the button is not drawn rather than doing nothing.

## Window, tray and quitting

The window's close button **hides the window**; the app keeps running — and keeps its server, so Model Control's
gateway keeps answering — in the menu bar (macOS) or the notification area (Linux, Windows). The tray's menu brings
it back, puts it away again, or quits for good; the macOS app menu, `Ctrl Q` and the launch screen's Quit do the
same. The rule is one line: **with a tray that something displays, closing only hides and quitting always closes; without
one, closing quits as it always did**, so a window can never be left invisible with no way back. A real quit sets a flag before the
window is closed, so the close handler lets it through, the tray is destroyed, and the server's process group is
stopped. Activating the app (the Dock, the taskbar, a second launch) restores the **same** window — there is never a
second one.

The icon is Sessionary's own mark: the coloured logo on Linux and Windows, and a black template on macOS (tinted for
light and dark menu bars). It is handed to MyGo's native tray (`mygo.NewTray`), so it is a real StatusNotifierItem /
NSStatusItem, not an element drawn in the page. Platform notes:

- **Linux** shows a tray only with `libayatana-appindicator3` (or `libappindicator3`) and a StatusNotifierItem host
  (a bar such as noctalia, waybar or the GNOME AppIndicator extension). Without the library `NewTray` fails and the
  app runs as before, without a tray (and closing quits). Whether a host exists is asked of the StatusNotifier watcher on the session
  bus (every 2 s, protocol only): with none, closing quits and Minimize is not drawn; if the host vanishes while the
  window is hidden the window is shown again, and when it returns the tray is used again. The menu works through the host's dbusmenu; a
  click on the icon itself is not delivered (AppIndicator has no click events).
- **macOS** keeps the Dock icon as well, so both the Dock and the menu bar restore the window; the app is not an
  accessory (no `SetActivationPolicy`).
- **Windows** uses the notification area (left- and right-clicks both open the menu). The server runs in a job
  object (`KILL_ON_JOB_CLOSE`), so it never outlives the app, even if the app crashes; a server that cannot be put in
  one is not started. Its agents and terminals end with it through Node.js's own job; terminals and editors
  Sessionary launched detached for you keep running, as on Unix. The tests run in CI's Windows job.

`scripts/dev/drive.py tray` reports whether the tray is installed, `window hide|show|quit` exercises the lifecycle, and
`go test ./...` covers the icon, the menu and the close-versus-quit rule (`cmd/sessionary-desktop/tray_test.go`).

## Continuous integration

`.github/workflows/ci.yml`, after Magpie's: on every push and pull request,
- **go** (Linux, macOS, Windows): gofmt, vet for all three platforms, tests (the lifecycle suite drives real process
  groups on Unix), race detector on Linux;
- **package** (linux/amd64, linux/arm64, darwin/arm64, windows/amd64): checks out Sessionary beside the app
  (`SESSIONARY_TOKEN` secret if private; `sessionary_ref` input), generates the bindings, type-checks the frontend, runs its tests, `go tool mygo build`;
  on macOS and Windows, `scripts/test/smoke-native.sh` runs the packaged executable once;
  on Linux, `scripts/test/smoke.sh` runs the binary under Xvfb and `scripts/test/smoke-deb.sh` installs the `.deb` on a clean
  Ubuntu without Node.js and runs the same smoke test. Packages and smoke reports are uploaded.

The smoke test (`SESSIONARY_DESKTOP_SMOKE=report.json`, any build) checks: the bundled server answers, the page
loads and reaches it through the app's origin, title buttons are drawn, a killed server is replaced and the page
recovers; then `smoke.sh` checks that quitting left no process of the installation running.

## Releases and versions

**Version.** [Semantic Versioning](https://semver.org), written once: `"version"` in `mygo.json` (no `v`), which
`mygo build` puts into the packages and the app (About, the smoke report). The git tag is `v` + that. While the major
is 0, anything may change between minors; PATCH is fixes, MINOR is features or a new bundled Sessionary / Node.js, and
1.0.0 is the promise that upgrades will not break settings. This version is the *app's*, not Sessionary's: the server
it bundles has its own, and each release names the Sessionary commit it was built from.

**Publishing** (`.github/workflows/release.yml`; a tag does everything, as a draft):

```sh
# 1. a pull request "release: v0.1.0" sets "version" in mygo.json; merge it
git switch main && git pull
git tag v0.1.0 && git push origin v0.1.0
# 2. Actions → Release: the CI jobs run on the tagged commit with the version enforced, then the draft appears
# 3. Releases → the draft: check the notes and the smoke summaries, then Publish
```

The workflow refuses a tag that is not `v` + `mygo.json`'s version or is not on `main`, runs the same build, tests and
smoke tests as CI (each packaged app must report that version), and drafts the release with the `.deb` and tarball
(+ `install.sh`) for Linux amd64/arm64, the `.dmg` for macOS (Apple silicon), the installer for Windows (amd64) and
`SHA256SUMS`. A suffix (`v0.2.0-rc.1`) makes a pre-release. Packages are not signed or notarized yet, and there is no
auto-update (mygo's updater needs a signing key). The Sessionary build can be pinned by setting the repository
variable `SESSIONARY_RELEASE_REF` to a tag or commit there (default `main`).

## DeepSeek Harness

[DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness) (`dsh`) is a **first-class Sessionary agent**,
like Claude Code, Codex, OpenCode and Pi: Sessionary's adapter reads its own session files, its Agent card and
AgentPage are the shared ones, and its conversations run on the harness's automation-only ACP server
(`dsh --profile acp`) through the ACP driver Sessionary already uses. Nothing is embedded, there is no second
application shell, and there is no second place to configure providers.

| | |
|---|---|
| Install | **not bundled**, and **nothing to configure**: one resolver finds a supported installation the machine already has — a `dsh` on `PATH`, an npm/pnpm/bun global installation, or a cached `npx` installation. The resolver needs no source checkout, no environment variable and no Settings entry, and it is the only thing that decides which installation is used, so the probe and the launcher cannot disagree. It validates more than a version string: the package must be `@deepseek-ai/dsh`, ship its CLI entry point, and boot its `acp` profile. The newest supported installation wins (semantic, prerelease-aware), a stale cached one can never hide a valid one, and nothing is downloaded or installed at runtime. `SESSIONARY_DSH` remains as an internal development override |
| Run | the Sessionary server starts it like any agent, through a launcher it writes beside its own data (the resolved entry point, with the very Node.js the server itself runs on — the app's bundled runtime in a packaged build): `--patch <sessionary>/launch/dsh-sessionary.yml --profile acp`. Its ACP sessions are held open by Sessionary's chat layer, with the restart recovery that already covers the server |
| Card & pages | exactly the shared Agent card and AgentPage. Open browses its real sessions, projects and search hits; **New session** runs the ordinary workspace flow. A machine with no supported installation shows the ordinary missing state — never an "available" agent that cannot start |
| Conversation | Sessionary's own chat pages: streaming replies, thinking, tool calls and diffs, the model picker, usage, approval prompts, interruption — the same interface as every other agent |
| Model Control | the route for `deepseek-harness` (Routing page) reaches it through Sessionary's gateway: the per-launch profile patch mounts `@deepseek-ai/dsh-llm-pi-ai` as Sessionary's own provider (`baseURL = <gateway>/v1`, the gateway key in the process environment). Providers and keys are configured once, in Sessionary. A resumed session whose own recorded model is no longer the route's carries that model only so the harness can restore its log, and is then switched to the route now selected |
| Isolation | `DSH_HOME` = `<Sessionary data>/deepseek-harness`; your own `~/.dsh` is never read or written, so the two never contend for a session lock. Telemetry off (`DSH_TELEMETRY_DISABLED=1`) unless your environment sets it |
| Icon | its published mark, bundled with the web client |

Not supported: dsh on SSH nodes (it runs on this computer), and the surfaces the automation-only ACP contract
intentionally omits (agent presets, the plugin manager, plans, terminals, elicitation). Harness's own settings
own anything the gateway cannot express.

## Agents tab order

Explicit, in `frontend/src/desktop/agentOrder.ts` (tested): **Claude Code, Codex, DeepSeek Harness, Pi / OpenCode,
Hermes, WorkBuddy, WorkBuddy AI**; agents it does not name follow in the server's order. Four cards a row where they
fit at the existing typography (the reading column ≥ 1040 px), else two — never three, which would break the rows —
else one. The order lives in `overrides/MachinePage.tsx`, the web client's file with only that change;
`frontend/test/overrides.test.ts` rebuilds the web client's file from it and fails when upstream changes.

## Not yet done

- macOS and Windows smoke runs are wired in CI (`scripts/test/smoke-native.sh`, best effort) but have not run yet; only Linux's is proven.
- macOS: vibrancy sidebar, signing and notarization. A preferences window, auto-update, a tray badge for running agents.

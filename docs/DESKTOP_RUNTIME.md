# Desktop runtime: backend lifecycle, window, tray

How the desktop shell manages the bundled server and the native window across platforms. For the product features
this enables, see the [README](../README.md).

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

| State | Phase | What shows, and where |
|---|---|---|
| Starting | `starting` | native wait window (`cmd/sessionary-desktop/launch.go`), no WebView yet |
| Connecting | `connecting` | the same native window, "Reading your agents' sessions…" |
| Ready | `ready` | the real window opens (or, after the first time, a new server's `generation` just remounts the already-open page) |
| Reconnecting | `reconnecting` | once the real window exists: a panel over the inert client, in the page (`BackendState`, `Launch.tsx`) |
| Restarting backend | `restarting` | the panel, with the attempt — or, if the real window is not open yet (first launch, or reopening after an idle teardown), the native window again |
| Backend unavailable | `failed` | the panel (or the native window) with Try Again, Show Details (the server's output, log path), Quit |

The window stays interactive throughout: title buttons, moving and resizing, and the panel's or native window's
actions. Panels (and the native window) appear only after a moment (no flash on a hiccup). The server's output goes
to `~/.config/Sessionary/logs/backend.log`. `/api` and `/gateway` are proxied to whichever server is current; while
there is none they answer 503.

**Why a native window first.** Creating the real window means loading the embedded WebView, which costs real memory
before there is anything to show in it (see [Memory: native startup, idle teardown](#memory-native-startup-idle-teardown)
below). `ensureMain` (`launch.go`) is the one place that decides which window exists: it opens the real one once
`PhaseReady` is reached, and otherwise keeps the native one up to date — created only after `startupDelay` (350 ms),
so a server that answers at once never flashes it. Once the real window has opened for the first time, later hiccups
(`reconnecting`, `restarting`, a later `failed`) show inside it instead of tearing it down, which is why the table
above says "the panel" for those once `main_` exists.

**Environment.** An app started from a launcher or desktop entry gets the session's `PATH`, not the terminal's, so
agents installed in `~/.local/bin`, npm/pnpm globals or a version manager would be "not found on PATH". On Linux and
macOS the server is therefore started with the user's login shell's `PATH` (`shell -ilc`, 8 s limit, markers around
the answer, merged with the current one; any failure keeps the current `PATH`). On Windows, a process started from a
shortcut or the Start menu inherits `explorer.exe`'s environment block from when the desktop session began, not the
registry as it stands now, so installing an agent (Node, nvm, pnpm's global bin, the agent's own installer) is
invisible until the next sign-out or reboot unless this is read fresh; the fix there is to read the Machine and User
`Path` values straight out of the registry instead, as a newly opened terminal would. Both resolve to the same
`PATH`, merged with whatever the process already had so nothing is lost — `internal/shellenv`. Detection,
re-detection and starting an agent all read the server's own environment, so they agree.

**Development:** with a server already on `:4777`, a dev build uses it and, if it goes away, waits for it ("Waiting
for the development server on port 4777…") rather than starting another one beside it. `SESSIONARY_DESKTOP_ATTACH=0`
always starts the app's own.

## Window controls

One strategy per OS, chosen by `GOOS` and nothing else — no code looks at X11 versus Wayland, the compositor or the
desktop environment:

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

## Memory: native startup, idle teardown

Measured on Linux (`.deb`, a 2,000-session history, under Xvfb): the WebView is the app's real memory cost, and it
is almost entirely independent of what the page is showing. Before the backend answers, the native startup window
(above) costs about 230 MB (the Go/GTK/MyGo process; the native `ui` package draws no WebView at all — confirmed by
`internal/linux/window.go`'s `if o.Surface { createSurface() } else { createWebView() }`, `Surface` being true
exactly when `WindowOptions.Content` is set). Once the real window opens, WebKitGTK's own multi-process model adds
a `WebKitWebProcess` (~260 MB) and a `WebKitNetworkProcess` (~65 MB) beside the main one — roughly +400 MB, for
either screen.

That cost is unavoidable while the window is open — it is a web app — but it is **not** given back when the window
is only hidden: `OnHide`/`OnShow` never tore the page down, so a window parked in the tray kept paying it for as
long as the app ran. `memory.go`'s idle teardown fixes that for a window that stays hidden a while:

- Hiding (the close button, Minimize, the tray's own Hide) is unchanged — instant, and the page is exactly where it
  was when shown again.
- If the window is still hidden `idleTeardown` later (5 minutes), it is `Destroy()`ed — not `Close()`: this is not
  the user closing it, and `shouldHideOnClose` would just hide it again. `main_` becomes `nil`.
- Showing it again (the tray, a second launch, a notification) goes through `ensureMain`, the same path as the very
  first launch: the native window if the backend is not ready (rare — only if the server also died while the
  window was torn down), otherwise straight back to the real one. The one visible trade-off is that this reloads
  the page from its root, losing wherever the user was, which is why this does not happen on every hide.
- `OnWindowAllClosed`'s guard (`!shouldHideOnClose()`) covers both this and the native-to-real transition at
  startup: neither is a close that should quit the app while the tray is a way back.

`idleTeardown` is a package var (`0` disables it) rather than a setting — there is nothing for a user to tune here,
only a number that could change if the trade-off needs revisiting.

See also [Architecture](ARCHITECTURE.md) for how this fits with the server and the page.

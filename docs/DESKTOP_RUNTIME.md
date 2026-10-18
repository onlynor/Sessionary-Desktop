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

See also [Architecture](ARCHITECTURE.md) for how this fits with the server and the page.

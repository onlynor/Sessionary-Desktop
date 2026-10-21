package main

import (
	"strconv"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/onlynor/sessionary-desktop/internal/assets"
	"github.com/onlynor/sessionary-desktop/internal/backend"
)

// Before the backend first answers, showing anything at all means loading the embedded web client, so the launch
// screen is drawn natively instead: a small window with no WebView, drawn from Go. Once the backend is ready the
// native window closes and openMain loads the real one; from there on, a restart or a failure shows inside the
// already-loaded page (BackendState, Launch.tsx) rather than tearing it down for a hiccup. The same native window
// also covers the moment the real window is recreated after memory.go's idle teardown closed it while the backend
// is not answering yet (rare: it only happens if the server dies while the window has been hidden for a while).
//
// Showing nothing for a moment avoids a flash for a server that answers at once (the common case); only a startup
// slower than startupDelay, or a failure, creates the window at all.
const startupDelay = 350 * time.Millisecond

var (
	startupMu    sync.Mutex
	startupWin   *mygo.Window
	startupSt    backend.Status
	startupLogs  []string
	startupBusy  bool // a Retry is in flight
	startupShow  bool // "Show Details" was clicked
	startupTimer *time.Timer
)

var startupIcon = sync.OnceValue(func() *ui.Bitmap {
	b, err := ui.DecodeBitmap(assets.Icon)
	if err != nil {
		return nil
	}
	return b
})

// everLaunched is set once the main window has been opened for the first time, so a later wait (the backend
// restarting, or the window reopening after an idle teardown) reads "Restarting Sessionary", not "Sessionary",
// matching Launch.tsx's own distinction.
var everLaunched bool

// ensureMain opens the main window once the backend answers, and otherwise makes sure the native startup window
// is showing (after startupDelay) or up to date. It is the only thing that creates main_: called once at launch,
// from every backend status change (on the backend's own goroutine, so this runs on the main thread like every
// other window operation), and from showMain when the window was closed (by the user, or by an idle teardown)
// while the backend was not ready.
func ensureMain() {
	mygo.RunOnMain(func() {
		s := server.Status()
		startupMu.Lock()
		startupSt = s
		if s.Phase != backend.PhaseFailed {
			startupLogs, startupShow, startupBusy = nil, false, false
		}
		startupMu.Unlock()

		if s.Phase == backend.PhaseReady {
			stopStartupTimer()
			// open the real window before closing the native one, so there is never a moment with no window
			// open at all: OnWindowAllClosed would read as "the app has nothing left to show" and, without a
			// tray, quit it
			if main_ == nil {
				openMain()
				everLaunched = true
			}
			closeStartupWindow()
			return
		}
		if startupWin != nil {
			startupWin.Invalidate()
			return
		}
		startStartupTimer()
	})
}

func startStartupTimer() {
	startupMu.Lock()
	defer startupMu.Unlock()
	if startupTimer != nil {
		return
	}
	startupTimer = time.AfterFunc(startupDelay, func() { mygo.RunOnMain(showStartupWindow) })
}

func stopStartupTimer() {
	startupMu.Lock()
	defer startupMu.Unlock()
	if startupTimer != nil {
		startupTimer.Stop()
		startupTimer = nil
	}
}

// showStartupWindow creates the native window, unless the backend answered while its timer was pending.
func showStartupWindow() {
	if startupWin != nil || server.Status().Phase == backend.PhaseReady {
		return
	}
	startupMu.Lock()
	startupTimer = nil
	startupMu.Unlock()
	w := mygo.NewWindow(mygo.WindowOptions{
		Title: "Sessionary",
		// X, Y left zero: the window is centered
		Width:           440,
		Height:          320,
		DisableResize:   true,
		DisableMaximize: true,
		BackgroundColor: "light-dark(#f6f6f4, #161618)",
		Content:         ui.View(startupView),
	})
	w.SetIcon(assets.Icon)
	w.OnClosed(func() {
		if startupWin == w {
			startupWin = nil
		}
	})
	startupWin = w
}

func closeStartupWindow() {
	if startupWin != nil {
		startupWin.Destroy() // not Close: nothing here needs OnClose's chance to cancel it
	}
}

// startupRetry asks the backend to try again and keeps the button disabled until a new status arrives.
func startupRetry() {
	startupMu.Lock()
	startupBusy = true
	startupMu.Unlock()
	server.Retry()
	if startupWin != nil {
		startupWin.Invalidate()
	}
}

func startupToggleLogs() {
	startupMu.Lock()
	startupShow = !startupShow
	show := startupShow
	startupMu.Unlock()
	if show {
		logs := server.Logs()
		startupMu.Lock()
		startupLogs = logs
		startupMu.Unlock()
	}
	if startupWin != nil {
		startupWin.Invalidate()
	}
}

// describeStatus says what the startup window shows for s, as Launch.tsx's describe() does for the page.
func describeStatus(s backend.Status, launched bool) (title, detail string, busy bool) {
	switch s.Phase {
	case backend.PhaseStarting:
		return "Sessionary", "Starting Sessionary…", true
	case backend.PhaseConnecting:
		title := "Sessionary"
		if launched {
			title = "Restarting Sessionary"
		}
		return title, orDefault(s.Detail, "Connecting…"), true
	case backend.PhaseReconnecting:
		detail := orDefault(s.Detail, "Sessionary isn’t responding…")
		if s.Attached {
			detail = "Waiting for the development server on port " + strconv.Itoa(s.Port) + "…"
		}
		return "Reconnecting", detail, true
	case backend.PhaseRestarting:
		detail := "Sessionary stopped. Starting it again…"
		if s.Attempt > 1 {
			detail = "Sessionary stopped. Starting it again (attempt " + strconv.Itoa(s.Attempt) + ")…"
		}
		return "Restarting Sessionary", detail, true
	case backend.PhaseFailed:
		title := "Sessionary couldn’t start"
		if launched {
			title = "Sessionary is unavailable"
		}
		return title, s.Detail, false
	case backend.PhaseStopped:
		return "Sessionary", "Quitting…", true
	default:
		return "Sessionary", "", false
	}
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func startupView(c *ui.Context) {
	startupMu.Lock()
	s, logs, showLogs, busy := startupSt, startupLogs, startupShow, startupBusy
	startupMu.Unlock()

	title, detail, working := describeStatus(s, everLaunched)
	failed := s.Phase == backend.PhaseFailed
	t := c.Theme()
	ui.Column(c).Fill().Center().Gap(14).Padding(24).Children(func() {
		if icon := startupIcon(); icon != nil {
			ui.Image(c, icon).Width(56).Height(56)
		}
		ui.Text(c, title).FontSize(18).Bold().TextAlign(ui.Center)
		if detail != "" {
			ui.Text(c, detail).FontSize(13).TextColor(t.TextMuted).TextAlign(ui.Center)
		}
		if working {
			ui.Spinner(c)
		}
		if failed {
			ui.Row(c).Gap(8).Children(func() {
				retryLabel := "Try Again"
				if busy {
					retryLabel = "Starting…"
				}
				if ui.PrimaryButton(c.Key("retry"), retryLabel).Disabled(busy).Clicked() {
					startupRetry()
				}
				detailsLabel := "Show Details"
				if showLogs {
					detailsLabel = "Hide Details"
				}
				if ui.Button(c.Key("details"), detailsLabel).Clicked() {
					startupToggleLogs()
				}
				if ui.Button(c.Key("quit"), "Quit").Clicked() {
					mygo.App.Quit()
				}
			})
			if showLogs {
				ui.Scroll(c).MaxHeight(110).Width(380).Children(func() {
					ui.Column(c).Gap(2).Children(func() {
						if len(logs) == 0 {
							ui.Text(c, "Nothing was printed.").FontSize(11).TextColor(t.TextMuted)
						}
						start := 0
						if len(logs) > 20 {
							start = len(logs) - 20
						}
						for _, line := range logs[start:] {
							ui.Text(c, line).FontSize(11).TextColor(t.TextMuted)
						}
					})
				})
				if s.LogPath != "" {
					ui.Text(c, "The full log is in "+s.LogPath).FontSize(11).TextColor(t.TextMuted)
				}
			}
		}
	})
}

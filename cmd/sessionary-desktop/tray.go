package main

import (
	"context"
	"log"
	"runtime"
	"time"

	"github.com/egoist/mygo"

	"github.com/onlynor/sessionary-desktop/internal/assets"
)

// Sessionary's tray icon: the mark from the app's own logo, on transparency. macOS gets the black template and
// tints it to match the menu bar; Linux and Windows get the coloured mark for the notification area. The images
// are 32x32 — a 16pt icon at 2x, and the size macOS scales to the menu bar's 18pt.

var (
	trayIcon         = assets.Tray
	trayIconTemplate = assets.TrayTemplate
)

var (
	tray      *mygo.Tray
	trayReady bool // a tray is installed (see trayUsable: it must also be reachable)
	trayHost  *trayWatch
	trayStop  context.CancelFunc
)

// TrayChanged is sent when the tray becomes reachable or stops being: the page draws Minimize only while it is.
var TrayChanged = mygo.NewEvent[bool]("tray")

// quitting is true once the app is on its way out. Closing the window only hides it — the app keeps serving the
// model gateway while it lives in the tray — so a real quit has to be told apart from a close.
var quitting bool

// installTray puts Sessionary in the menu bar (macOS) or the notification area (Linux, Windows), once the app is
// ready. The tray is the way back to a window that was closed to the tray, and the way to quit for good.
func installTray() {
	icon, template := trayIconFor(runtime.GOOS)
	t, err := mygo.NewTray(mygo.TrayOptions{
		Icon:           icon,
		IconIsTemplate: template,
		ToolTip:        "Sessionary",
		Menu:           trayMenu(),
	})
	if err != nil {
		// linux tray icons use AppIndicator (libayatana-appindicator3): without it the app still runs, without a tray
		log.Println("tray:", err)
		return
	}
	tray, trayReady = t, true
	trayHost = &trayWatch{probe: defaultTrayProbe(), onChange: trayReachable}
	trayHost.check() // before the page asks (Info.Tray)
	var ctx context.Context
	ctx, trayStop = context.WithCancel(context.Background())
	go trayHost.run(ctx, 2*time.Second)
}

// trayReachable reacts to the host appearing or disappearing. A window that was put away must never stay away when
// its way back is gone, so it is shown again at once.
func trayReachable(up bool) {
	_ = TrayChanged.Broadcast(up)
	if !up && main_ != nil && !main_.IsVisible() {
		showMain()
	}
}

// trayIconFor picks the image for a platform: the black template macOS tints to match the menu bar, the coloured
// mark everywhere else.
func trayIconFor(goos string) (icon []byte, template bool) {
	if goos == "darwin" {
		return trayIconTemplate, true
	}
	return trayIcon, false
}

// trayMenu is the tray's menu: bring the window back, put it away, or quit.
func trayMenu() *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Show Sessionary", Click: func(*mygo.MenuItem, *mygo.Window) { showMain() }},
		{Label: "Hide Window", Click: func(*mygo.MenuItem, *mygo.Window) { hideMain() }},
		mygo.Separator(),
		{Label: "Quit Sessionary", Accelerator: "CmdOrCtrl+Q", Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }},
	})
}

// hideMain puts the window away without closing it: the app (and the server it runs) stays up, reachable from
// the tray or the Dock/taskbar.
func hideMain() {
	if main_ != nil {
		main_.Hide()
	}
}

// shouldHideOnClose reports whether closing the window should hide it rather than close it. It hides only while a
// tray is there to get it back from, and never during a quit — so a window is never left invisible with no way
// back (without a tray, closing quits as it always did), and an explicit quit still closes and stops everything.
func shouldHideOnClose() bool { return !quitting && trayUsable() }

// trayInstalled reports whether the app created a tray, for the drive socket to check a development build.
func trayInstalled() bool { return trayReady }

// trayUsable reports whether hiding the window is safe: there is a tray and something is displaying it.
func trayUsable() bool { return trayReady && trayHost != nil && trayHost.Up() }

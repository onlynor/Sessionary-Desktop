// Sessionary Desktop: Sessionary's agent control center as a desktop app.
//
// The window shows Sessionary's web client with a desktop layer on top
// (frontend/), served on a loopback port that also forwards to the Sessionary
// server (internal/backend), which this app starts, or finds already running.
package main

import (
	"log"
	"net/url"
	"os"
	"runtime"

	"github.com/egoist/mygo"

	"github.com/onlynor/sessionary-desktop/frontend"
	"github.com/onlynor/sessionary-desktop/internal/assets"
	"github.com/onlynor/sessionary-desktop/internal/backend"
	"github.com/onlynor/sessionary-desktop/internal/uiserver"
)

// version is set by `mygo build` from mygo.config.ts; "dev" otherwise.
var version = "dev"

var (
	server *backend.Backend
	ui     *uiserver.Server
	main_  *mygo.Window
)

func main() {
	mygo.App.SetName("Sessionary")
	if !mygo.App.RequestSingleInstanceLock() {
		return // the running instance comes to the front
	}
	mygo.App.OnSecondInstance(func(args []string, dir string) { showMain() })

	server = backend.New(backendOptions())
	mygo.Bind(&Desktop{})
	mygo.App.WhenReady(func() {
		var err error
		ui, err = uiserver.Start(uiserver.Options{
			Backend: server.URL,
			Files:   frontend.Files(),
			DevURL:  devURL(),
		})
		if err != nil {
			log.Fatal(err)
		}
		server.Start()
		installTray() // before the page asks whether there is one (Info.Tray)
		openMain()
		installMenu()
		startDrive()
		startSmoke()
	})
	mygo.App.OnActivate(func(visible bool) {
		if !visible {
			showMain()
		}
	})
	mygo.App.OnWindowAllClosed(func() {
		if runtime.GOOS != "darwin" {
			mygo.App.Quit()
		}
	})
	// a quit is not a close: the window's close button only hides it, so quitting must say so first
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { quitting = true })
	mygo.App.OnQuit(func() {
		if tray != nil {
			tray.Destroy()
			tray = nil
		}
		trayReady = false
		if trayStop != nil {
			trayStop()
		}
		server.Stop()
		if ui != nil {
			ui.Close()
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
	if smokeFailed {
		os.Exit(1)
	}
}

func openMain() {
	main_ = mygo.NewWindow(windowOptions(ui.URL + "/"))
	win := main_
	win.SetIcon(assets.Icon) // the taskbar's icon for a build run without its desktop entry (Linux, Windows)
	win.OnReadyToShow(win.Show)
	followState(win)
	page := win.Page()
	// The app's own pages stay in the window; links the user follows to the web open in the browser. Nothing the
	// user did not ask for is ever handed to the browser (a frame's load, a redirect): it is only refused.
	page.OnWillNavigate(func(e *mygo.NavigateEvent) {
		u, err := url.Parse(e.URL)
		if err != nil {
			e.PreventDefault()
			return
		}
		origin := u.Scheme + "://" + u.Host
		if origin == ui.URL || u.Scheme == "about" {
			return
		}
		e.PreventDefault()
		if e.UserInitiated && (u.Scheme == "http" || u.Scheme == "https" || u.Scheme == "mailto") {
			go mygo.Shell.OpenExternal(e.URL)
		}
	})
	page.OnRenderProcessGone(func(reason string) {
		log.Println("page process gone:", reason)
		page.Reload()
	})
	win.OnClosed(func() { main_ = nil })
	// Closing the window hides it: the app keeps running in the tray (its server keeps serving the gateway).
	// Quitting — the tray, the menu, Ctrl+Q — closes it for real, because `quitting` is set first.
	win.OnClose(func(e *mygo.CloseEvent) {
		if shouldHideOnClose() {
			e.PreventDefault()
			win.Hide()
		}
	})
}

func showMain() {
	if main_ == nil {
		openMain()
		return
	}
	main_.Restore()
	main_.Show()
	main_.Focus()
}

// devURL is the frontend's dev server while developing (`mygo dev` sets it).
func devURL() string {
	if !mygo.IsDev() {
		return ""
	}
	if u := os.Getenv("SESSIONARY_DESKTOP_DEV_URL"); u != "" {
		return u
	}
	return os.Getenv("MYGO_DEV_URL")
}

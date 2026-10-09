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

	"github.com/onlynor/sessionary-desktop/internal/backend"
	"github.com/onlynor/sessionary-desktop/internal/uiserver"
)

// version is set by `mygo build` from mygo.config.ts; "dev" otherwise.
var version = "dev"

var (
	blur   bool // the web view draws backdrop-filter (see configureRenderer)
	server *backend.Backend
	ui     *uiserver.Server
	main_  *mygo.Window
)

func main() {
	mygo.App.SetName("Sessionary")
	blur = configureRenderer()
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
			Files:   frontendFiles(),
			DevURL:  devURL(),
		})
		if err != nil {
			log.Fatal(err)
		}
		server.Start()
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
	mygo.App.OnQuit(func() {
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
	win.SetIcon(appIcon) // the taskbar's icon for a build run without its desktop entry (Linux, Windows)
	win.OnReadyToShow(win.Show)
	followState(win)
	page := win.Page()
	// the app's own pages stay in the window; links to the web open in the browser
	page.OnWillNavigate(func(e *mygo.NavigateEvent) {
		u, err := url.Parse(e.URL)
		if err != nil || (u.Scheme+"://"+u.Host) == ui.URL {
			return
		}
		if u.Scheme == "http" || u.Scheme == "https" || u.Scheme == "mailto" {
			e.PreventDefault()
			go mygo.Shell.OpenExternal(e.URL)
		}
	})
	page.OnRenderProcessGone(func(reason string) {
		log.Println("page process gone:", reason)
		page.Reload()
	})
	win.OnClosed(func() { main_ = nil })
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

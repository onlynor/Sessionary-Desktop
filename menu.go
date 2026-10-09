package main

import (
	"runtime"

	"github.com/egoist/mygo"
)

// installMenu gives macOS its menu bar. On Linux and Windows the window keeps
// its whole height for the page, which draws its own toolbar and answers the
// same shortcuts.
func installMenu() {
	if runtime.GOOS != "darwin" {
		return
	}
	cmd := func(c string) func(*mygo.MenuItem, *mygo.Window) {
		return func(_ *mygo.MenuItem, w *mygo.Window) {
			if w == nil {
				w = main_
			}
			if w != nil {
				_ = Command.Emit(w, c)
			}
		}
	}
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "File", Submenu: []*mygo.MenuItem{
			{Label: "New Session…", Accelerator: "CmdOrCtrl+N", Click: cmd("new-session")},
			{Label: "Settings…", Accelerator: "CmdOrCtrl+,", Click: cmd("settings")},
			{Label: "Open Terminal", Accelerator: "CmdOrCtrl+T", Click: cmd("terminal")},
			mygo.Separator(),
			{Role: mygo.RoleClose},
		}},
		{Role: mygo.RoleEditMenu},
		{Label: "View", Submenu: []*mygo.MenuItem{
			{Label: "Search…", Accelerator: "CmdOrCtrl+K", Click: cmd("palette")},
			{Label: "Toggle Sidebar", Accelerator: "CmdOrCtrl+Alt+S", Click: cmd("sidebar")},
			mygo.Separator(),
			{Label: "Home", Accelerator: "CmdOrCtrl+1", Click: cmd("go:home")},
			{Label: "Projects", Accelerator: "CmdOrCtrl+2", Click: cmd("go:projects")},
			{Label: "Sessions", Accelerator: "CmdOrCtrl+3", Click: cmd("go:sessions")},
			{Label: "Models", Accelerator: "CmdOrCtrl+4", Click: cmd("go:models")},
			{Label: "Usage", Accelerator: "CmdOrCtrl+5", Click: cmd("go:usage")},
			mygo.Separator(),
			{Role: mygo.RoleResetZoom}, {Role: mygo.RoleZoomIn}, {Role: mygo.RoleZoomOut},
			mygo.Separator(),
			{Role: mygo.RoleToggleFullScreen},
			{Role: mygo.RoleToggleDevTools},
		}},
		{Role: mygo.RoleWindowMenu},
	}))
}

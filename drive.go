package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/egoist/mygo"
)

// startDrive lets scripts drive a development build (scripts/drive.py): one
// JSON request per line on the Unix socket SESSIONARY_DESKTOP_DRIVE, one JSON
// answer per line. It exists for screenshots and checks during development,
// never in production builds.
func startDrive() {
	path := os.Getenv("SESSIONARY_DESKTOP_DRIVE")
	if path == "" || !mygo.IsDev() {
		return
	}
	_ = os.Remove(path)
	l, err := net.Listen("unix", path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "drive:", err)
		return
	}
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go serveDrive(c)
		}
	}()
}

type driveReq struct {
	Eval   string `json:"eval"`   // JavaScript, whose value is answered
	Size   []int  `json:"size"`   // [width, height] of the window
	Shot   string `json:"shot"`   // a path: what the window shows, as a PNG
	Theme  string `json:"theme"`  // light, dark or system: the appearance
	Wait   int    `json:"wait"`   // milliseconds to wait first
	Status bool   `json:"status"` // the backend's status
	Tray   bool   `json:"tray"`   // whether the tray is installed, and whether the window shows
	Window string `json:"window"` // "state", or an action: minimize, maximize, unmaximize, fullscreen, unfullscreen, close, hide, show, quit
}

func serveDrive(c net.Conn) {
	defer c.Close()
	sc := bufio.NewScanner(c)
	sc.Buffer(make([]byte, 1<<20), 8<<20)
	enc := json.NewEncoder(c)
	for sc.Scan() {
		var req driveReq
		if err := json.Unmarshal(sc.Bytes(), &req); err != nil {
			_ = enc.Encode(map[string]any{"error": err.Error()})
			continue
		}
		res, err := drive(req)
		if err != nil {
			_ = enc.Encode(map[string]any{"error": err.Error()})
			continue
		}
		_ = enc.Encode(map[string]any{"ok": true, "value": res})
	}
}

func drive(req driveReq) (any, error) {
	if req.Wait > 0 {
		time.Sleep(time.Duration(req.Wait) * time.Millisecond)
	}
	w := main_
	if w == nil {
		return nil, fmt.Errorf("no window")
	}
	switch {
	case req.Status:
		return server.Status(), nil
	case req.Tray:
		return map[string]any{"installed": trayInstalled(), "visible": w.IsVisible()}, nil
	case req.Window != "":
		switch req.Window {
		case "minimize":
			w.Minimize()
		case "maximize":
			w.Maximize()
		case "unmaximize":
			w.Unmaximize()
		case "fullscreen":
			w.SetFullScreen(true)
		case "unfullscreen":
			w.SetFullScreen(false)
		case "close":
			w.Close()
			return map[string]any{"visible": w.IsVisible()}, nil
		case "hide":
			hideMain()
		case "show":
			showMain()
		case "quit":
			// answer first, then quit, so the caller sees the reply before the socket goes
			go func() { time.Sleep(150 * time.Millisecond); mygo.App.Quit() }()
			return map[string]any{"quitting": true}, nil
		}
		time.Sleep(300 * time.Millisecond)
		b := w.Bounds()
		return map[string]any{"state": stateOf(w), "bounds": b, "minimized": w.IsMinimized(), "visible": w.IsVisible()}, nil
	case req.Theme != "":
		src := map[string]mygo.ThemeSource{"light": mygo.ThemeLight, "dark": mygo.ThemeDark, "system": mygo.ThemeSystem}[req.Theme]
		mygo.Theme.SetSource(src)
		return nil, nil
	case len(req.Size) == 2:
		w.SetSize(req.Size[0], req.Size[1])
		return nil, nil
	case req.Shot != "":
		png, err := w.CapturePage()
		if err != nil {
			return nil, err
		}
		return nil, os.WriteFile(req.Shot, png, 0o644)
	case req.Eval != "":
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		return w.Page().EvalContext(ctx, req.Eval)
	}
	return nil, nil
}

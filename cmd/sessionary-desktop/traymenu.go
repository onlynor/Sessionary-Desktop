package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/egoist/mygo"
)

// The tray menu's words come from the page: Sessionary's own catalog (web/src/locales.ts plus the desktop layer's
// rows) translates them into the language the user chose, so there is exactly one set of translations. The page
// sends the strings whenever the language changes (frontend/src/desktop/tray.ts); the app keeps the last set in
// tray.json so the next launch's menu is right before the page has loaded. These English defaults are what a first
// launch shows until then; frontend/test/tray.test.ts keeps them in step with the page's keys.
var trayDefaults = map[string]string{
	"show":     "Show Sessionary",
	"hide":     "Hide Sessionary",
	"new":      "New session",
	"recent":   "Recent sessions",
	"models":   "Model Control",
	"usage":    "Usage",
	"settings": "Settings",
	"about":    "About Sessionary",
	"quit":     "Quit Sessionary",
	"tooltip":  "Sessionary",
}

// TrayRecent is a session the tray offers: where the page opens it (its own route) and the title it shows.
type TrayRecent struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Route string `json:"route"`
}

const trayRecentMax = 3

// trayModel is everything the menu is built from.
type trayModel struct {
	Labels  map[string]string
	Recents []TrayRecent
	Visible bool // the window is on screen
}

func (m trayModel) label(key string) string {
	if s := strings.TrimSpace(m.Labels[key]); s != "" {
		return s
	}
	return trayDefaults[key]
}

// trayActions is what the menu's entries do. They reuse the application's own commands (the page's routes, its New
// Session flow, its Settings menu); the menu implements no navigation of its own.
type trayActions struct {
	toggle  func()       // show the window, or put it away
	command func(string) // reveal and focus the window, then ask the page (Command)
	about   func()
	quit    func()
}

func trimTitle(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 48 {
		return string(r[:47]) + "…"
	}
	return s
}

// trayItems builds the menu: window and sessions, then places, then the application.
func trayItems(m trayModel, a trayActions) []*mygo.MenuItem {
	do := func(cmd string) func(*mygo.MenuItem, *mygo.Window) {
		return func(*mygo.MenuItem, *mygo.Window) { a.command(cmd) }
	}
	toggle := m.label("hide")
	if !m.Visible {
		toggle = m.label("show")
	}
	items := []*mygo.MenuItem{
		{ID: "toggle", Label: toggle, Click: func(*mygo.MenuItem, *mygo.Window) { a.toggle() }},
		{ID: "new", Label: m.label("new"), Click: do("new-session")},
	}
	var recents []*mygo.MenuItem
	for i, r := range m.Recents {
		if i == trayRecentMax {
			break
		}
		if r.Route == "" || strings.TrimSpace(r.Title) == "" {
			continue
		}
		recents = append(recents, &mygo.MenuItem{Label: trimTitle(r.Title), Click: do("go:" + r.Route)})
	}
	if len(recents) > 0 {
		items = append(items, &mygo.MenuItem{ID: "recent", Label: m.label("recent"), Submenu: recents})
	}
	items = append(items,
		mygo.Separator(),
		&mygo.MenuItem{ID: "models", Label: m.label("models"), Click: do("go:models")},
		&mygo.MenuItem{ID: "usage", Label: m.label("usage"), Click: do("go:usage")},
		&mygo.MenuItem{ID: "settings", Label: m.label("settings"), Click: do("settings")},
		mygo.Separator(),
		&mygo.MenuItem{ID: "about", Label: m.label("about"), Click: func(*mygo.MenuItem, *mygo.Window) { a.about() }},
		&mygo.MenuItem{ID: "quit", Label: m.label("quit"), Accelerator: "CmdOrCtrl+Q", Click: func(*mygo.MenuItem, *mygo.Window) { a.quit() }},
	)
	return items
}

// ---------- the last labels the page sent, kept between launches ----------

var trayState = struct {
	sync.Mutex
	labels  map[string]string
	recents []TrayRecent
}{}

func trayLabelsPath() string {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "tray.json")
}

func loadTrayLabels() {
	trayState.Lock()
	defer trayState.Unlock()
	if trayState.labels != nil {
		return
	}
	trayState.labels = map[string]string{}
	if p := trayLabelsPath(); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			_ = json.Unmarshal(b, &trayState.labels)
		}
	}
}

func currentTrayModel(visible bool) trayModel {
	loadTrayLabels()
	trayState.Lock()
	defer trayState.Unlock()
	labels := make(map[string]string, len(trayState.labels))
	for k, v := range trayState.labels {
		labels[k] = v
	}
	return trayModel{Labels: labels, Recents: append([]TrayRecent(nil), trayState.recents...), Visible: visible}
}

// SetTray receives the menu's words (in the user's language) and recent sessions from the page.
func (Desktop) SetTray(labels map[string]string, recents []TrayRecent) {
	loadTrayLabels()
	clean := map[string]string{}
	for k := range trayDefaults {
		if s := strings.TrimSpace(labels[k]); s != "" {
			clean[k] = s
		}
	}
	trayState.Lock()
	changed := !sameLabels(trayState.labels, clean)
	trayState.labels, trayState.recents = clean, recents
	trayState.Unlock()
	if changed {
		if p := trayLabelsPath(); p != "" {
			if b, err := json.Marshal(clean); err == nil {
				_ = os.WriteFile(p, b, 0o600)
			}
		}
	}
	refreshTray()
}

func sameLabels(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

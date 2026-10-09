package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/egoist/mygo"
)

// The page keeps its settings (theme, language, pane widths, recent sessions)
// in localStorage, which belongs to the page's origin — a loopback port that
// changes from launch to launch. The page mirrors them here instead, in
// prefs.json in the user data directory, and restores them before it starts.
var prefs = struct {
	sync.Mutex
	loaded bool
	values map[string]string
}{}

func prefsPath() string {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "prefs.json")
}

func loadPrefsLocked() {
	if prefs.loaded {
		return
	}
	prefs.loaded = true
	prefs.values = map[string]string{}
	if p := prefsPath(); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			_ = json.Unmarshal(b, &prefs.values)
		}
	}
}

// Prefs returns the page's saved settings.
func (Desktop) Prefs() map[string]string {
	prefs.Lock()
	defer prefs.Unlock()
	loadPrefsLocked()
	out := make(map[string]string, len(prefs.values))
	for k, v := range prefs.values {
		out[k] = v
	}
	return out
}

// SavePrefs records changed settings; a nil value removes one.
func (Desktop) SavePrefs(changes map[string]*string) error {
	prefs.Lock()
	defer prefs.Unlock()
	loadPrefsLocked()
	for k, v := range changes {
		if v == nil {
			delete(prefs.values, k)
		} else {
			prefs.values[k] = *v
		}
	}
	p := prefsPath()
	if p == "" {
		return nil
	}
	b, err := json.MarshalIndent(prefs.values, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

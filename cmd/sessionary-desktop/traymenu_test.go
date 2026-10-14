package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo"
)

type recorder struct{ calls []string }

func (r *recorder) actions() trayActions {
	return trayActions{
		toggle:  func() { r.calls = append(r.calls, "toggle") },
		command: func(c string) { r.calls = append(r.calls, "cmd:"+c) },
		about:   func() { r.calls = append(r.calls, "about") },
		quit:    func() { r.calls = append(r.calls, "quit") },
	}
}

func labelsOf(items []*mygo.MenuItem) []string {
	var out []string
	for _, it := range items {
		if it.Type == mygo.MenuItemSeparator {
			out = append(out, "-")
		} else {
			out = append(out, it.Label)
		}
	}
	return out
}

func TestTrayMenuStructureAndGroups(t *testing.T) {
	var r recorder
	m := trayModel{Visible: true, Recents: []TrayRecent{{ID: "a", Title: "Fix the bug", Route: "#/m/local/s/a"}}}
	got := labelsOf(trayItems(m, r.actions()))
	want := []string{"Hide Sessionary", "New session", "Recent sessions", "-", "Model Control", "Usage", "Settings", "-", "About Sessionary", "Quit Sessionary"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("menu = %v\nwant   %v", got, want)
	}
}

func TestShowHideFollowsTheWindow(t *testing.T) {
	var r recorder
	if got := trayItems(trayModel{Visible: true}, r.actions())[0].Label; got != "Hide Sessionary" {
		t.Errorf("visible window offers %q", got)
	}
	if got := trayItems(trayModel{Visible: false}, r.actions())[0].Label; got != "Show Sessionary" {
		t.Errorf("hidden window offers %q", got)
	}
}

func TestNoRecentsMeansNoRecentEntry(t *testing.T) {
	var r recorder
	for _, rec := range [][]TrayRecent{nil, {{ID: "x", Title: "  ", Route: "#/x"}}, {{ID: "y", Title: "t", Route: ""}}} {
		for _, l := range labelsOf(trayItems(trayModel{Recents: rec}, r.actions())) {
			if l == "Recent sessions" {
				t.Errorf("an empty Recent submenu is a placeholder: %v", rec)
			}
		}
	}
}

func TestRecentsAreLastThreeWithShortTitles(t *testing.T) {
	var r recorder
	long := strings.Repeat("长", 80) + "\nsecond line"
	m := trayModel{Recents: []TrayRecent{
		{ID: "1", Title: long, Route: "#/1"}, {ID: "2", Title: "two", Route: "#/2"}, {ID: "3", Title: "three", Route: "#/3"}, {ID: "4", Title: "four", Route: "#/4"},
	}}
	var sub []*mygo.MenuItem
	for _, it := range trayItems(m, r.actions()) {
		if it.ID == "recent" {
			sub = it.Submenu
		}
	}
	if len(sub) != 3 {
		t.Fatalf("recent sessions = %d, want the last three", len(sub))
	}
	if n := len([]rune(sub[0].Label)); n > 48 || strings.Contains(sub[0].Label, "\n") {
		t.Errorf("title is not one short line (%d runes): %q", n, sub[0].Label)
	}
	sub[1].Click(sub[1], nil)
	if !reflect.DeepEqual(r.calls, []string{"cmd:go:#/2"}) {
		t.Errorf("a recent session opens its own route: %v", r.calls)
	}
}

// each entry asks the page for the command or route it already has, and nothing else
func TestActionsRouteToExistingCommands(t *testing.T) {
	var r recorder
	byID := map[string]*mygo.MenuItem{}
	for _, it := range trayItems(trayModel{}, r.actions()) {
		byID[it.ID] = it
	}
	for id, want := range map[string]string{
		"toggle": "toggle", "new": "cmd:new-session", "models": "cmd:go:models", "usage": "cmd:go:usage",
		"settings": "cmd:settings", "about": "about", "quit": "quit",
	} {
		r.calls = nil
		byID[id].Click(byID[id], nil)
		if !reflect.DeepEqual(r.calls, []string{want}) {
			t.Errorf("%s did %v, want %s", id, r.calls, want)
		}
	}
	if byID["quit"].Accelerator == "" {
		t.Error("Quit has no accelerator")
	}
}

// labels come from the page's catalog; whatever it does not send is English, so a menu is never blank
func TestLabelsFallBackToEnglish(t *testing.T) {
	m := trayModel{Labels: map[string]string{"models": "模型控制", "usage": "  "}}
	if got := m.label("models"); got != "模型控制" {
		t.Errorf("sent label: %q", got)
	}
	if got := m.label("usage"); got != "Usage" {
		t.Errorf("blank label falls back: %q", got)
	}
	if got := m.label("quit"); got != "Quit Sessionary" {
		t.Errorf("missing label falls back: %q", got)
	}
	var r recorder
	got := labelsOf(trayItems(trayModel{Visible: true, Labels: map[string]string{"hide": "隐藏 Sessionary", "new": "新建会话", "quit": "退出 Sessionary"}}, r.actions()))
	if got[0] != "隐藏 Sessionary" || got[1] != "新建会话" || got[len(got)-1] != "退出 Sessionary" {
		t.Errorf("zh-CN labels: %v", got)
	}
}

func TestSameLabels(t *testing.T) {
	if !sameLabels(map[string]string{"a": "1"}, map[string]string{"a": "1"}) || sameLabels(map[string]string{"a": "1"}, map[string]string{"a": "2"}) || sameLabels(nil, map[string]string{"a": "1"}) {
		t.Error("sameLabels")
	}
}

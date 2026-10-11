package main

import "testing"

func TestNormalizeLayout(t *testing.T) {
	for in, want := range map[string]string{
		"appmenu:close":                    ":close",
		"menu:minimize,maximize,close":     ":minimize,maximize,close",
		"close,minimize,maximize:":         "close,minimize,maximize:",
		"icon,close:appmenu":               "close:",
		"minimize,maximize,close":          ":minimize,maximize,close",
		"":                                 ":",
		" close , spacer , maximize :menu": "close,maximize:",
	} {
		if got := normalizeLayout(in); got != want {
			t.Errorf("normalizeLayout(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLayoutOnBareCompositors(t *testing.T) {
	t.Setenv("SESSIONARY_WINDOW_BUTTONS", "")
	t.Setenv("XDG_CURRENT_DESKTOP", "niri")
	t.Setenv("DESKTOP_SESSION", "niri")
	t.Setenv("NIRI_SOCKET", "/run/user/1000/niri.sock")
	if desktopOwnsLayout() || canMinimize() {
		t.Fatal("niri neither owns the layout nor minimizes")
	}
	if got := withoutButton(":minimize,maximize,close", "minimize"); got != ":maximize,close" {
		t.Fatalf("withoutButton = %q", got)
	}
	t.Setenv("NIRI_SOCKET", "")
	t.Setenv("XDG_CURRENT_DESKTOP", "ubuntu:GNOME")
	if !desktopOwnsLayout() || !canMinimize() {
		t.Fatal("GNOME owns the layout and minimizes")
	}
	t.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	if !desktopOwnsLayout() {
		t.Fatal("KDE owns the layout")
	}
}

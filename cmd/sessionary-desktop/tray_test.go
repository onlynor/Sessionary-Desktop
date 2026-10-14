package main

import (
	"bytes"
	"image/png"
	"reflect"
	"testing"

	"github.com/egoist/mygo"
)

func TestTrayIconFor(t *testing.T) {
	if icon, template := trayIconFor("darwin"); !template || !bytes.Equal(icon, trayIconTemplate) {
		t.Error("macOS uses the black template icon")
	}
	if icon, template := trayIconFor("linux"); template || !bytes.Equal(icon, trayIcon) {
		t.Error("Linux uses the coloured icon")
	}
	if icon, template := trayIconFor("windows"); template || !bytes.Equal(icon, trayIcon) {
		t.Error("Windows uses the coloured icon")
	}
	// both are real PNGs, 32x32: a 16pt icon at 2x, and what macOS scales to its menu bar's 18pt
	for name, b := range map[string][]byte{"tray.png": trayIcon, "trayTemplate.png": trayIconTemplate} {
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			t.Fatalf("%s is not a PNG: %v", name, err)
		}
		if w, h := img.Bounds().Dx(), img.Bounds().Dy(); w != 32 || h != 32 {
			t.Errorf("%s is %dx%d, want 32x32", name, w, h)
		}
	}
}

func TestTrayMenu(t *testing.T) {
	items := trayMenu().Items()
	labels := make([]string, 0, len(items))
	for _, it := range items {
		if it.Type == mygo.MenuItemSeparator {
			labels = append(labels, "-")
			continue
		}
		labels = append(labels, it.Label)
	}
	if want := []string{"Show Sessionary", "Hide Window", "-", "Quit Sessionary"}; !reflect.DeepEqual(labels, want) {
		t.Fatalf("tray menu = %v, want %v", labels, want)
	}
	// every entry does something: showing, hiding, quitting
	for i, it := range items {
		if i == 2 {
			continue
		}
		if it.Click == nil {
			t.Errorf("%q has no action", it.Label)
		}
	}
	// quitting is explicit, and reachable from the keyboard
	if quit := items[len(items)-1]; quit.Accelerator == "" {
		t.Error("Quit has no accelerator")
	}
}

func TestWindowCloseHidesOnlyWithAReachableTray(t *testing.T) {
	host := &trayWatch{probe: func() bool { return true }}
	oldHost := trayHost
	t.Cleanup(func() { quitting, trayReady, trayHost = false, false, oldHost })

	quitting, trayReady, trayHost = false, false, host
	host.check()
	// no tray (the AppIndicator library is missing): closing must quit rather than leave an invisible app
	if shouldHideOnClose() {
		t.Error("without a tray, closing quits")
	}
	trayReady = true
	if !shouldHideOnClose() || !trayUsable() {
		t.Error("with a displayed tray, closing hides the window")
	}
	// the tray exists but nothing displays it: hiding would strand the window
	host.probe = func() bool { return false }
	host.check()
	if shouldHideOnClose() || trayUsable() {
		t.Error("a tray no host displays is not a way back: closing quits")
	}
	host.probe = func() bool { return true }
	host.check()
	quitting = true
	if shouldHideOnClose() {
		t.Error("a quit closes the window")
	}
}

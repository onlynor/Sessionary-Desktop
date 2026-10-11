package main

import (
	"os"
	"strings"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/egoist/mygo"
)

// Window chrome on Linux.
//
// MyGo's hidden title bar puts GTK's own title buttons over the page, unless the
// Wayland compositor says it prefers server-side decorations, in which case it
// shows none and leaves the buttons to the compositor's title bar. That holds for
// KWin (a title bar of its own, then doubled by the page's toolbar) and not for
// tiling compositors such as niri, which prefer server-side decorations and draw
// no title bar: the window ends up with no buttons at all.
//
// So on Linux the window is frameless: MyGo tells the compositor the window
// decorates itself (no compositor title bar is added, on any compositor), and
// the page draws the title buttons, which call MyGo's window methods. Which
// buttons and on which side come from gtk-decoration-layout, the setting GTK's
// own header bars follow (GNOME, KDE and Xfce keep it in step with their window
// managers).
func chromeOptions(o *mygo.WindowOptions) {
	o.Frameless = true
	o.TitleBarStyle = ""
}

// buttonLayout is gtk-decoration-layout as "start:end", e.g.
// "close:" or ":minimize,maximize,close", with only window buttons kept.
//
// GTK reads it from the desktop. Where no desktop owns it — a bare compositor
// such as niri, Sway, Hyprland or i3 — the value GTK reports is GNOME's default
// (close only), which describes no one's preference; the window then shows all
// three buttons on the side the value names. SESSIONARY_WINDOW_BUTTONS (same
// format) overrides both.
func buttonLayout(w *mygo.Window) string {
	if v := os.Getenv("SESSIONARY_WINDOW_BUTTONS"); v != "" {
		return normalizeLayout(v)
	}
	layout := normalizeLayout(gtkDecorationLayout())
	if !desktopOwnsLayout() {
		start, _, _ := strings.Cut(layout, ":")
		if strings.Contains(start, "close") {
			layout = "close,maximize,minimize:"
		} else {
			layout = ":minimize,maximize,close"
		}
	}
	if onX11() {
		// the window manager says what it allows this window; one that says nothing promises only to close it
		allowed, ok := allowedActions(w)
		if !ok || !allowed["MINIMIZE"] {
			layout = withoutButton(layout, "minimize")
		}
		if !ok || !allowed["MAXIMIZE_HORZ"] || !allowed["MAXIMIZE_VERT"] {
			layout = withoutButton(layout, "maximize")
		}
		return layout
	}
	if !canMinimize() {
		layout = withoutButton(layout, "minimize")
	}
	return layout
}

// canMinimize reports whether the compositor minimizes windows. Wayland tells
// clients (xdg_toplevel.wm_capabilities), but GTK 3 drops what it says, so the
// compositors known to have no minimized state are listed here: a button that
// does nothing is worse than none. niri advertises maximize and fullscreen
// only; Sway, Hyprland and i3 (X11) have no minimize either.
func canMinimize() bool {
	if os.Getenv("NIRI_SOCKET") != "" || os.Getenv("SWAYSOCK") != "" || os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != "" || os.Getenv("I3SOCK") != "" {
		return false
	}
	d := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	for _, name := range []string{"niri", "sway", "hyprland", "i3"} {
		if strings.Contains(d, name) {
			return false
		}
	}
	return true
}

func withoutButton(layout, button string) string {
	start, end, _ := strings.Cut(layout, ":")
	drop := func(side string) string {
		var out []string
		for _, b := range strings.Split(side, ",") {
			if b != "" && b != button {
				out = append(out, b)
			}
		}
		return strings.Join(out, ",")
	}
	return drop(start) + ":" + drop(end)
}

// desktopOwnsLayout reports a desktop environment that keeps gtk-decoration-layout
// in step with its own window decorations.
func desktopOwnsLayout() bool {
	d := strings.ToUpper(os.Getenv("XDG_CURRENT_DESKTOP") + ":" + os.Getenv("DESKTOP_SESSION"))
	for _, name := range []string{"GNOME", "KDE", "XFCE", "CINNAMON", "MATE", "BUDGIE", "PANTHEON", "UNITY", "LXQT", "COSMIC", "DEEPIN"} {
		if strings.Contains(d, name) {
			return true
		}
	}
	return false
}

func normalizeLayout(v string) string {
	start, end, found := strings.Cut(v, ":")
	if !found {
		end, start = start, ""
	}
	keep := func(side string) string {
		var out []string
		for _, b := range strings.Split(side, ",") {
			switch b = strings.TrimSpace(b); b {
			case "minimize", "maximize", "close":
				out = append(out, b)
			}
		}
		return strings.Join(out, ",")
	}
	return keep(start) + ":" + keep(end)
}

var gtkLayout struct {
	settingsDefault func() uintptr
	objectGet       func(obj uintptr, name *byte, out unsafe.Pointer, end uintptr)
	free            func(p *byte)
	ok              bool
}

// gtkDecorationLayout reads GTK's setting; it runs on the main thread, once GTK
// is up (MyGo has loaded GTK by then, so dlopen finds it in the process).
func gtkDecorationLayout() string {
	if !gtkLayout.ok {
		gtk, err := purego.Dlopen("libgtk-3.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			return ":close"
		}
		gobject, err := purego.Dlopen("libgobject-2.0.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			return ":close"
		}
		glib, err := purego.Dlopen("libglib-2.0.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			return ":close"
		}
		purego.RegisterLibFunc(&gtkLayout.settingsDefault, gtk, "gtk_settings_get_default")
		purego.RegisterLibFunc(&gtkLayout.objectGet, gobject, "g_object_get")
		purego.RegisterLibFunc(&gtkLayout.free, glib, "g_free")
		gtkLayout.ok = true
	}
	out := ":close"
	mygo.RunOnMain(func() {
		s := gtkLayout.settingsDefault()
		if s == 0 {
			return
		}
		var p *byte
		name := append([]byte("gtk-decoration-layout"), 0)
		gtkLayout.objectGet(s, &name[0], unsafe.Pointer(&p), 0)
		if p != nil {
			out = goString(p)
			gtkLayout.free(p)
		}
	})
	return out
}

// goString copies a C string GTK returned.
func goString(p *byte) string {
	n := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(p), n)) != 0 {
		n++
	}
	return string(unsafe.Slice(p, n))
}

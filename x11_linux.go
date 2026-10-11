package main

import (
	"os"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/egoist/mygo"
)

// On X11 the window manager says, per window, what it will do with it:
// _NET_WM_ALLOWED_ACTIONS (EWMH), which stacking window managers (KWin, Mutter,
// xfwm4, Openbox…) keep up to date. A window manager that sets none (i3 and
// other tiling or minimal ones) promises nothing but closing.

// onX11 reports whether GTK runs on an X server rather than Wayland.
func onX11() bool {
	if b := os.Getenv("GDK_BACKEND"); b != "" {
		return strings.HasPrefix(b, "x11")
	}
	return os.Getenv("WAYLAND_DISPLAY") == "" && os.Getenv("DISPLAY") != ""
}

var x11 struct {
	once              sync.Once
	ok                bool
	widgetGetWindow   func(widget uintptr) uintptr
	windowGetXID      func(window uintptr) uint64
	openDisplay       func(name *byte) uintptr
	internAtom        func(dpy uintptr, name *byte, onlyIfExists int32) uint64
	getAtomName       func(dpy uintptr, atom uint64) *byte
	getWindowProperty func(dpy uintptr, w uint64, prop uint64, off, length int64, del int32, reqType uint64, actualType *uint64, actualFormat *int32, n *uint64, after *uint64, data **uint64) int32
	free              func(p unsafe.Pointer) int32
	dpy               uintptr
}

func loadX11() bool {
	x11.once.Do(func() {
		gtk, err1 := purego.Dlopen("libgtk-3.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		gdk, err2 := purego.Dlopen("libgdk-3.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		xlib, err3 := purego.Dlopen("libX11.so.6", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err1 != nil || err2 != nil || err3 != nil {
			return
		}
		purego.RegisterLibFunc(&x11.widgetGetWindow, gtk, "gtk_widget_get_window")
		purego.RegisterLibFunc(&x11.windowGetXID, gdk, "gdk_x11_window_get_xid")
		purego.RegisterLibFunc(&x11.openDisplay, xlib, "XOpenDisplay")
		purego.RegisterLibFunc(&x11.internAtom, xlib, "XInternAtom")
		purego.RegisterLibFunc(&x11.getAtomName, xlib, "XGetAtomName")
		purego.RegisterLibFunc(&x11.getWindowProperty, xlib, "XGetWindowProperty")
		purego.RegisterLibFunc(&x11.free, xlib, "XFree")
		x11.dpy = x11.openDisplay(nil) // a connection of our own, for reading properties only
		x11.ok = x11.dpy != 0
	})
	return x11.ok
}

// allowedActions returns the window manager's _NET_WM_ALLOWED_ACTIONS for the
// window, without the _NET_WM_ACTION_ prefix, and whether it set any. It waits a
// moment for a window that was just mapped.
func allowedActions(w *mygo.Window) (map[string]bool, bool) {
	if w == nil || !loadX11() {
		return nil, false
	}
	var xid uint64
	mygo.RunOnMain(func() {
		if gw := x11.widgetGetWindow(w.NativeHandle()); gw != 0 {
			xid = x11.windowGetXID(gw)
		}
	})
	if xid == 0 {
		return nil, false
	}
	name := append([]byte("_NET_WM_ALLOWED_ACTIONS"), 0)
	prop := x11.internAtom(x11.dpy, &name[0], 0)
	const xaAtom = 4
	for try := 0; try < 10; try++ {
		var typ, n, after uint64
		var format int32
		var data *uint64
		if x11.getWindowProperty(x11.dpy, xid, prop, 0, 64, 0, xaAtom, &typ, &format, &n, &after, &data) == 0 && data != nil {
			atoms := unsafe.Slice(data, n) // format 32 is returned as longs
			out := map[string]bool{}
			for _, a := range atoms {
				if p := x11.getAtomName(x11.dpy, a); p != nil {
					out[strings.TrimPrefix(goString(p), "_NET_WM_ACTION_")] = true
					x11.free(unsafe.Pointer(p))
				}
			}
			x11.free(unsafe.Pointer(data))
			if n > 0 {
				return out, true
			}
		}
		time.Sleep(100 * time.Millisecond) // not managed yet
	}
	return nil, false
}

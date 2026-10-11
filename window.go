package main

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/egoist/mygo"
)

// WindowChanged is sent when the window is maximized or restored, enters or
// leaves full screen, or gains or loses the focus.
var WindowChanged = mygo.NewEvent[WindowState]("window")

// WindowState is what the page's title bar draws from.
type WindowState struct {
	Maximized  bool `json:"maximized"`
	FullScreen bool `json:"fullScreen"`
	Focused    bool `json:"focused"`
}

// Chrome says what the page draws of the window's frame.
type Chrome struct {
	// Buttons is the title buttons the page draws, "start:end" as in
	// gtk-decoration-layout (e.g. ":minimize,maximize,close"); empty where the
	// system draws them (macOS, Windows).
	Buttons string `json:"buttons"`
}

// Chrome returns what the page draws of the window's frame.
func (Desktop) Chrome(ctx context.Context) Chrome {
	return Chrome{Buttons: layoutOf(mygo.CallerWindow(ctx))}
}

// layouts are worked out once per window, from when it shows (on X11 that means
// waiting for the window manager to say what it allows), so that the page,
// which asks as it mounts, rarely waits.
var layouts sync.Map // *mygo.Window → func() string

func layoutOf(w *mygo.Window) string {
	f, _ := layouts.LoadOrStore(w, sync.OnceValue(func() string { return buttonLayout(w) }))
	return f.(func() string)()
}

// WindowState returns the calling window's state.
func (Desktop) WindowState(ctx context.Context) WindowState {
	if w := mygo.CallerWindow(ctx); w != nil {
		return stateOf(w)
	}
	return WindowState{}
}

// Quit quits the app, as the window's close button or the Quit menu item does.
func (Desktop) Quit() { mygo.App.Quit() }

func stateOf(w *mygo.Window) WindowState {
	return WindowState{Maximized: w.IsMaximized(), FullScreen: w.IsFullScreen(), Focused: w.IsFocused()}
}

// followState sends the window's state to its page whenever it changes.
//
// A listener runs before the window's reported state has caught up with the
// event (IsMaximized is still the old value inside OnMaximize), so the state is
// read once the main loop has moved on, and again a little later for compositors
// that confirm a change in more than one configure.
func followState(w *mygo.Window) {
	w.OnShow(func() { go layoutOf(w) })
	var mu sync.Mutex
	last := WindowState{}
	emit := func() {
		s := stateOf(w)
		mu.Lock()
		changed := s != last
		last = s
		mu.Unlock()
		if changed {
			_ = WindowChanged.Emit(w, s)
		}
	}
	send := func() {
		go func() {
			time.Sleep(30 * time.Millisecond)
			emit()
			time.Sleep(250 * time.Millisecond)
			emit()
		}()
	}
	for _, on := range []func(func()) func(){w.OnMaximize, w.OnUnmaximize, w.OnEnterFullScreen, w.OnLeaveFullScreen, w.OnFocus, w.OnBlur, w.OnRestore} {
		on(send)
	}
}

// windowOptions are the main window's options for this platform.
func windowOptions(url string) mygo.WindowOptions {
	o := mygo.WindowOptions{
		Title:          "Sessionary",
		URL:            url,
		Width:          1440,
		Height:         900,
		MinWidth:       900,
		MinHeight:      580,
		StateKey:       "main",
		Hidden:         true,
		TitleBarStyle:  mygo.TitleBarHiddenInset,
		TitleBarHeight: 52,
		// what shows until the page paints; the page sets its own theme's once it has
		BackgroundColor: "light-dark(#f6f6f4, #161618)",
		Page:            mygo.PageOptions{TrustedOrigins: []string{url[:len(url)-1]}},
	}
	if runtime.GOOS == "windows" {
		o.TitleBarStyle = mygo.TitleBarHidden // Windows 11's caption buttons over the page
	}
	chromeOptions(&o)
	return o
}

package main

import (
	"context"
	"errors"
	"net/url"
	"runtime"

	"github.com/egoist/mygo"

	"github.com/onlynor/sessionary-desktop/internal/backend"
)

// BackendChanged is sent when the Sessionary server starts, stops or fails.
var BackendChanged = mygo.NewEvent[backend.Status]("backend")

// Command is sent when the app's menu, or the system, asks the page to do
// something: "palette", "new-session", "settings", "go:<route>", …
var Command = mygo.NewEvent[string]("command")

// Desktop is what the page asks of the desktop app.
type Desktop struct{}

// Info describes the app and the system it runs on.
type Info struct {
	Platform string `json:"platform"` // darwin, linux, windows
	Version  string `json:"version"`
	Dev      bool   `json:"dev"`
	Blur     bool   `json:"blur"` // whether materials can blur what is behind them
}

// Info returns facts about the app.
func (Desktop) Info() Info {
	return Info{Platform: runtime.GOOS, Version: version, Dev: mygo.IsDev(), Blur: blur()}
}

// Backend returns the Sessionary server's status.
func (Desktop) Backend() backend.Status { return server.Status() }

// BackendLogs returns the server's recent output, for a failure's details.
func (Desktop) BackendLogs() []string { return server.Logs() }

// Retry starts the server again at once after it failed (or while it waits to).
func (Desktop) Retry() { server.Retry() }

// OpenExternal opens a web address in the default browser.
func (Desktop) OpenExternal(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "mailto") {
		return errors.New("only web addresses open in the browser")
	}
	return mygo.Shell.OpenExternal(raw)
}

// Notify shows a desktop notification; a click brings the window forward and
// tells the page which notice it was.
func (Desktop) Notify(id, title, body string) error {
	n := mygo.NewNotification(mygo.NotificationOptions{ID: id, Title: title, Body: body})
	n.OnClick(func() {
		showMain()
		if main_ != nil && id != "" {
			_ = Command.Emit(main_, "notice:"+id)
		}
	})
	return n.Show()
}

// SetBackground matches the window's background to the page's theme, so that
// resizing and reloading never flash another color.
func (Desktop) SetBackground(ctx context.Context, color string) {
	if w := mygo.CallerWindow(ctx); w != nil {
		w.SetBackgroundColor(color)
	}
}

// SetBadge shows how many things wait for the user on the app's icon.
func (Desktop) SetBadge(n int) { mygo.App.SetBadgeCount(n) }

// Attention asks for the user's attention when the window is in the background.
func (Desktop) Attention(ctx context.Context) {
	if w := mygo.CallerWindow(ctx); w != nil && !w.IsFocused() {
		w.FlashFrame(true)
	}
}

// blur reports whether materials may blur what is behind them. macOS's and Windows's web views composite on the
// GPU; MyGo turns WebKitGTK's DMA-BUF renderer off everywhere for safety, which also drops backdrop-filter, so on
// Linux the page draws its materials opaque instead (`data-no-blur`) rather than guessing per GPU or machine.
func blur() bool { return runtime.GOOS != "linux" }

package main

import (
	"time"

	"github.com/egoist/mygo"
)

// idleTeardown is how long the main window may sit hidden in the tray before Sessionary gives its WebView back to
// the OS. Hiding — the close button, Minimize, the tray's own Hide — is instant and leaves the page exactly where
// it was, so a quick glance away and back costs nothing; the WebView engine is the app's whole memory cost
// (measured on Linux: ~210 MB, essentially independent of what the page is showing) for as long as it is kept
// around, and a window left hidden this long is probably not coming back soon. Reopening it after a teardown
// reloads the page from its root — the one visible trade-off, and why this does not fire on every hide.
//
// 0 disables it (tests, or anyone who would rather keep the memory than pay for a reload).
var idleTeardown = 5 * time.Minute

var idleTimer *time.Timer

// armIdleTeardown starts (or restarts, if the window was shown and hidden again) the countdown to closing a
// hidden window for its memory. Call it from the window's OnHide.
func armIdleTeardown() {
	disarmIdleTeardown()
	if idleTeardown <= 0 {
		return
	}
	idleTimer = time.AfterFunc(idleTeardown, func() {
		// on the main thread, with Show (also serialized there): the window is torn down only if it is still
		// hidden at the moment this actually runs, never racing a Show that landed just before it
		mygo.RunOnMain(func() {
			if main_ != nil && !main_.IsVisible() {
				main_.Destroy() // not Close: this is not the user closing it, and shouldHideOnClose would just hide it again
			}
		})
	})
}

// disarmIdleTeardown cancels a pending teardown. Call it from OnShow and OnClosed.
func disarmIdleTeardown() {
	if idleTimer != nil {
		idleTimer.Stop()
		idleTimer = nil
	}
}

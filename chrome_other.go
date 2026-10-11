//go:build !linux

package main

import "github.com/egoist/mygo"

// On macOS and Windows the system draws the title buttons over the page (the
// traffic lights; Windows 11's caption buttons with snap layouts), which the app
// keeps: they are the platform's own and work as users expect.
func chromeOptions(o *mygo.WindowOptions) {}

// buttonLayout is empty: the page draws no title buttons of its own.
func buttonLayout(*mygo.Window) string { return "" }

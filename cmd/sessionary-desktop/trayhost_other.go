//go:build !linux

package main

// macOS's menu bar and Windows's notification area are always there: a tray that was created is reachable.
func defaultTrayProbe() func() bool { return func() bool { return true } }

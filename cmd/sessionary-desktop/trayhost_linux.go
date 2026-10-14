package main

import (
	"context"
	"time"

	"github.com/godbus/dbus/v5"
)

// hostProbe returns a probe of the StatusNotifier protocol (the one AppIndicator speaks): the watcher on the
// session bus says whether a host is registered. It looks at the protocol only — which bar or desktop provides the
// host is of no interest — and a bus that cannot be reached or a watcher that is absent means "no host".
func hostProbe(connect func() (*dbus.Conn, error)) func() bool {
	var conn *dbus.Conn
	return func() bool {
		if conn == nil {
			c, err := connect()
			if err != nil {
				return false
			}
			conn = c
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		var v dbus.Variant
		err := conn.Object("org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher").
			CallWithContext(ctx, "org.freedesktop.DBus.Properties.Get", 0,
				"org.kde.StatusNotifierWatcher", "IsStatusNotifierHostRegistered").Store(&v)
		if err != nil {
			if !conn.Connected() {
				conn = nil // the bus went away: connect again next time
			}
			return false
		}
		up, _ := v.Value().(bool)
		return up
	}
}

func defaultTrayProbe() func() bool {
	return hostProbe(func() (*dbus.Conn, error) { return dbus.ConnectSessionBus() })
}

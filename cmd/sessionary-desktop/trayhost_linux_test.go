package main

import (
	"bufio"
	"os/exec"
	"strings"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
)

const watcherName = "org.kde.StatusNotifierWatcher"

// privateBus starts a session bus of its own, so no test ever touches the real one.
func privateBus(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("dbus-daemon is not installed")
	}
	cmd := exec.Command(bin, "--session", "--nofork", "--print-address")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	addr, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(addr)
}

func connect(t *testing.T, addr string) *dbus.Conn {
	t.Helper()
	c, err := dbus.Connect(addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// a fake StatusNotifierWatcher, as a bar or a panel extension provides it
func watcher(t *testing.T, addr string, hostRegistered bool) *dbus.Conn {
	t.Helper()
	c := connect(t, addr)
	_, err := prop.Export(c, "/StatusNotifierWatcher", prop.Map{
		watcherName: {"IsStatusNotifierHostRegistered": {Value: hostRegistered, Writable: false, Emit: prop.EmitFalse}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if r, err := c.RequestName(watcherName, dbus.NameFlagDoNotQueue); err != nil || r != dbus.RequestNameReplyPrimaryOwner {
		t.Fatalf("name: %v %v", r, err)
	}
	return c
}

func TestHostProbeFollowsTheWatcher(t *testing.T) {
	addr := privateBus(t)
	probe := hostProbe(func() (*dbus.Conn, error) { return dbus.Connect(addr) })

	if probe() {
		t.Fatal("no watcher on the bus: there is no host")
	}
	w := watcher(t, addr, false)
	if probe() {
		t.Fatal("a watcher with no host registered: nothing displays the icon")
	}
	_ = w.Close() // the watcher goes
	if probe() {
		t.Fatal("watcher gone: no host")
	}
	w = watcher(t, addr, true) // it comes back, with a host
	if !probe() {
		t.Fatal("watcher with a host registered: the icon is displayed")
	}
	_ = w.Close()
	if probe() {
		t.Fatal("host vanished: not reachable")
	}
	watcher(t, addr, true) // restarted
	if !probe() {
		t.Fatal("host restarted: reachable again")
	}
}

func TestHostProbeWithoutABus(t *testing.T) {
	probe := hostProbe(func() (*dbus.Conn, error) { return dbus.Connect("unix:path=/nonexistent/bus") })
	if probe() {
		t.Fatal("no session bus: no host")
	}
}

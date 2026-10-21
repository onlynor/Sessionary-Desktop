package main

import (
	"testing"

	"github.com/onlynor/sessionary-desktop/internal/backend"
)

func TestDescribeStatusMatchesLaunchTsx(t *testing.T) {
	cases := []struct {
		name     string
		s        backend.Status
		launched bool
		title    string
		detail   string
		busy     bool
	}{
		{"starting", backend.Status{Phase: backend.PhaseStarting}, false, "Sessionary", "Starting Sessionary…", true},
		{"connecting, first launch", backend.Status{Phase: backend.PhaseConnecting}, false, "Sessionary", "Connecting…", true},
		{"connecting, detail from the server", backend.Status{Phase: backend.PhaseConnecting, Detail: "Reading your agents’ sessions…"}, false, "Sessionary", "Reading your agents’ sessions…", true},
		{"connecting after a restart", backend.Status{Phase: backend.PhaseConnecting}, true, "Restarting Sessionary", "Connecting…", true},
		{"reconnecting", backend.Status{Phase: backend.PhaseReconnecting}, true, "Reconnecting", "Sessionary isn’t responding…", true},
		{"reconnecting, attached to a dev server", backend.Status{Phase: backend.PhaseReconnecting, Attached: true, Port: 4777}, true, "Reconnecting", "Waiting for the development server on port 4777…", true},
		{"restarting, first attempt", backend.Status{Phase: backend.PhaseRestarting, Attempt: 1}, true, "Restarting Sessionary", "Sessionary stopped. Starting it again…", true},
		{"restarting, later attempt", backend.Status{Phase: backend.PhaseRestarting, Attempt: 3}, true, "Restarting Sessionary", "Sessionary stopped. Starting it again (attempt 3)…", true},
		{"failed before ever launching", backend.Status{Phase: backend.PhaseFailed, Detail: "no Node.js found"}, false, "Sessionary couldn’t start", "no Node.js found", false},
		{"failed after launching once", backend.Status{Phase: backend.PhaseFailed, Detail: "crashed"}, true, "Sessionary is unavailable", "crashed", false},
		{"stopped", backend.Status{Phase: backend.PhaseStopped}, true, "Sessionary", "Quitting…", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			title, detail, busy := describeStatus(c.s, c.launched)
			if title != c.title || detail != c.detail || busy != c.busy {
				t.Errorf("describeStatus(%+v, %v) = (%q, %q, %v), want (%q, %q, %v)", c.s, c.launched, title, detail, busy, c.title, c.detail, c.busy)
			}
		})
	}
}

func TestIdleTeardownDisabledAtZero(t *testing.T) {
	old := idleTeardown
	t.Cleanup(func() { idleTeardown, idleTimer = old, nil })

	idleTeardown = 0
	armIdleTeardown()
	if idleTimer != nil {
		t.Error("idleTeardown = 0 must not arm a timer")
	}
}

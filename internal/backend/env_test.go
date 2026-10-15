//go:build !windows

package backend

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The regression: an app started from a launcher gets the session's PATH, which lacks the user's ~/.local/bin and
// the like, so the server's detection ("command -v claude") and its launching (spawn "claude") both fail — while the
// same build started from a shell works. These runs use the fake server, which detects and starts an agent by bare
// name exactly as Sessionary's does, once with the launcher's PATH alone and once with the user's shell PATH
// resolved at start (what cmd/sessionary-desktop does with shellenv).

func agentHarness(t *testing.T, resolve func() []string) *harness {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "userbin") // stands in for ~/.local/bin
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	agent := filepath.Join(bin, "claude")
	if err := os.WriteFile(agent, []byte("#!/bin/sh\necho agent-started\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, "never", func(o *Options) {
		// the session's PATH: the system directories, nothing of the user's
		o.Env = append(o.Env, "PATH=/usr/bin:/bin", "FAKE_AGENT=claude", "FAKE_USERBIN="+bin)
		if resolve != nil {
			o.ResolveEnv = func() []string { return resolve() }
		}
	})
	t.Setenv("FAKE_USERBIN", bin)
	return h
}

func (h *harness) agentState() (detected, output string) {
	d, _ := os.ReadFile(filepath.Join(h.dir, "agent-detected"))
	o, _ := os.ReadFile(filepath.Join(h.dir, "agent-output"))
	return strings.TrimSpace(string(d)), strings.TrimSpace(string(o))
}

func (h *harness) waitAgent() {
	h.t.Helper()
	h.b.Start()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		// detection is written first, then the agent runs and its output is written: wait for both
		if d, o := h.agentState(); d == "none" || (d != "" && o != "") {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	h.t.Fatal("the server never reported on the agent")
}

func TestLauncherPathAloneFindsNoAgent(t *testing.T) {
	h := agentHarness(t, nil)
	h.waitAgent()
	if d, o := h.agentState(); d != "none" || o != "" {
		t.Fatalf("with the session's PATH the agent is not found and not started; got detected=%q output=%q", d, o)
	}
}

func TestUsersShellPathFindsAndStartsTheAgent(t *testing.T) {
	h := agentHarness(t, func() []string {
		// what shellenv.ResolvePath returns: the user's directories, then the session's
		return []string{"PATH=" + os.Getenv("FAKE_USERBIN") + ":/usr/bin:/bin"}
	})
	userbin := os.Getenv("FAKE_USERBIN")
	h.waitAgent()
	d, o := h.agentState()
	if want := filepath.Join(userbin, "claude"); d != want {
		t.Fatalf("detected %q, want %q", d, want)
	}
	// detection and launching share one environment: what was detected is what ran
	if o != "agent-started" {
		t.Fatalf("the agent did not start by name: %q", o)
	}
}

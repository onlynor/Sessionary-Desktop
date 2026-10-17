package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo"

	"github.com/onlynor/sessionary-desktop/internal/backend"
)

// The smoke test runs the real app — a production build included — and checks
// what a user would notice first: the bundled server starts, the page loads and
// talks to it through the app's origin, the app recovers when the server dies,
// and quitting leaves nothing behind (scripts/smoke.sh checks that last part).
// SESSIONARY_DESKTOP_SMOKE names the JSON report; a screenshot goes beside it.
// The app exits with status 1 when a check fails.

type smokeReport struct {
	OK       bool              `json:"ok"`
	Version  string            `json:"version"`
	Error    string            `json:"error,omitempty"`
	Steps    []string          `json:"steps"`
	Backend  backend.Status    `json:"backend"`
	Chrome   string            `json:"chrome"`
	Timings  map[string]string `json:"timings"`
	Recovery string            `json:"recovery,omitempty"`
}

var smokeFailed bool

func startSmoke() {
	path := os.Getenv("SESSIONARY_DESKTOP_SMOKE")
	if path == "" {
		return
	}
	go func() {
		r := smokeReport{Timings: map[string]string{}, Version: appVersion()}
		start := time.Now()
		err := runSmoke(&r, start)
		r.OK = err == nil
		if err != nil {
			r.Error = err.Error()
			smokeFailed = true
		}
		r.Backend = server.Status()
		raw, _ := json.MarshalIndent(r, "", "  ")
		_ = os.WriteFile(path, raw, 0o644)
		fmt.Fprintf(os.Stderr, "smoke: ok=%v %s\n", r.OK, r.Error)
		mygo.App.Quit()
	}()
}

func runSmoke(r *smokeReport, start time.Time) error {
	// a release build says which version it must be (the tag): a package built from the wrong tree fails here
	if want := os.Getenv("SESSIONARY_EXPECT_VERSION"); want != "" && want != appVersion() {
		return fmt.Errorf("the app says it is version %q, the release is %q", appVersion(), want)
	}
	step := func(s string) {
		r.Steps = append(r.Steps, s)
		r.Timings[s] = time.Since(start).Round(time.Millisecond).String()
	}
	ready := func(gen int, within time.Duration) error {
		deadline := time.Now().Add(within)
		for time.Now().Before(deadline) {
			if s := server.Status(); s.Phase == backend.PhaseReady && s.Generation >= gen {
				return nil
			} else if s.Phase == backend.PhaseFailed {
				return fmt.Errorf("the server failed: %s", s.Detail)
			}
			time.Sleep(100 * time.Millisecond)
		}
		return fmt.Errorf("the server was not ready within %s (%s: %s)", within, server.Status().Phase, server.Status().Detail)
	}
	eval := func(js string) (any, error) {
		w := main_
		if w == nil {
			return nil, fmt.Errorf("no window")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return w.Page().EvalContext(ctx, js)
	}
	pageUp := func(within time.Duration) error {
		deadline := time.Now().Add(within)
		var last any
		for time.Now().Before(deadline) {
			// the client mounted, no launch or outage panel over it, and the API answers through the app's origin
			v, err := eval(`(async () => { const nav = !!document.querySelector('.dt-nav'); const over = !!document.querySelector('.dt-launch'); const r = await fetch('/api/status'); return nav && !over && r.status === 200 ? 'ok' : 'nav=' + nav + ' panel=' + over + ' api=' + r.status })()`)
			if err == nil && v == "ok" {
				return nil
			}
			last = v
			if err != nil {
				last = err.Error()
			}
			time.Sleep(250 * time.Millisecond)
		}
		return fmt.Errorf("the page did not come up within %s (%v)", within, last)
	}

	if err := ready(1, 3*time.Minute); err != nil {
		return err
	}
	s := server.Status()
	if !s.Attached && !strings.Contains(s.Node, "runtime") && os.Getenv("SESSIONARY_NODE") == "" {
		return fmt.Errorf("the server runs on %s, not the bundled runtime", s.Node)
	}
	step("server ready")
	if err := pageUp(time.Minute); err != nil {
		return err
	}
	step("page loaded")
	// the title buttons the page draws (Linux), once the window manager has said what it allows
	for i := 0; i < 30; i++ {
		c, err := eval(`[...document.querySelectorAll('.dt-wc-btn')].map(b => b.classList[1].slice(6)).join(',')`)
		if err == nil && c != "" {
			r.Chrome = fmt.Sprint(c)
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if runtime.GOOS == "linux" && r.Chrome == "" {
		return fmt.Errorf("the window has no title buttons")
	}
	shot(strings.TrimSuffix(os.Getenv("SESSIONARY_DESKTOP_SMOKE"), ".json") + ".png")

	if s.Attached {
		step("attached: recovery not tested")
		return nil
	}
	pid := server.PID()
	if pid == 0 {
		return fmt.Errorf("no server process to stop")
	}
	p, _ := os.FindProcess(pid)
	_ = p.Kill()
	step("server killed")
	if err := ready(s.Generation+1, time.Minute); err != nil {
		return fmt.Errorf("after the server was killed: %w", err)
	}
	step("server restarted")
	if err := pageUp(time.Minute); err != nil {
		return fmt.Errorf("after the restart: %w", err)
	}
	r.Recovery = fmt.Sprintf("server %d replaced by %d", pid, server.PID())
	step("page recovered")
	return nil
}

func shot(path string) {
	if main_ == nil {
		return
	}
	if png, err := main_.CapturePage(); err == nil {
		_ = os.WriteFile(path, png, 0o644)
	}
}

// Package backend runs and supervises the Sessionary server the desktop app
// talks to.
//
// The server is the unmodified Sessionary Node.js program (cli.mjs, bundled from
// ../Sessionary). A production app always owns its server: it starts it on a
// loopback port of its own, waits until it answers, watches it, and restarts it
// when it dies or stops answering — with a bounded backoff, then a failed state
// that waits for the user's Retry. A development app may instead use the server
// a developer already runs (`pnpm dev:server` on :4777), and waits for it to come
// back when it restarts, rather than starting another one beside it.
//
// One goroutine owns the lifecycle; everything else reads Status or sends Retry
// and Stop. The page never supervises anything: it shows the status it is sent.
package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// DevPort is where `pnpm dev:server` (and `sessionary`) listen by default.
const DevPort = 4777

// Phase is where the backend is in its life.
type Phase string

const (
	PhaseStarting     Phase = "starting"     // launching the server for the first time
	PhaseConnecting   Phase = "connecting"   // the server runs and does not answer yet (it indexes first)
	PhaseReady        Phase = "ready"        // it answers
	PhaseReconnecting Phase = "reconnecting" // it answered and stopped answering; still running, or (development) someone else's
	PhaseRestarting   Phase = "restarting"   // it stopped; starting it again
	PhaseFailed       Phase = "failed"       // it could not be started; waits for Retry
	PhaseStopped      Phase = "stopped"      // the app is quitting
)

// Status is what the window shows about the backend.
type Status struct {
	Phase      Phase  `json:"phase"`
	URL        string `json:"url"`        // http://127.0.0.1:<port>, or "" before one is chosen
	Port       int    `json:"port"`       // the server's port, which Model Control's gateway shares
	Attached   bool   `json:"attached"`   // a development server this app did not start
	Detail     string `json:"detail"`     // what is happening, or why it failed
	Attempt    int    `json:"attempt"`    // failed starts in a row
	Generation int    `json:"generation"` // counts servers that answered: a new one means new state (token, chats)
	Node       string `json:"node"`       // the Node.js that runs an owned server
	Server     string `json:"server"`     // the server script
	LogPath    string `json:"logPath"`    // where its output is kept
}

// Options say how to find, start and watch the server.
type Options struct {
	// Attach uses a Sessionary already answering on DevPort (development only).
	Attach  bool
	DevPort int

	// Node is the Node.js to run; when empty, FindNode looks for one.
	Node string
	// NodeCandidates are tried, in order, when Node is empty (a bundled runtime first).
	NodeCandidates []string
	// SystemNode allows a Node.js from PATH and version managers (development).
	SystemNode bool

	Scripts []string // candidate paths of the server script, tried in order
	Env     []string // extra environment for the server

	// StatePath remembers the port between launches, so that the gateway's
	// address stays the same; "" chooses a new one each time.
	StatePath string
	// LogPath receives the server's output (truncated when large); "" keeps it in memory only.
	LogPath string

	StartupTimeout time.Duration // from start until it answers; default 2 minutes (it indexes first)
	HealthInterval time.Duration // between checks once it answers; default 2s
	HealthTimeout  time.Duration // for one check; default 2.5s
	Unresponsive   int           // failed checks in a row before a running server is restarted; default 5
	MaxFailures    int           // failed starts in a row before giving up; default 5
	StopTimeout    time.Duration // for a graceful stop before the group is killed; default 5s

	OnChange    func(Status)
	LogCapacity int // lines kept in memory; default 500
}

func (o *Options) defaults() {
	if o.DevPort == 0 {
		o.DevPort = DevPort
	}
	if o.StartupTimeout == 0 {
		o.StartupTimeout = 2 * time.Minute
	}
	if o.HealthInterval == 0 {
		o.HealthInterval = 2 * time.Second
	}
	if o.HealthTimeout == 0 {
		o.HealthTimeout = 2500 * time.Millisecond
	}
	if o.Unresponsive == 0 {
		o.Unresponsive = 5
	}
	if o.MaxFailures == 0 {
		o.MaxFailures = 5
	}
	if o.StopTimeout == 0 {
		o.StopTimeout = 5 * time.Second
	}
	if o.LogCapacity == 0 {
		o.LogCapacity = 500
	}
}

// Backend is the supervised (or attached) server.
type Backend struct {
	opts Options
	logs *logBook

	mu      sync.Mutex
	status  Status
	proc    *process
	started bool

	retry chan struct{}
	stop  chan struct{}
	done  chan struct{}
	once  sync.Once
}

// New prepares a backend; Start brings it up.
func New(opts Options) *Backend {
	opts.defaults()
	return &Backend{
		opts:   opts,
		logs:   newLogBook(opts.LogCapacity, opts.LogPath),
		status: Status{Phase: PhaseStarting, Detail: "Starting Sessionary…", LogPath: opts.LogPath},
		retry:  make(chan struct{}, 1),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

// Status returns the current status.
func (b *Backend) Status() Status {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.status
}

// URL returns the server's base URL while it answers, else "".
func (b *Backend) URL() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.status.Phase == PhaseReady {
		return b.status.URL
	}
	return ""
}

// Logs returns the server's recent output.
func (b *Backend) Logs() []string { return b.logs.lines() }

// Start begins supervising in the background. It returns at once.
func (b *Backend) Start() {
	b.mu.Lock()
	if b.started {
		b.mu.Unlock()
		return
	}
	b.started = true
	b.mu.Unlock()
	go func() {
		defer close(b.done)
		if b.opts.Attach && isSessionary(context.Background(), b.opts.DevPort, b.opts.HealthTimeout) {
			b.attached(b.opts.DevPort)
			return
		}
		b.supervise()
	}()
}

// Retry starts again after a failure, at once (it does nothing otherwise).
func (b *Backend) Retry() {
	select {
	case b.retry <- struct{}{}:
	default:
	}
}

// Stop ends the server this app owns (an attached one keeps running) and
// returns once it has exited, killing its process group if it does not stop
// within StopTimeout.
func (b *Backend) Stop() {
	b.once.Do(func() { close(b.stop) })
	b.mu.Lock()
	started := b.started
	b.mu.Unlock()
	if !started {
		return
	}
	select {
	case <-b.done:
	case <-time.After(b.opts.StopTimeout + 3*time.Second):
	}
}

func (b *Backend) stopping() bool {
	select {
	case <-b.stop:
		return true
	default:
		return false
	}
}

func (b *Backend) set(f func(*Status)) {
	b.mu.Lock()
	f(&b.status)
	s := b.status
	b.mu.Unlock()
	if b.opts.OnChange != nil {
		b.opts.OnChange(s)
	}
}

// ---------- owning the server ----------

func (b *Backend) supervise() {
	node, script, err := b.locate()
	if err != nil {
		b.set(func(s *Status) { s.Phase = PhaseFailed; s.Detail = err.Error() })
		// nothing to run until the app changes: wait for Retry (which looks again) or Stop
		for {
			select {
			case <-b.stop:
				b.set(func(s *Status) { s.Phase = PhaseStopped })
				return
			case <-b.retry:
				if node, script, err = b.locate(); err == nil {
					goto run
				}
				b.set(func(s *Status) { s.Detail = err.Error() })
			}
		}
	}
run:
	b.set(func(s *Status) { s.Node, s.Server = node, script })
	// failures counts starts in a row that never answered: those give up after
	// MaxFailures. quick counts servers in a row that answered and died soon
	// after: those are restarted for as long as it takes, more slowly each time.
	failures, quick, lastPort, lastFailed := 0, 0, 0, false
	for !b.stopping() {
		port, err := b.choosePort(lastPort, lastFailed)
		if err != nil {
			failures++
			b.logs.add("sessionary-desktop: " + err.Error())
		} else {
			lastPort = port
			healthyFor, err := b.runOnce(node, script, port, failures)
			if b.stopping() {
				break
			}
			switch {
			case healthyFor == 0:
				failures++
			case healthyFor < 30*time.Second:
				failures, quick = 0, quick+1
			default:
				failures, quick = 0, 0
			}
			// the port is given up only for a port problem; any other failure keeps the gateway's address
			lastFailed = healthyFor == 0 && b.logs.contains("EADDRINUSE")
			if err != nil {
				b.logs.add("sessionary-desktop: " + err.Error())
			}
		}
		if failures > b.opts.MaxFailures {
			b.set(func(s *Status) {
				s.Phase, s.Attempt = PhaseFailed, failures
				s.Detail = b.explain()
			})
			select {
			case <-b.stop:
			case <-b.retry:
				failures, quick = 0, 0
				b.set(func(s *Status) { s.Phase, s.Attempt, s.Detail = PhaseRestarting, 0, "Starting Sessionary…" })
			}
			continue
		}
		delay := backoff(max(failures, quick))
		b.set(func(s *Status) {
			s.Phase, s.Attempt = PhaseRestarting, failures
			s.Detail = "Sessionary stopped — restarting…"
		})
		select {
		case <-b.stop:
		case <-b.retry:
		case <-time.After(delay):
		}
	}
	b.set(func(s *Status) { s.Phase, s.Detail = PhaseStopped, "Stopped" })
}

// backoff is the pause before the n-th restart in a row: 0.25s, 0.5s, 1s, … 8s.
func backoff(n int) time.Duration {
	d := 250 * time.Millisecond
	for i := 1; i < n && d < 8*time.Second; i++ {
		d *= 2
	}
	return min(d, 8*time.Second)
}

// runOnce runs the server until it exits or is stopped, and returns how long it
// answered (0 if it never did).
func (b *Backend) runOnce(node, script string, port, failures int) (time.Duration, error) {
	first := b.Status().Generation == 0 && failures == 0
	b.set(func(s *Status) {
		s.URL, s.Port, s.Attempt = baseURL(port), port, failures
		if first {
			s.Phase, s.Detail = PhaseStarting, "Starting Sessionary…"
		} else if s.Phase != PhaseRestarting {
			s.Phase, s.Detail = PhaseRestarting, "Restarting Sessionary…"
		}
	})
	b.logs.add(fmt.Sprintf("sessionary-desktop: starting %s %s on port %d", node, script, port))
	p, err := startProcess(node, script, port, b.opts.Env, b.logs)
	if err != nil {
		return 0, err
	}
	b.mu.Lock()
	b.proc = p
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		b.proc = nil
		b.mu.Unlock()
	}()
	b.set(func(s *Status) {
		s.Phase = PhaseConnecting
		if first {
			s.Detail = "Reading your agents’ sessions…"
		} else {
			s.Detail = "Reconnecting…"
		}
	})
	b.logs.setOnIndexed(func() {
		b.set(func(s *Status) {
			if s.Phase == PhaseConnecting && first {
				s.Detail = "Almost ready…"
			}
		})
	})

	// ---- until it answers ----
	deadline := time.NewTimer(b.opts.StartupTimeout)
	defer deadline.Stop()
	poll := time.NewTicker(150 * time.Millisecond)
	defer poll.Stop()
	for answered := false; !answered; {
		select {
		case <-b.stop:
			p.terminate(b.opts.StopTimeout)
			return 0, nil
		case <-p.exited:
			return 0, fmt.Errorf("the server exited before it answered (%s)", p.exitText())
		case <-deadline.C:
			p.terminate(b.opts.StopTimeout)
			return 0, fmt.Errorf("the server did not answer within %s", b.opts.StartupTimeout)
		case <-poll.C:
			answered = isSessionary(context.Background(), port, b.opts.HealthTimeout)
		}
	}
	readyAt := time.Now()
	b.rememberPort(port)
	b.set(func(s *Status) { s.Phase, s.Detail, s.Attempt = PhaseReady, "", 0; s.Generation++ })

	// ---- while it runs: is it still answering? ----
	tick := time.NewTicker(b.opts.HealthInterval)
	defer tick.Stop()
	misses := 0
	for {
		select {
		case <-b.stop:
			p.terminate(b.opts.StopTimeout)
			return time.Since(readyAt), nil
		case <-p.exited:
			return time.Since(readyAt), fmt.Errorf("the server exited (%s)", p.exitText())
		case <-tick.C:
			if isSessionary(context.Background(), port, b.opts.HealthTimeout) {
				if misses > 0 {
					b.set(func(s *Status) { s.Phase, s.Detail = PhaseReady, "" })
				}
				misses = 0
				continue
			}
			misses++
			if misses == 2 {
				b.set(func(s *Status) { s.Phase, s.Detail = PhaseReconnecting, "Sessionary isn’t responding…" })
			}
			if misses >= b.opts.Unresponsive {
				b.logs.add("sessionary-desktop: the server stopped answering; restarting it")
				p.terminate(b.opts.StopTimeout)
				return time.Since(readyAt), errors.New("the server stopped answering")
			}
		}
	}
}

// ---------- a server someone else runs (development) ----------

// attached follows a development server it did not start. When the server goes
// away (a developer restarts it, or stops it), the app waits for it to come back
// rather than starting another one beside it.
func (b *Backend) attached(port int) {
	b.set(func(s *Status) {
		*s = Status{Phase: PhaseReady, URL: baseURL(port), Port: port, Attached: true, Generation: 1, LogPath: b.opts.LogPath}
	})
	tick := time.NewTicker(b.opts.HealthInterval)
	defer tick.Stop()
	misses := 0
	for {
		select {
		case <-b.stop:
			b.set(func(s *Status) { s.Phase, s.Detail = PhaseStopped, "Stopped" })
			return
		case <-tick.C:
		}
		if isSessionary(context.Background(), port, b.opts.HealthTimeout) {
			if misses >= 2 {
				// back: a restarted server is a new one (new token, no chats)
				b.set(func(s *Status) { s.Phase, s.Detail = PhaseReady, ""; s.Generation++ })
			}
			misses = 0
			continue
		}
		if misses++; misses == 2 {
			b.set(func(s *Status) {
				s.Phase = PhaseReconnecting
				s.Detail = fmt.Sprintf("Waiting for the development server on port %d…", port)
			})
		}
	}
}

// ---------- ports ----------

type savedState struct {
	Port int `json:"port"`
}

// choosePort keeps the port of the last run, so that Model Control's gateway
// keeps its address, unless it is taken (by anything: the app never shares
// a port) or the last start on it failed.
func (b *Backend) choosePort(last int, lastFailed bool) (int, error) {
	candidate := last
	if candidate == 0 && b.opts.StatePath != "" {
		var st savedState
		if raw, err := os.ReadFile(b.opts.StatePath); err == nil && json.Unmarshal(raw, &st) == nil {
			candidate = st.Port
		}
	}
	if candidate > 0 && !lastFailed && portFree(candidate) {
		return candidate, nil
	}
	return freePort()
}

func (b *Backend) rememberPort(port int) {
	if b.opts.StatePath == "" {
		return
	}
	raw, _ := json.Marshal(savedState{Port: port})
	_ = os.MkdirAll(filepath.Dir(b.opts.StatePath), 0o755)
	tmp := b.opts.StatePath + ".tmp"
	if os.WriteFile(tmp, raw, 0o600) == nil {
		_ = os.Rename(tmp, b.opts.StatePath)
	}
}

func baseURL(port int) string { return "http://127.0.0.1:" + strconv.Itoa(port) }

func portFree(port int) bool {
	l, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	l.Close()
	return true
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// isSessionary reports whether a Sessionary server answers on port: it hands
// same-origin pages a token, which nothing else does.
func isSessionary(ctx context.Context, port int, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, baseURL(port)+"/api/token", nil)
	res, err := healthClient.Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	var body struct {
		Token string `json:"token"`
	}
	return res.StatusCode == 200 && json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&body) == nil && body.Token != ""
}

// healthClient never reuses a connection: a server that died must not look alive through one.
var healthClient = &http.Client{Transport: &http.Transport{DisableKeepAlives: true, Proxy: nil}}

// ---------- finding what to run ----------

func (b *Backend) locate() (node, script string, err error) {
	for _, s := range b.opts.Scripts {
		if s == "" {
			continue
		}
		if _, err := os.Stat(s); err == nil {
			script, _ = filepath.Abs(s)
			break
		}
	}
	if script == "" {
		return "", "", errors.New("the Sessionary server is missing from this installation")
	}
	if b.opts.Node != "" {
		return b.opts.Node, script, nil
	}
	node, err = FindNode(b.opts.NodeCandidates, b.opts.SystemNode)
	return node, script, err
}

// explain says why the server could not be started, in words a user can act on.
func (b *Backend) explain() string {
	switch {
	case b.logs.contains("database is locked"):
		return "Another Sessionary is using the same data (for example `sessionary` running in a terminal). Quit it, then try again."
	case b.logs.contains("EACCES") || b.logs.contains("permission denied"):
		return "Sessionary cannot read or write its data: " + b.logs.last()
	}
	return "Sessionary stopped and could not be started again: " + b.logs.last()
}

// PID returns the process id of the server this app runs, or 0.
func (b *Backend) PID() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.proc == nil || b.proc.cmd.Process == nil {
		return 0
	}
	return b.proc.cmd.Process.Pid
}

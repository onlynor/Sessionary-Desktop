//go:build !windows

package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// The test binary is also the fake server: started with FAKE_SESSIONARY=1 it
// acts out the next step of the plan in FAKE_DIR/plan (one step per launch, the
// last one repeating):
//
//	ok          answer until stopped
//	exit        exit at once with status 1
//	never       run without ever answering
//	crash:<d>   answer, then exit after d
//	hang:<d>    answer, then stop answering after d (and keep running)
//	stubborn    answer, and ignore SIGTERM
//	child       answer, and start a child process in the same group
func TestMain(m *testing.M) {
	if os.Getenv("FAKE_SESSIONARY") == "1" {
		fakeServer()
		return
	}
	os.Exit(m.Run())
}

func fakeServer() {
	dir := os.Getenv("FAKE_DIR")
	port := ""
	for i, a := range os.Args {
		if a == "--port" && i+1 < len(os.Args) {
			port = os.Args[i+1]
		}
	}
	n := bumpCounter(filepath.Join(dir, "launches"))
	appendLine(filepath.Join(dir, "pids"), strconv.Itoa(os.Getpid()))
	steps := strings.Split(strings.TrimSpace(readFile(filepath.Join(dir, "plan"))), ",")
	step := steps[min(n-1, len(steps)-1)]
	kind, arg, _ := strings.Cut(step, ":")
	d, _ := time.ParseDuration(arg)
	fmt.Println("fake server, launch", n, step)
	// what the real server does with its environment: detect an agent by name on PATH, and start it by name
	_ = os.WriteFile(filepath.Join(dir, "path"), []byte(os.Getenv("PATH")), 0o644)
	if agent := os.Getenv("FAKE_AGENT"); agent != "" {
		found, err := exec.LookPath(agent)
		if err != nil {
			found = "none"
		}
		_ = os.WriteFile(filepath.Join(dir, "agent-detected"), []byte(found), 0o644)
		if err == nil {
			out, _ := exec.Command(agent).CombinedOutput()
			_ = os.WriteFile(filepath.Join(dir, "agent-output"), out, 0o644)
		}
	}
	switch kind {
	case "exit":
		fmt.Println("Error: fake failure")
		os.Exit(1)
	case "locked":
		fmt.Println("file:///cli.mjs:938")
		fmt.Println("Error: database is locked")
		fmt.Println("    at IndexStore.dropSession")
		fmt.Println("Node.js v24.21.0")
		os.Exit(1)
	case "exit-inuse":
		fmt.Println("Error: listen EADDRINUSE: address already in use 127.0.0.1:" + port)
		os.Exit(1)
	case "never":
		sleepForever()
	case "stubborn":
		signalIgnore()
	case "child":
		c := exec.Command("sleep", "300")
		_ = c.Start()
		appendLine(filepath.Join(dir, "children"), strconv.Itoa(c.Process.Pid))
	}
	l, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		fmt.Println("listen:", err)
		os.Exit(1)
	}
	fmt.Println("indexed in 1ms")
	if os.Getenv("FAKE_ANNOUNCE") == "1" {
		// as DeepSeek Harness's web server says where it listens when started on port 0
		fmt.Println("dsh web: listening on 127.0.0.1 over plain HTTP; restrict this port to a trusted proxy or network")
		fmt.Printf("dsh web: http://%s/?token=SECRET-%d\n", l.Addr(), n)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"token":"t`+strconv.Itoa(n)+`"}`)
	})}
	go srv.Serve(l)
	switch kind {
	case "crash":
		time.Sleep(d)
		os.Exit(2)
	case "hang":
		time.Sleep(d)
		l.Close()
		srv.Close()
		sleepForever()
	}
	sleepForever()
}

// sleepForever blocks without relying on `select {}`: with no other goroutine running, the Go runtime may report a
// deadlock and exit (the likely cause of two macOS CI failures), which turns "a server that never answers" into "a
// server that exits".
func sleepForever() {
	for {
		time.Sleep(time.Hour)
	}
}

func signalIgnore() { signalNotifyIgnore(syscall.SIGTERM) }

// ---------- helpers ----------

func bumpCounter(path string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(readFile(path)))
	n++
	_ = os.WriteFile(path, []byte(strconv.Itoa(n)), 0o644)
	return n
}

func appendLine(path, line string) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err == nil {
		fmt.Fprintln(f, line)
		f.Close()
	}
}

func readFile(path string) string {
	b, _ := os.ReadFile(path)
	return string(b)
}

type harness struct {
	t   *testing.T
	dir string
	b   *Backend

	mu      sync.Mutex
	history []Status
}

func newHarness(t *testing.T, plan string, tune func(*Options)) *harness {
	t.Helper()
	dir := t.TempDir()
	h := &harness{t: t, dir: dir}
	h.setPlan(plan)
	script := filepath.Join(dir, "cli.mjs")
	_ = os.WriteFile(script, []byte("// fake"), 0o644)
	exe, _ := os.Executable()
	o := Options{
		Node:           exe,
		Scripts:        []string{script},
		Env:            []string{"FAKE_SESSIONARY=1", "FAKE_DIR=" + dir},
		StartupTimeout: 5 * time.Second,
		HealthInterval: 80 * time.Millisecond,
		HealthTimeout:  300 * time.Millisecond,
		Unresponsive:   3,
		MaxFailures:    3,
		StopTimeout:    time.Second,
		OnChange: func(s Status) {
			h.mu.Lock()
			h.history = append(h.history, s)
			h.mu.Unlock()
		},
	}
	if tune != nil {
		tune(&o)
	}
	h.b = New(o)
	t.Cleanup(func() { h.b.Stop() })
	return h
}

func (h *harness) setPlan(plan string) {
	_ = os.WriteFile(filepath.Join(h.dir, "plan"), []byte(plan), 0o644)
}

func (h *harness) launches() int {
	n, _ := strconv.Atoi(strings.TrimSpace(readFile(filepath.Join(h.dir, "launches"))))
	return n
}

// waitFor waits until the status satisfies ok.
func (h *harness) waitFor(what string, ok func(Status) bool) Status {
	h.t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if s := h.b.Status(); ok(s) {
			return s
		}
		time.Sleep(20 * time.Millisecond)
	}
	h.t.Fatalf("timed out waiting for %s; status %+v\nlogs:\n%s", what, h.b.Status(), strings.Join(h.b.Logs(), "\n"))
	return Status{}
}

func (h *harness) saw(p Phase) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, s := range h.history {
		if s.Phase == p {
			return true
		}
	}
	return false
}

func phase(p Phase) func(Status) bool { return func(s Status) bool { return s.Phase == p } }

func alive(pid int) bool {
	// a zombie still answers kill(0); /proc tells whether it really runs
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return syscall.Kill(pid, 0) == nil
	}
	fields := strings.Fields(string(b))
	return len(fields) > 2 && fields[2] != "Z"
}

func pids(path string) []int {
	var out []int
	for _, l := range strings.Fields(readFile(path)) {
		if n, err := strconv.Atoi(l); err == nil {
			out = append(out, n)
		}
	}
	return out
}

// ---------- tests ----------

func TestStartsAndAnswers(t *testing.T) {
	h := newHarness(t, "ok", nil)
	h.b.Start()
	s := h.waitFor("ready", phase(PhaseReady))
	if s.Attached || s.Generation != 1 || s.Port == 0 {
		t.Fatalf("unexpected status %+v", s)
	}
	if h.b.URL() != s.URL || !isSessionary(context.Background(), s.Port, time.Second) {
		t.Fatalf("server at %s does not answer", s.URL)
	}
	if !h.saw(PhaseConnecting) {
		t.Fatal("never reported connecting")
	}
}

func TestRestartsAfterCrash(t *testing.T) {
	h := newHarness(t, "crash:300ms,ok", nil)
	h.b.Start()
	h.waitFor("first server", func(s Status) bool { return s.Phase == PhaseReady && s.Generation == 1 })
	s := h.waitFor("second server", func(s Status) bool { return s.Phase == PhaseReady && s.Generation == 2 })
	if !h.saw(PhaseRestarting) {
		t.Fatal("never reported restarting")
	}
	if h.launches() != 2 {
		t.Fatalf("launches = %d, want 2", h.launches())
	}
	if !isSessionary(context.Background(), s.Port, time.Second) {
		t.Fatal("restarted server does not answer")
	}
}

func TestKilledRepeatedly(t *testing.T) {
	h := newHarness(t, "ok", nil)
	h.b.Start()
	for i := 1; i <= 4; i++ {
		h.waitFor(fmt.Sprintf("server %d", i), func(s Status) bool { return s.Phase == PhaseReady && s.Generation == i })
		ps := pids(filepath.Join(h.dir, "pids"))
		_ = syscall.Kill(ps[len(ps)-1], syscall.SIGKILL)
	}
	h.waitFor("server 5", func(s Status) bool { return s.Phase == PhaseReady && s.Generation == 5 })
}

func TestRestartsUnresponsiveServer(t *testing.T) {
	h := newHarness(t, "hang:300ms,ok", nil)
	h.b.Start()
	h.waitFor("first server", func(s Status) bool { return s.Phase == PhaseReady && s.Generation == 1 })
	h.waitFor("second server", func(s Status) bool { return s.Phase == PhaseReady && s.Generation == 2 })
	if !h.saw(PhaseReconnecting) {
		t.Fatal("never reported reconnecting")
	}
	first := pids(filepath.Join(h.dir, "pids"))[0]
	if alive(first) {
		t.Fatal("the unresponsive server was left running")
	}
}

func TestGivesUpThenRetries(t *testing.T) {
	h := newHarness(t, "exit", nil)
	h.b.Start()
	s := h.waitFor("failed", phase(PhaseFailed))
	if !strings.Contains(s.Detail, "fake failure") {
		t.Fatalf("detail %q does not explain the failure", s.Detail)
	}
	if n := h.launches(); n != 4 { // MaxFailures 3: the first start and three restarts
		t.Fatalf("launches = %d, want 4", n)
	}
	time.Sleep(300 * time.Millisecond)
	if n := h.launches(); n != 4 {
		t.Fatalf("kept restarting after giving up: %d launches", n)
	}
	h.setPlan("ok")
	h.b.Retry()
	h.waitFor("ready after retry", phase(PhaseReady))
}

func TestStartupTimeout(t *testing.T) {
	h := newHarness(t, "never", func(o *Options) { o.StartupTimeout = 400 * time.Millisecond; o.MaxFailures = 1 })
	h.b.Start()
	h.waitFor("failed", phase(PhaseFailed))
	for _, pid := range pids(filepath.Join(h.dir, "pids")) {
		if alive(pid) {
			t.Fatalf("server %d that never answered is still running", pid)
		}
	}
	if !strings.Contains(strings.Join(h.b.Logs(), "\n"), "did not answer within") {
		t.Fatal("the timeout is not in the log")
	}
}

func TestRememberedPortTakenChoosesAnother(t *testing.T) {
	held, _ := net.Listen("tcp", "127.0.0.1:0")
	defer held.Close()
	taken := held.Addr().(*net.TCPAddr).Port
	var state string
	h := newHarness(t, "ok", func(o *Options) {
		state = filepath.Join(filepath.Dir(o.Scripts[0]), "backend.json")
		_ = os.WriteFile(state, []byte(fmt.Sprintf(`{"port":%d}`, taken)), 0o644)
		o.StatePath = state
	})
	h.b.Start()
	s := h.waitFor("ready", phase(PhaseReady))
	if s.Port == taken {
		t.Fatal("started on a port that was taken")
	}
	if !strings.Contains(readFile(state), strconv.Itoa(s.Port)) {
		t.Fatal("the new port was not remembered")
	}
}

func TestRememberedPortIsReused(t *testing.T) {
	free, _ := freePort()
	h := newHarness(t, "ok", func(o *Options) {
		o.StatePath = filepath.Join(filepath.Dir(o.Scripts[0]), "backend.json")
		_ = os.WriteFile(o.StatePath, []byte(fmt.Sprintf(`{"port":%d}`, free)), 0o644)
	})
	h.b.Start()
	if s := h.waitFor("ready", phase(PhaseReady)); s.Port != free {
		t.Fatalf("port = %d, want the remembered %d", s.Port, free)
	}
}

func TestStopEndsTheGroup(t *testing.T) {
	h := newHarness(t, "child", nil)
	h.b.Start()
	h.waitFor("ready", phase(PhaseReady))
	server := pids(filepath.Join(h.dir, "pids"))[0]
	child := pids(filepath.Join(h.dir, "children"))[0]
	h.b.Stop()
	// without /proc (macOS) a killed process answers kill(0) until init has reaped it: give that a moment
	for i := 0; i < 100 && (alive(server) || alive(child)); i++ {
		time.Sleep(50 * time.Millisecond)
	}
	if sa, ca := alive(server), alive(child); sa || ca {
		t.Fatalf("left running after Stop: server %v, child %v", sa, ca)
	}
	if h.b.Status().Phase != PhaseStopped {
		t.Fatalf("phase = %s, want stopped", h.b.Status().Phase)
	}
}

func TestStopKillsAStubbornServer(t *testing.T) {
	h := newHarness(t, "stubborn", nil)
	h.b.Start()
	h.waitFor("ready", phase(PhaseReady))
	server := pids(filepath.Join(h.dir, "pids"))[0]
	start := time.Now()
	h.b.Stop()
	if alive(server) {
		t.Fatal("a server that ignores SIGTERM was left running")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("stopping took %s", time.Since(start))
	}
}

func TestAttachedServerIsAwaitedNotReplaced(t *testing.T) {
	dev := t.TempDir()
	_ = os.WriteFile(filepath.Join(dev, "plan"), []byte("ok"), 0o644)
	port, _ := freePort()
	exe, _ := os.Executable()
	startDev := func() *exec.Cmd {
		c := exec.Command(exe, "x", "--port", strconv.Itoa(port))
		c.Env = append(os.Environ(), "FAKE_SESSIONARY=1", "FAKE_DIR="+dev)
		if err := c.Start(); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(5 * time.Second)
		for !isSessionary(context.Background(), port, 200*time.Millisecond) && time.Now().Before(deadline) {
			time.Sleep(20 * time.Millisecond)
		}
		return c
	}
	devServer := startDev()
	h := newHarness(t, "ok", func(o *Options) { o.Attach, o.DevPort = true, port })
	h.b.Start()
	s := h.waitFor("attached", phase(PhaseReady))
	if !s.Attached || s.Port != port {
		t.Fatalf("did not attach: %+v", s)
	}
	_ = devServer.Process.Kill()
	_ = devServer.Wait()
	s = h.waitFor("waiting", phase(PhaseReconnecting))
	if !strings.Contains(s.Detail, strconv.Itoa(port)) {
		t.Fatalf("detail %q does not say what it waits for", s.Detail)
	}
	time.Sleep(300 * time.Millisecond)
	if h.launches() != 0 {
		t.Fatal("started a server of its own beside the development server")
	}
	devServer = startDev()
	defer devServer.Process.Kill()
	h.waitFor("back", func(s Status) bool { return s.Phase == PhaseReady && s.Generation == 2 })
}

func TestMissingRuntime(t *testing.T) {
	h := newHarness(t, "ok", func(o *Options) {
		o.Node, o.NodeCandidates, o.SystemNode = "", []string{filepath.Join(t.TempDir(), "runtime", "node")}, false
	})
	t.Setenv("SESSIONARY_NODE", "")
	h.b.Start()
	s := h.waitFor("failed", phase(PhaseFailed))
	if !strings.Contains(s.Detail, "runtime is missing") {
		t.Fatalf("detail = %q", s.Detail)
	}
}

func TestBackoffIsBounded(t *testing.T) {
	want := []time.Duration{250 * time.Millisecond, 500 * time.Millisecond, time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 8 * time.Second}
	for i, w := range want {
		if got := backoff(i + 1); got != w {
			t.Errorf("backoff(%d) = %s, want %s", i+1, got, w)
		}
	}
}

func TestPortInUseAtBindMovesOn(t *testing.T) {
	// the remembered port is free when checked, then taken before the server binds it
	free, _ := freePort()
	h := newHarness(t, "exit-inuse,ok", func(o *Options) {
		o.StatePath = filepath.Join(filepath.Dir(o.Scripts[0]), "backend.json")
		_ = os.WriteFile(o.StatePath, []byte(fmt.Sprintf(`{"port":%d}`, free)), 0o644)
	})
	h.b.Start()
	s := h.waitFor("ready", phase(PhaseReady))
	if s.Port == free {
		t.Fatal("stayed on a port the server could not bind")
	}
}

func TestOtherFailureKeepsThePort(t *testing.T) {
	free, _ := freePort()
	h := newHarness(t, "exit,ok", func(o *Options) {
		o.StatePath = filepath.Join(filepath.Dir(o.Scripts[0]), "backend.json")
		_ = os.WriteFile(o.StatePath, []byte(fmt.Sprintf(`{"port":%d}`, free)), 0o644)
	})
	h.b.Start()
	if s := h.waitFor("ready", phase(PhaseReady)); s.Port != free {
		t.Fatalf("port = %d, want the remembered %d (the failure was not about the port)", s.Port, free)
	}
}

func TestExplainsADataConflict(t *testing.T) {
	h := newHarness(t, "locked", nil)
	h.b.Start()
	s := h.waitFor("failed", phase(PhaseFailed))
	if !strings.Contains(s.Detail, "Another Sessionary is using the same data") {
		t.Fatalf("detail = %q", s.Detail)
	}
}

func TestFailureNamesTheError(t *testing.T) {
	h := newHarness(t, "exit", nil)
	h.b.Start()
	if s := h.waitFor("failed", phase(PhaseFailed)); !strings.HasSuffix(s.Detail, "Error: fake failure") {
		t.Fatalf("detail = %q", s.Detail)
	}
}

// a server that picks its own port and announces it, as `dsh web --port 0` does
var announcedProgram = Program{
	Name: "DeepSeek Harness",
	Args: func(script string, port int) []string {
		return []string{script, "web", "--no-open", "--port", strconv.Itoa(port)}
	},
	Announce: func(line string) (int, string, bool) {
		rest, ok := strings.CutPrefix(line, "dsh web: http://127.0.0.1:")
		if !ok {
			return 0, "", false
		}
		portText, _, _ := strings.Cut(rest, "/")
		port, err := strconv.Atoi(portText)
		return port, strings.TrimPrefix(line, "dsh web: "), err == nil
	},
	Redact: func(line string) string {
		if i := strings.Index(line, "token="); i >= 0 {
			return line[:i] + "token=…"
		}
		return line
	},
	TermProcess: true,
}

func TestAnnouncedServer(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "harness.log")
	h := newHarness(t, "crash:400ms,ok", func(o *Options) {
		o.Program = announcedProgram
		o.Env = append(o.Env, "FAKE_ANNOUNCE=1")
		o.LogPath = logFile
	})
	h.b.Start()
	s := h.waitFor("first server", func(s Status) bool { return s.Phase == PhaseReady && s.Generation == 1 })
	if s.Port == 0 || !strings.Contains(s.Entry, fmt.Sprintf("127.0.0.1:%d/?token=SECRET-1", s.Port)) {
		t.Fatalf("entry %q does not match port %d", s.Entry, s.Port)
	}
	first := s.Port
	s = h.waitFor("second server", func(s Status) bool { return s.Phase == PhaseReady && s.Generation == 2 })
	if !strings.HasSuffix(s.Entry, "token=SECRET-2") {
		t.Fatalf("after a restart the entry is %q, want the new server's token", s.Entry)
	}
	t.Logf("ports %d then %d", first, s.Port)
	if strings.Contains(strings.Join(h.b.Logs(), "\n"), "SECRET") || strings.Contains(readFile(logFile), "SECRET") {
		t.Fatal("a launch token was kept in the logs")
	}
	if !strings.Contains(readFile(logFile), "token=…") {
		t.Fatal("the announcement is missing from the log file")
	}
	if raw, _ := json.Marshal(s); strings.Contains(string(raw), "SECRET") {
		t.Fatal("the launch token is in the status sent to the page")
	}
}

func TestAnnouncedServerNeverAnnouncing(t *testing.T) {
	h := newHarness(t, "ok", func(o *Options) {
		o.Program = announcedProgram // no FAKE_ANNOUNCE: it serves, but never says where
		o.StartupTimeout, o.MaxFailures = 500*time.Millisecond, 1
	})
	h.b.Start()
	s := h.waitFor("failed", phase(PhaseFailed))
	if !strings.HasPrefix(s.Detail, "DeepSeek Harness stopped") {
		t.Fatalf("detail %q does not name the program", s.Detail)
	}
}

func TestStopTheServerAloneFirst(t *testing.T) {
	h := newHarness(t, "child", func(o *Options) {
		o.Program = announcedProgram
		o.Env = append(o.Env, "FAKE_ANNOUNCE=1")
	})
	h.b.Start()
	h.waitFor("ready", phase(PhaseReady))
	server := pids(filepath.Join(h.dir, "pids"))[0]
	child := pids(filepath.Join(h.dir, "children"))[0]
	h.b.Stop()
	// the child gets the group's SIGKILL once the server has exited: give the kernel a moment to deliver it
	for i := 0; i < 50 && (alive(server) || alive(child)); i++ {
		time.Sleep(20 * time.Millisecond)
	}
	if alive(server) || alive(child) {
		t.Fatalf("left running: server %v, child %v", alive(server), alive(child))
	}
}

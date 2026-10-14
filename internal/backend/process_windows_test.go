package backend

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// These run only on Windows (CI's windows runner, which has Node.js): the job object is what no other platform can
// prove. They use real Node.js processes, as the app does, and check the three promises of process_windows.go:
//   - stopping the server, or the app dying without cleanup, ends the server;
//   - the agents and terminals it started (not detached) end with it;
//   - what Sessionary launched detached for the user (a terminal, an editor) is left running.

// the structure handed to SetInformationJobObject is the Win32 one: 144 bytes on 64-bit Windows
func TestJobStructureSize(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("64-bit layout")
	}
	if n := unsafe.Sizeof(extendedLimits{}); n != 144 {
		t.Fatalf("JOBOBJECT_EXTENDED_LIMIT_INFORMATION is %d bytes, want 144", n)
	}
}

func alive(pid int) bool {
	h, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)
	var code uint32
	if syscall.GetExitCodeProcess(h, &code) != nil {
		return false
	}
	return code == 259 // STILL_ACTIVE
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for i := 0; i < 200; i++ {
		if ok() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func needNode(t *testing.T) string {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not on PATH")
	}
	return node
}

// the "server": starts an agent (a plain child) and a detached launch, reports both pids, then idles
const fakeServer = `
const { spawn } = require('child_process')
const idle = ['-e', 'setInterval(() => {}, 1000)']
const agent = spawn(process.execPath, idle, { stdio: 'ignore' })
const launched = spawn(process.execPath, idle, { stdio: 'ignore', detached: true })
launched.unref()
console.log('PIDS ' + JSON.stringify({ agent: agent.pid, launched: launched.pid }))
setInterval(() => {}, 1000)
`

type pids struct{ Agent, Launched int }

// startFakeServer starts it the way the app starts the real one, and waits for its report.
func startFakeServer(t *testing.T) (*process, pids) {
	t.Helper()
	node := needNode(t)
	logs := newLogBook(100, "", nil)
	got := make(chan pids, 1)
	var once sync.Once
	logs.setWatch(func(raw string) {
		if rest, ok := strings.CutPrefix(raw, "PIDS "); ok {
			var p struct{ Agent, Launched int }
			if json.Unmarshal([]byte(rest), &p) == nil {
				once.Do(func() { got <- pids(p) })
			}
		}
	})
	p, err := startProcess(node, "", []string{"-e", fakeServer}, t.TempDir(), nil, logs, false)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case ids := <-got:
		t.Cleanup(func() { killPid(ids.Launched) }) // the launch is meant to survive: the test ends it
		return p, ids
	case <-time.After(15 * time.Second):
		p.terminate(time.Second)
		t.Fatal("the server never reported its children")
		return nil, pids{}
	}
}

func killPid(pid int) {
	if pr, err := os.FindProcess(pid); err == nil {
		_ = pr.Kill()
	}
}

// stopping the server (a restart, a quit): it ends, so do its agents; what it launched for the user stays
func TestStopEndsTheServerAndItsAgentsButNotDetachedLaunches(t *testing.T) {
	p, ids := startFakeServer(t)
	if !alive(ids.Agent) || !alive(ids.Launched) {
		t.Fatal("the children are not running to begin with")
	}
	p.terminate(2 * time.Second)
	waitFor(t, "the server's agent to end with it", func() bool { return !alive(ids.Agent) })
	if !alive(ids.Launched) {
		t.Error("a detached launch (a terminal the user opened) must outlive the server")
	}
}

// the app dying — a crash, a kill, no cleanup at all — takes the server and its agents with it
func TestAppDeathEndsTheServerAndItsAgents(t *testing.T) {
	needNode(t)
	cmd := exec.Command(os.Args[0], "-test.run=^TestJobHelper$")
	cmd.Env = append(os.Environ(), "BACKEND_JOB_HELPER=1")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		t.Fatalf("helper: %v", err)
	}
	var h struct{ Server, Agent, Launched int }
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &h); err != nil {
		t.Fatalf("helper said %q", line)
	}
	t.Cleanup(func() { killPid(h.Launched) })
	if !alive(h.Server) || !alive(h.Agent) {
		t.Fatal("the server and its agent are not running to begin with")
	}
	_ = cmd.Process.Kill() // no deferred cleanup runs
	_ = cmd.Wait()
	waitFor(t, "the server to end with the app", func() bool { return !alive(h.Server) })
	waitFor(t, "its agent to end with the app", func() bool { return !alive(h.Agent) })
	if !alive(h.Launched) {
		t.Error("a detached launch must outlive the app")
	}
}

// TestJobHelper is the "app": it starts the server, reports it and waits to be killed.
func TestJobHelper(t *testing.T) {
	if os.Getenv("BACKEND_JOB_HELPER") != "1" {
		t.Skip("run by TestAppDeathEndsTheServerAndItsAgents")
	}
	p, ids := startFakeServer(t)
	b, _ := json.Marshal(map[string]int{"Server": p.cmd.Process.Pid, "Agent": ids.Agent, "Launched": ids.Launched})
	os.Stdout.Write(append(b, '\n'))
	time.Sleep(time.Minute)
}

// a server that cannot be put in a job is refused, not run unconfined
func TestAdoptRefusesWhatItCannotOwn(t *testing.T) {
	if _, err := adopt(&exec.Cmd{}); err == nil {
		t.Error("no process: adopt must fail")
	}
	c := exec.Command("cmd.exe", "/c", "exit 0")
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	if _, err := adopt(c); err == nil {
		t.Error("a process that has already ended cannot be adopted")
	}
}

package backend

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// process is one run of the server.
type process struct {
	cmd    *exec.Cmd
	alone  bool          // stop the server alone first (Program.TermProcess), not its group
	exited chan struct{} // closed when it has exited and its output is read
	err    error         // how it exited, once exited is closed
}

// spawner starts every server from one OS thread that never exits. On Linux the
// server is told to end when its parent dies (Pdeathsig), and "parent" is the
// thread that forked it: a thread Go might retire would end the server with it.
var spawner = sync.OnceValue(func() chan func() {
	ch := make(chan func())
	go func() {
		runtime.LockOSThread() // never unlocked: the thread lives as long as the app
		for f := range ch {
			f()
		}
	}()
	return ch
})

func startProcess(node, script string, args []string, dir string, env []string, logs *logBook, alone bool) (*process, error) {
	cmd := exec.Command(node, args...)
	cmd.Dir = dir
	if dir == "" {
		cmd.Dir = filepath.Dir(script)
	}
	cmd.Env = append(os.Environ(), env...)
	configure(cmd)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout
	errc := make(chan error, 1)
	spawner() <- func() { errc <- cmd.Start() }
	if err := <-errc; err != nil {
		return nil, fmt.Errorf("could not start %s: %w", node, err)
	}
	p := &process{cmd: cmd, exited: make(chan struct{}), alone: alone}
	release, err := adopt(cmd)
	if err != nil {
		// refuse to run a server whose lifetime the app cannot guarantee (Windows: no job object)
		_ = cmd.Process.Kill()
		go func() { copyLines(out, logs); _ = cmd.Wait() }()
		return nil, fmt.Errorf("could not take ownership of %s: %w", node, err)
	}
	go func() {
		copyLines(out, logs)
		p.err = cmd.Wait()
		release()
		close(p.exited)
	}()
	return p, nil
}

// nodeArgs runs the server on a loopback port; node:sqlite's experimental
// warning is noise in a desktop app's log.
func nodeArgs(script string, port int) []string { //nolint:unparam // the default Program.Args
	return []string{"--disable-warning=ExperimentalWarning", script, "--port", strconv.Itoa(port), "--host", "127.0.0.1", "--no-open"}
}

func copyLines(r io.Reader, logs *logBook) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		logs.add(sc.Text())
	}
}

// terminate asks the server's process group to stop (the server stops the
// agents it started, which share the group) — or the server alone, when it
// stops its own children — and kills the group if it has not exited within
// timeout.
func (p *process) terminate(timeout time.Duration) {
	select {
	case <-p.exited:
		return
	default:
	}
	if p.alone {
		_ = p.cmd.Process.Signal(syscall.SIGTERM)
	} else {
		signalGroup(p.cmd, false)
	}
	select {
	case <-p.exited:
	case <-time.After(timeout):
		signalGroup(p.cmd, true)
		<-p.exited
	}
	signalGroup(p.cmd, true) // whatever of the group outlived the server
}

func (p *process) exitText() string {
	if p.err == nil {
		return "exit status 0"
	}
	return p.err.Error()
}

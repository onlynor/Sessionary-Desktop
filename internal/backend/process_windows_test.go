package backend

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

// These run only on Windows (CI's windows runner): the job object is the one thing here that no other platform
// can prove. They start a real process tree (cmd.exe → ping) the way the app starts the server.

// the structure handed to SetInformationJobObject is the Win32 one: 144 bytes on 64-bit Windows
func TestJobStructureSize(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("64-bit layout")
	}
	if n := unsafe.Sizeof(extendedLimits{}); n != 144 {
		t.Fatalf("JOBOBJECT_EXTENDED_LIMIT_INFORMATION is %d bytes, want 144", n)
	}
}

// children returns the pids whose parent is pid.
func children(pid int) []int {
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer syscall.CloseHandle(snap)
	var e syscall.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	var out []int
	for err := syscall.Process32First(snap, &e); err == nil; err = syscall.Process32Next(snap, &e) {
		if int(e.ParentProcessID) == pid {
			out = append(out, int(e.ProcessID))
		}
	}
	return out
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
	for i := 0; i < 100; i++ {
		if ok() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func startTree(t *testing.T) (*process, int) {
	t.Helper()
	logs := newLogBook(100, "", nil)
	p, err := startProcess("cmd.exe", "", []string{"/c", "ping -n 120 127.0.0.1 >nul"}, t.TempDir(), nil, logs, false)
	if err != nil {
		t.Fatal(err)
	}
	var child int
	waitFor(t, "the server's child", func() bool {
		c := children(p.cmd.Process.Pid)
		if len(c) == 0 {
			return false
		}
		child = c[0]
		return true
	})
	return p, child
}

// stopping the server (a restart, a quit) takes the agents it started with it
func TestTerminateEndsDescendants(t *testing.T) {
	p, child := startTree(t)
	p.terminate(2 * time.Second)
	waitFor(t, "the descendant to end", func() bool { return !alive(child) })
}

// the app dying — a crash, a kill, no cleanup at all — takes them with it: this is what the job's handle is for
func TestAppDeathEndsDescendants(t *testing.T) {
	if os.Getenv("BACKEND_JOB_HELPER") == "1" {
		return
	}
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
	child, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "child=")))
	if err != nil {
		t.Fatalf("helper said %q", line)
	}
	if !alive(child) {
		t.Fatal("the descendant is not running to begin with")
	}
	_ = cmd.Process.Kill() // no deferred cleanup runs
	_ = cmd.Wait()
	waitFor(t, "the descendant to end with the app", func() bool { return !alive(child) })
}

// TestJobHelper is the "app": it starts the tree, reports the descendant and waits to be killed.
func TestJobHelper(t *testing.T) {
	if os.Getenv("BACKEND_JOB_HELPER") != "1" {
		t.Skip("run by TestAppDeathEndsDescendants")
	}
	_, child := startTree(t)
	fmt.Printf("child=%d\n", child)
	time.Sleep(time.Minute)
}

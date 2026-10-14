//go:build !windows

package backend

import (
	"os/exec"
	"syscall"
)

// configure puts the server in a process group of its own, with the agents and
// terminals it starts, so that the group can be stopped as one.
func configure(cmd *exec.Cmd) { cmd.SysProcAttr = sysProcAttr() }

func signalGroup(cmd *exec.Cmd, kill bool) {
	if cmd.Process == nil {
		return
	}
	sig := syscall.SIGTERM
	if kill {
		sig = syscall.SIGKILL
	}
	_ = syscall.Kill(-cmd.Process.Pid, sig)
}

// adopt: the process group made by configure is the unit; nothing more to do.
func adopt(*exec.Cmd) (release func(), err error) { return func() {}, nil }

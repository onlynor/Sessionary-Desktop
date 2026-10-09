package backend

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

func configure(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow | syscall.CREATE_NEW_PROCESS_GROUP}
}

// Windows has no process groups to signal; a job object would carry the agents along (not yet).
func signalGroup(cmd *exec.Cmd, kill bool) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

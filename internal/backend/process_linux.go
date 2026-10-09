package backend

import "syscall"

// The kernel ends the server if the app dies without the chance to stop it (SIGKILL, a crash).
func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGTERM}
}

//go:build !windows

package shellenv

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// Timeout bounds how long the login shell may take: a start-up file that waits for a network or a prompt must not
// keep the app from starting its server.
const Timeout = 8 * time.Second

// ResolvePath returns current's directories plus those of the user's login shell. Whatever goes wrong — no shell, a
// start-up file that fails or hangs, no answer — it returns current unchanged: the app then behaves as it would
// have without this, never worse.
func ResolvePath(current string) string {
	return resolve(os.Getenv("SHELL"), current, Timeout)
}

func resolve(shell, current string, timeout time.Duration) string {
	if shell == "" || !filepath.IsAbs(shell) {
		shell = "/bin/sh"
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// -i as well as -l: bash and zsh keep most PATH additions (nvm, pnpm, ~/.local/bin) in the interactive file
	cmd := exec.CommandContext(ctx, shell, "-ilc", script)
	var out bytes.Buffer
	cmd.Stdout = &out
	// a session of its own: an interactive shell with no terminal of its own must not touch the app's (job control
	// would stop it when the app was started from one), and the whole group can be ended on a timeout
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second
	if err := cmd.Run(); err != nil && out.Len() == 0 {
		return current
	}
	login := parse(out.String())
	if login == "" {
		return current
	}
	return Merge(login, current)
}

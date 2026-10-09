//go:build !windows

package backend

import (
	"os"
	"os/signal"
)

func signalNotifyIgnore(sig os.Signal) { signal.Ignore(sig) }

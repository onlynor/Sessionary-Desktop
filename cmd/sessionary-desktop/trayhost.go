package main

import (
	"context"
	"sync"
	"time"
)

// A tray icon the user cannot see is not a way back to the window. Creating one only proves that the toolkit's
// library loaded; on Linux it is the StatusNotifier *host* (a bar, a panel extension) that draws it, and a session
// may have none, lose it, or restart it. trayWatch follows that, so that the window is hidden into the tray only
// while the tray can be reached, and is never left hidden when it cannot.
type trayWatch struct {
	probe    func() bool // whether something is displaying tray icons right now
	onChange func(up bool)

	mu sync.Mutex
	up bool
}

// check asks the host and reports the change, if any, to onChange (outside the lock).
func (w *trayWatch) check() {
	up := w.probe()
	w.mu.Lock()
	changed := up != w.up
	w.up = up
	w.mu.Unlock()
	if changed && w.onChange != nil {
		w.onChange(up)
	}
}

// Up reports the host's last known state.
func (w *trayWatch) Up() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.up
}

// run polls until ctx ends. Polling, not signals: a host that restarts or vanishes without saying so is caught the
// same way as one that does, with one code path.
func (w *trayWatch) run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.check()
		}
	}
}

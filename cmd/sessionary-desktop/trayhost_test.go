package main

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

// the watch follows a host that is absent, appears, vanishes and comes back, and reports each change once
func TestTrayWatchFollowsTheHost(t *testing.T) {
	var up atomic.Bool
	var seen []bool
	w := &trayWatch{probe: up.Load, onChange: func(b bool) { seen = append(seen, b) }}

	w.check() // no host at launch: down, and not a "change"
	if w.Up() || len(seen) != 0 {
		t.Fatalf("no host: up=%v seen=%v", w.Up(), seen)
	}
	up.Store(true)
	w.check()
	w.check() // unchanged: silent
	up.Store(false)
	w.check()
	up.Store(true)
	w.check()
	if want := []bool{true, false, true}; !reflect.DeepEqual(seen, want) || !w.Up() {
		t.Fatalf("changes = %v, want %v (up=%v)", seen, want, w.Up())
	}
}

func TestTrayWatchRunStopsWithItsContext(t *testing.T) {
	var n atomic.Int32
	w := &trayWatch{probe: func() bool { n.Add(1); return true }}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { w.run(ctx, 5*time.Millisecond); close(done) }()
	time.Sleep(60 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("run did not stop")
	}
	if n.Load() == 0 || !w.Up() {
		t.Errorf("never polled: %d", n.Load())
	}
}

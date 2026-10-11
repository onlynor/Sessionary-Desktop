package backend

import (
	"os"
	"strings"
	"sync"
	"time"
)

// logBook keeps the server's recent output in memory, for the failure view,
// and in a file, for diagnostics.
type logBook struct {
	mu        sync.Mutex
	capacity  int
	buf       []string
	file      *os.File
	onIndexed func()
	redact    func(string) string // removes secrets before a line is kept
	watch     func(raw string)    // sees every line as printed (Program.Announce)
}

const maxLogFile = 2 << 20

func newLogBook(capacity int, path string, redact func(string) string) *logBook {
	l := &logBook{capacity: capacity, redact: redact}
	if path != "" {
		if fi, err := os.Stat(path); err == nil && fi.Size() > maxLogFile {
			_ = os.Rename(path, path+".1")
		}
		if f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			l.file = f
		}
	}
	return l
}

func (l *logBook) add(raw string) {
	l.mu.Lock()
	watch, line := l.watch, raw
	if l.redact != nil {
		line = l.redact(raw)
	}
	l.buf = append(l.buf, line)
	if len(l.buf) > l.capacity {
		l.buf = l.buf[len(l.buf)-l.capacity:]
	}
	if l.file != nil {
		_, _ = l.file.WriteString(time.Now().Format("2006-01-02 15:04:05 ") + line + "\n")
	}
	indexed := l.onIndexed
	l.mu.Unlock()
	if watch != nil {
		watch(raw)
	}
	if indexed != nil && strings.HasPrefix(line, "indexed in") {
		indexed()
	}
}

func (l *logBook) setWatch(f func(raw string)) {
	l.mu.Lock()
	l.watch = f
	l.mu.Unlock()
}

func (l *logBook) setOnIndexed(f func()) {
	l.mu.Lock()
	l.onIndexed = f
	l.mu.Unlock()
}

func (l *logBook) lines() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.buf...)
}

// last explains the latest run's failure in a line: the last error the server
// printed, else its last line.
func (l *logBook) last() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := len(l.buf) - 1; i >= 0; i-- {
		s := strings.TrimSpace(l.buf[i])
		if strings.HasPrefix(s, "sessionary-desktop: starting") {
			break
		}
		if strings.HasPrefix(s, "Error") || strings.Contains(s, "Error: ") {
			return s
		}
	}
	for i := len(l.buf) - 1; i >= 0; i-- {
		if s := strings.TrimSpace(l.buf[i]); s != "" && !strings.HasPrefix(s, "sessionary-desktop:") {
			return s
		}
	}
	for i := len(l.buf) - 1; i >= 0; i-- {
		if s := strings.TrimSpace(l.buf[i]); s != "" {
			return strings.TrimPrefix(s, "sessionary-desktop: ")
		}
	}
	return "no output"
}

// contains reports whether the latest run's output mentions s.
func (l *logBook) contains(s string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := len(l.buf) - 1; i >= 0; i-- {
		if strings.Contains(l.buf[i], s) {
			return true
		}
		if strings.HasPrefix(l.buf[i], "sessionary-desktop: starting") {
			return false
		}
	}
	return false
}

//go:build !windows

package shellenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMerge(t *testing.T) {
	got := Merge("/home/u/.local/bin:/usr/bin:", "/usr/bin:/opt/x::/bin")
	if want := "/home/u/.local/bin:/usr/bin:/opt/x:/bin"; got != want {
		t.Errorf("Merge = %q, want %q", got, want)
	}
	if got := Merge("", "/usr/bin"); got != "/usr/bin" {
		t.Errorf("no login PATH keeps the current one: %q", got)
	}
}

func TestParseIgnoresWhatStartupFilesPrint(t *testing.T) {
	out := "Welcome back!\nnvm: using node\n" + marker + "/a/bin:/usr/bin" + marker + "trailing\n"
	if got := parse(out); got != "/a/bin:/usr/bin" {
		t.Errorf("parse = %q", got)
	}
	for _, bad := range []string{"", "no markers", marker + "unterminated"} {
		if got := parse(bad); got != "" {
			t.Errorf("parse(%q) = %q, want none", bad, got)
		}
	}
}

// fakeShell writes an executable script and returns its path; it is run as `shell -ilc <script>` like a real one.
func fakeShell(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fakesh")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolveAsksTheLoginShell(t *testing.T) {
	// a shell whose start-up file adds ~/.local/bin and prints a banner, and which is run with -i -l -c
	sh := fakeShell(t, `echo "banner"; [ "$1" = "-ilc" ] || exit 9; PATH="/home/u/.local/bin:$PATH"; eval "$2"`)
	got := resolve(sh, "/usr/bin:/bin", 5*time.Second)
	if !strings.HasPrefix(got, "/home/u/.local/bin:") || !strings.Contains(got, "/usr/bin") {
		t.Errorf("PATH = %q", got)
	}
}

func TestResolveKeepsTheCurrentPathWhenTheShellFails(t *testing.T) {
	for name, body := range map[string]string{
		"exits non-zero": `exit 3`,
		"prints nothing": `:`,
		"no markers":     `echo hello`,
	} {
		if got := resolve(fakeShell(t, body), "/usr/bin:/bin", 5*time.Second); got != "/usr/bin:/bin" {
			t.Errorf("%s: PATH = %q, want the current one", name, got)
		}
	}
	if got := resolve("/nonexistent/shell", "/usr/bin", 5*time.Second); got != "/usr/bin" {
		t.Errorf("missing shell: %q", got)
	}
}

// a start-up file that waits (for a network, a prompt) must not hold the app: the shell is ended at the timeout,
// together with whatever it started
func TestResolveGivesUpOnAShellThatHangs(t *testing.T) {
	sh := fakeShell(t, `sleep 30 & wait`)
	start := time.Now()
	got := resolve(sh, "/usr/bin", 300*time.Millisecond)
	if got != "/usr/bin" {
		t.Errorf("PATH = %q", got)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("took %s", d)
	}
}

// without SHELL (a minimal session) the system shell is used, and the call still answers
func TestResolveWithoutAShellVariable(t *testing.T) {
	got := resolve("", "/usr/bin", 5*time.Second)
	if got == "" {
		t.Error("empty PATH")
	}
}

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The page keeps its settings in localStorage; the desktop mirrors them to a file so a new loopback port does not
// forget them. These are the read/write half of that.
func TestPrefsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prefs.json")
	if got := readPrefs(path); len(got) != 0 {
		t.Fatalf("a missing file reads empty, got %v", got)
	}

	in := map[string]string{"sessionary:theme": `"ember"`, "sessionary:lang": `"zh-Hans"`, "sessionary:navCollapsed": "true"}
	if err := writePrefs(path, in); err != nil {
		t.Fatal(err)
	}
	if got := readPrefs(path); !reflect.DeepEqual(got, in) {
		t.Fatalf("round trip = %v, want %v", got, in)
	}

	// the file is replaced whole, not appended to
	if err := writePrefs(path, map[string]string{"sessionary:theme": `"paper"`}); err != nil {
		t.Fatal(err)
	}
	if got := readPrefs(path); !reflect.DeepEqual(got, map[string]string{"sessionary:theme": `"paper"`}) {
		t.Fatalf("after replace = %v", got)
	}

	// the temp file is moved into place, never left behind
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("a .tmp file was left behind")
	}

	// an unreadable file is not fatal: the client starts with its defaults
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readPrefs(path); len(got) != 0 {
		t.Fatalf("garbage reads empty, got %v", got)
	}
	if err := writePrefs(path, nil); err != nil {
		t.Fatalf("writing no settings still succeeds: %v", err)
	}
	if got := readPrefs(path); len(got) != 0 {
		t.Fatalf("an empty write reads empty, got %v", got)
	}
}

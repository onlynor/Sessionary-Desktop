package backend

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MinNode is the oldest Node.js with node:sqlite, which Sessionary needs.
var MinNode = [3]int{22, 13, 0}

// FindNode returns the first of candidates that is a Node.js new enough to run
// Sessionary — the runtime bundled with the app comes first — and, with system,
// then looks on PATH and in the usual version managers' places (development: an
// app started from a launcher gets a short PATH). SESSIONARY_NODE wins over all.
func FindNode(candidates []string, system bool) (string, error) {
	var list []string
	if p := os.Getenv("SESSIONARY_NODE"); p != "" {
		list = append(list, p)
	}
	list = append(list, candidates...)
	if system {
		if p, err := exec.LookPath("node"); err == nil {
			list = append(list, p)
		}
		home, _ := os.UserHomeDir()
		for _, g := range []string{
			"/usr/local/bin/node", "/usr/bin/node", "/opt/homebrew/bin/node",
			filepath.Join(home, ".volta/bin/node"),
			filepath.Join(home, ".local/share/mise/shims/node"),
			filepath.Join(home, ".local/share/fnm/aliases/default/bin/node"),
			filepath.Join(home, ".nvm/versions/node/*/bin/node"),
			filepath.Join(home, ".asdf/shims/node"),
		} {
			m, _ := filepath.Glob(g)
			list = append(list, m...)
		}
	}
	seen := map[string]bool{}
	var tooOld []string
	for _, c := range list {
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		v, ok := nodeVersion(c)
		if !ok {
			continue
		}
		if less(v, MinNode) {
			tooOld = append(tooOld, fmt.Sprintf("%s (%d.%d.%d)", c, v[0], v[1], v[2]))
			continue
		}
		return c, nil
	}
	if len(tooOld) > 0 {
		return "", fmt.Errorf("Sessionary needs Node.js %d.%d or newer; found only %s", MinNode[0], MinNode[1], strings.Join(tooOld, ", "))
	}
	if !system {
		return "", errors.New("the Node.js runtime is missing from this installation; reinstall Sessionary")
	}
	return "", fmt.Errorf("Sessionary needs Node.js %d.%d or newer, and none was found", MinNode[0], MinNode[1])
}

func nodeVersion(path string) ([3]int, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return [3]int{}, false
	}
	parts := strings.SplitN(strings.TrimPrefix(strings.TrimSpace(string(out)), "v"), ".", 3)
	var v [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		v[i], _ = strconv.Atoi(parts[i])
	}
	return v, v[0] > 0
}

func less(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// Package shellenv gives the app the environment a user's terminal would have had.
//
// An application started from a launcher, a desktop entry or the Dock does not run in a terminal: its PATH is the
// session's (systemd's user manager, the display manager's), which lacks everything the user's shell start-up files
// add — ~/.local/bin, npm and pnpm globals, version managers, ~/.opencode/bin. The agents' command line programs live
// there, so a server started with that PATH reports every agent "not found on PATH" and cannot start one, while the
// same build started from a shell works. The fix is the one Sessionary's own server uses for a node it reaches over
// ssh: ask the user's login shell what its PATH is.
package shellenv

import (
	"path/filepath"
	"strings"
)

const marker = "__SESSIONARY_PATH__"

// script is what the shell is asked to run: the markers set the answer apart from whatever a start-up file prints.
const script = `printf '` + marker + `%s` + marker + `' "$PATH"`

// parse takes the PATH out of a shell's output, or "" when the markers are not there.
func parse(out string) string {
	i := strings.Index(out, marker)
	if i < 0 {
		return ""
	}
	rest := out[i+len(marker):]
	j := strings.Index(rest, marker)
	if j < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:j])
}

// Merge returns login's directories followed by those of current that login lacks, so that nothing the process was
// given is lost and what the user's shell would find first is found first. Empty entries are dropped.
func Merge(login, current string) string {
	sep := string(filepath.ListSeparator)
	seen := map[string]bool{}
	var out []string
	for _, list := range []string{login, current} {
		for _, d := range strings.Split(list, sep) {
			if d == "" || seen[d] {
				continue
			}
			seen[d] = true
			out = append(out, d)
		}
	}
	return strings.Join(out, sep)
}

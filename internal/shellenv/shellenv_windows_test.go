package shellenv

import (
	"strings"
	"testing"

	"golang.org/x/sys/windows/registry"
)

// The registry keys this reads always exist on a real Windows machine (Session Manager\Environment holds at least
// SystemRoot's PATH entries), so this is a smoke test against the real thing rather than a fake one: it confirms
// ResolvePath actually reads the registry instead of just echoing current back, which was the bug (the old
// implementation returned its argument unchanged, so a just-installed agent's directory was invisible until reboot).
func TestResolvePathReadsTheRegistry(t *testing.T) {
	machine := registryPath(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`)
	if machine == "" {
		t.Skip("no Machine Environment key on this runner")
	}
	if strings.Contains(machine, "%") {
		t.Errorf("Machine PATH was not expanded: %q", machine)
	}
	got := ResolvePath("C:\\only\\the\\current\\one")
	if !strings.Contains(got, "C:\\only\\the\\current\\one") {
		t.Errorf("ResolvePath dropped the current PATH: %q", got)
	}
	for _, dir := range strings.Split(machine, ";") {
		if dir == "" {
			continue
		}
		if !strings.Contains(got, dir) {
			t.Errorf("ResolvePath = %q, missing Machine entry %q", got, dir)
			break
		}
	}
}

func TestRegistryPathOnAMissingKey(t *testing.T) {
	if got := registryPath(registry.CURRENT_USER, `Software\Sessionary\DoesNotExist`); got != "" {
		t.Errorf("registryPath = %q, want empty for a missing key", got)
	}
}

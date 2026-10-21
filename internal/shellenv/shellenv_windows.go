package shellenv

import (
	"golang.org/x/sys/windows/registry"
)

// ResolvePath returns current's directories plus the Machine and User Path the registry holds right now.
//
// A process started from a shortcut, the Start menu or a pinned taskbar icon does not read the registry itself: it
// inherits explorer.exe's environment block, built once when the desktop session started. Installing something
// afterward — Node, nvm, pnpm's global bin, an agent's own installer — updates HKLM's or HKCU's Environment key and
// broadcasts WM_SETTINGCHANGE, which a running terminal's shell picks up on its next prompt, but explorer.exe (and
// so this app, launched from it) never re-reads its own environment block without being restarted. The effect is
// the same as the Unix case this package also handles: the app sees the PATH its session started with, not the PATH
// a newly opened terminal would show, so a just-installed agent is "not found on PATH" until the next reboot or
// sign-out. Reading both keys directly, as a fresh terminal does, avoids waiting for that.
func ResolvePath(current string) string {
	machine := registryPath(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`)
	user := registryPath(registry.CURRENT_USER, `Environment`)
	// Windows itself builds a process's PATH as the Machine value followed by the User value.
	return Merge(Merge(machine, user), current)
}

// registryPath reads and expands key\Path ("" when it is absent or unreadable); REG_EXPAND_SZ entries such as
// "%SystemRoot%\system32" are expanded, REG_SZ entries are returned as they are.
func registryPath(root registry.Key, path string) string {
	k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	v, _, err := k.GetStringValue("Path")
	if err != nil {
		return ""
	}
	if expanded, err := registry.ExpandString(v); err == nil {
		return expanded
	}
	return v
}

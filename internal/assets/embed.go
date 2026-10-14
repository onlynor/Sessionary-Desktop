// Package assets embeds the app's icons: the window's and the tray's. (Not under resources/, which `mygo build` ships.)
package assets

import _ "embed"

// Icon is the application icon (window, taskbar); Tray and TrayTemplate are the tray's (template: macOS).
var (
	//go:embed icon.png
	Icon []byte
	//go:embed tray.png
	Tray []byte
	//go:embed trayTemplate.png
	TrayTemplate []byte
)

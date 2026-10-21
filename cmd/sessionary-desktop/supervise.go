package main

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/egoist/mygo"

	"github.com/onlynor/sessionary-desktop/internal/backend"
	"github.com/onlynor/sessionary-desktop/internal/shellenv"
)

// backendOptions say where the server and its runtime are, and how the app
// treats them.
//
// A production build owns its server: the bundled Node.js runs the bundled
// server on a port of the app's own (remembered between launches, so the
// gateway keeps its address), and nothing else is ever used.
//
// A development build may use the server a developer runs (`pnpm dev:server`
// on :4777, SESSIONARY_DESKTOP_ATTACH=0 to never do so), and may run its own on
// any Node.js it finds when the runtime was not fetched (scripts/fetch-node.mjs).
func backendOptions() backend.Options {
	dev := mygo.IsDev()
	o := backend.Options{
		Attach:         dev && os.Getenv("SESSIONARY_DESKTOP_ATTACH") != "0",
		NodeCandidates: runtimeCandidates(),
		SystemNode:     dev || os.Getenv("SESSIONARY_SYSTEM_NODE") == "1",
		Scripts:        serverScripts(),
		OnChange: func(s backend.Status) {
			BackendChanged.Broadcast(s)
			ensureMain() // opens the real window once this reaches PhaseReady; a native wait otherwise (launch.go)
		},
		// the server and every agent it starts inherit the user's shell PATH, not the launcher's (see shellenv)
		ResolveEnv: func() []string { return []string{"PATH=" + shellenv.ResolvePath(os.Getenv("PATH"))} },
	}
	if dir, err := mygo.App.Path(mygo.PathUserData); err == nil {
		o.StatePath = filepath.Join(dir, "backend.json")
	}
	if dir, err := mygo.App.Path(mygo.PathLogs); err == nil {
		o.LogPath = filepath.Join(dir, "backend.log")
	}
	return o
}

// The installed layout (mygo.json's resources keep their base names):
//
//	<resources>/runtime/bin/node    Node.js, version pinned in scripts/fetch-node.mjs
//	<resources>/server/cli.mjs      the Sessionary server, bundled from ../Sessionary
//
// A development checkout has the same two under .stage/ once built.
func resourceDirs() []string {
	var dirs []string
	if p := os.Getenv("SESSIONARY_RESOURCES"); p != "" {
		dirs = append(dirs, p)
	}
	if res, err := mygo.App.Path(mygo.PathResources); err == nil {
		dirs = append(dirs, res)
	}
	if exe, err := os.Executable(); err == nil {
		if exe, err = filepath.EvalSymlinks(exe); err == nil {
			dirs = append(dirs, filepath.Dir(exe))
		}
	}
	if mygo.IsDev() {
		if wd, err := os.Getwd(); err == nil {
			dirs = append(dirs, filepath.Join(wd, ".stage"))
		}
	}
	return dirs
}

func runtimeCandidates() []string {
	name := "node"
	if runtime.GOOS == "windows" {
		name = "node.exe"
	}
	var out []string
	for _, d := range resourceDirs() {
		out = append(out, filepath.Join(d, "runtime", "bin", name))
	}
	return out
}

func serverScripts() []string {
	out := []string{os.Getenv("SESSIONARY_SERVER")}
	for _, d := range resourceDirs() {
		out = append(out, filepath.Join(d, "server", "cli.mjs"))
	}
	return out
}

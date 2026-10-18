# Development

Building, running, and testing Sessionary Desktop from source. For installing a released build, see the
[README](../README.md).

## Requirements

- Go 1.27+
- [Bun](https://bun.sh)
- Node.js (for the build tools)
- WebKitGTK 4.1 (Linux, to build and run)
- `pnpm install` done in a sibling checkout of [Sessionary](https://github.com/onlynor/Sessionary) (`../Sessionary`), or set `SESSIONARY_DIR` to point at one

To run an already-built app: nothing beyond GTK 3 and WebKitGTK on Linux. macOS and Windows need nothing extra.

## Commands

```sh
bun install --cwd frontend
bun run dev            # mygo dev: Vite with hot reload through the app's own origin, Go rebuilds on change
bun run build          # frontend + server + pinned Node.js runtime + mygo build → build/<os>-<arch>/ (.deb, tarball, …)
bun run typecheck      # go vet (all platforms) + tsc over the desktop layer and the client it pulls in
```

Scripted runs, for screenshots and checks (development builds only):

```sh
scripts/dev/run.sh                   # build, start, wait until the server answers
scripts/dev/drive.py go '#/models'   # navigate · eval · theme · size · shot · tray · status
scripts/dev/drive.py tour .shots/x ember  # every main surface, one PNG each
```

Environment: `SESSIONARY_DESKTOP_ATTACH=0` always starts an own server · `SESSIONARY_SERVER` a server script ·
`SESSIONARY_NODE` a Node.js · `SESSIONARY_HOME` Sessionary's data dir (handy for a clean first run).

## Tests

```sh
go test ./...                                         # lifecycle (fake servers act out crashes, hangs, …), proxy, layout, dsh
bun run --cwd frontend test                           # agents' order; overrides in step with the web client
scripts/test/smoke.sh build/linux-amd64/sessionary    # the built app, as a first-time user
scripts/test/smoke-deb.sh build/linux-amd64/*.deb     # the .deb on a clean Ubuntu in Docker
```

`scripts/dev/drive.py tray` reports whether the tray is installed, `window hide|show|quit` exercises the lifecycle, and
`go test ./...` covers the icon, the menu and the close-versus-quit rule (`cmd/sessionary-desktop/tray_test.go`).

## Continuous integration

`.github/workflows/ci.yml`, after [Magpie](https://github.com/yetone/magpie)'s approach: on every push and pull request,

- **go** (Linux, macOS, Windows): gofmt, vet for all three platforms, tests (the lifecycle suite drives real process
  groups on Unix), race detector on Linux;
- **package** (linux/amd64, linux/arm64, darwin/arm64, windows/amd64): checks out Sessionary beside the app
  (`SESSIONARY_TOKEN` secret if private; `sessionary_ref` input), generates the bindings, type-checks the frontend, runs its tests, `go tool mygo build`;
  on macOS and Windows, `scripts/test/smoke-native.sh` runs the packaged executable once;
  on Linux, `scripts/test/smoke.sh` runs the binary under Xvfb and `scripts/test/smoke-deb.sh` installs the `.deb` on a clean
  Ubuntu without Node.js and runs the same smoke test. Packages and smoke reports are uploaded.

The smoke test (`SESSIONARY_DESKTOP_SMOKE=report.json`, any build) checks: the bundled server answers, the page
loads and reaches it through the app's origin, title buttons are drawn, a killed server is replaced and the page
recovers; then `smoke.sh` checks that quitting left no process of the installation running.

See also [Architecture](ARCHITECTURE.md), [Desktop runtime](DESKTOP_RUNTIME.md), and [Releases](RELEASES.md).

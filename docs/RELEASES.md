# Releases and versions

How Sessionary Desktop is versioned, published, and what it bundles. For installation instructions, see the
[README](../README.md).

## Version

[Semantic Versioning](https://semver.org), written once: `"version"` in `mygo.json` (no `v`), which `mygo build`
puts into the packages and the app (About, the smoke report). The git tag is `v` + that. While the major is 0,
anything may change between minors; PATCH is fixes, MINOR is features or a new bundled Sessionary / Node.js, and
1.0.0 is the promise that upgrades will not break settings. This version is the *app's*, not Sessionary's: the server
it bundles has its own, and each release names the Sessionary commit it was built from.

## Publishing

`.github/workflows/release.yml`; a tag does everything, as a draft:

```sh
# 1. a pull request "release: v0.1.0" sets "version" in mygo.json; merge it
git switch main && git pull
git tag v0.1.0 && git push origin v0.1.0
# 2. Actions → Release: the CI jobs run on the tagged commit with the version enforced, then the draft appears
# 3. Releases → the draft: check the notes and the smoke summaries, then Publish
```

The workflow refuses a tag that is not `v` + `mygo.json`'s version or is not on `main`, runs the same build, tests and
smoke tests as CI (each packaged app must report that version), and drafts the release with the `.deb` and tarball
(+ `install.sh`) for Linux amd64/arm64, the `.dmg` for macOS (Apple silicon), the installer for Windows (amd64) and
`SHA256SUMS`. The notes are the Conventional Commit subjects since the previous tag, grouped (the first release says
so instead). A release also needs the macOS and Windows app launches to pass, which CI alone tolerates. A suffix
(`v0.2.0-rc.1`) makes a pre-release. **Rehearsal:** Actions → Release → Run workflow on any branch runs everything
and prints the notes without creating a release.

Packages are not signed or notarized yet, and there is no auto-update (MyGo's updater needs a signing key). The
Sessionary build can be pinned by setting the repository variable `SESSIONARY_RELEASE_REF` to a tag or commit there
(default `main`).

## Bundled runtime

Production builds carry their own Node.js, so nothing needs to be installed on the machine:

| | |
|---|---|
| Version | Node.js **24.21.0** (LTS), pinned with each platform's SHA-256 in `scripts/build/fetch-node.mjs` |
| Fetched | at **build** time (`node scripts/build/fetch-node.mjs`, part of `bun run build`), verified, cached in `~/.cache/sessionary-desktop/node`; never at first launch |
| Staged | `.stage/runtime/{bin/node,LICENSE}` (stripped on Linux: 108 MB), `.stage/server/cli.mjs` (the server, one self-contained file) |
| Installed | next to the executable: `/opt/sessionary/{sessionary,runtime/bin/node,server/cli.mjs}` from the `.deb` (with `/usr/bin/sessionary-desktop`), the same three in the tarball |
| Found | `runtimeCandidates()` / `serverScripts()` in `cmd/sessionary-desktop/supervise.go`: `SESSIONARY_NODE` / `SESSIONARY_SERVER`, then the resources dir, then the executable's dir (symlinks resolved) |
| Development | the same `.stage/` once built; failing that, a dev build may use any Node.js ≥ 22.13 it finds (PATH, nvm, volta, fnm, mise, asdf). A production build never uses a system Node.js (`SESSIONARY_SYSTEM_NODE=1` to allow it) |

Package sizes (linux/amd64): `.deb` 46 MB, tarball 46 MB (the runtime is 108 MB of the 121 MB installed).

The bundled Node.js runtime's own LICENSE is staged and shipped alongside it (`.stage/runtime/LICENSE`); it is not
part of this project's source and keeps its own upstream license. This project's own `LICENSE` and
[THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md) (for the Go libraries statically linked into the binary) are
listed in `mygo.json`'s `resources` and so ship next to the executable in every package too — verified by inspecting
the built `.deb` and tarball.

See also [Development](DEVELOPMENT.md) for how a build is produced locally.

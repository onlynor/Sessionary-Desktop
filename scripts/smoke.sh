#!/usr/bin/env bash
# smoke.sh APP [OUT_DIR] — runs the built app as a first-time user would, and checks it:
#   the bundled server starts; the page loads and reaches it through the app's origin; the app recovers when the
#   server is killed; quitting leaves no process of the installation behind.
# A fresh HOME stands in for a new user (no agents' history, no settings). Without a display it runs under xvfb-run.
set -euo pipefail
app=$(realpath "$1")
out=$(realpath -m "${2:-.smoke}")
install_dir=$(dirname "$(readlink -f "$app")")
rm -rf "$out"; mkdir -p "$out/home"
export HOME="$out/home" XDG_CONFIG_HOME="$out/home/.config" XDG_DATA_HOME="$out/home/.local/share" XDG_CACHE_HOME="$out/home/.cache"
export SESSIONARY_DESKTOP_SMOKE="$out/report.json" SESSIONARY_DESKTOP_ATTACH=0
unset SESSIONARY_NODE SESSIONARY_SERVER SESSIONARY_RESOURCES SESSIONARY_DESKTOP_DRIVE

run=("$app")
if [ -z "${WAYLAND_DISPLAY:-}" ] && [ -z "${DISPLAY:-}" ]; then
  command -v xvfb-run >/dev/null || { echo "smoke: no display and no xvfb-run" >&2; exit 2; }
  run=(xvfb-run -a -s "-screen 0 1600x1000x24" dbus-run-session -- "$app")
fi

status=0
timeout 300 "${run[@]}" >"$out/app.log" 2>&1 || status=$?
cat "$out/report.json" 2>/dev/null || echo "smoke: no report"
echo "app exit status: $status"

# nothing of the installation may outlive the app (the server, its runtime, anything they started)
sleep 1
left=$(ps -eo pid=,args= | awk -v d="$install_dir/" 'index($0, d) && !/awk/ && !/smoke.sh/' || true)
if [ -n "$left" ]; then echo "smoke: left running after quit:"; echo "$left"; status=1; fi

if [ "$status" -ne 0 ] || ! grep -q '"ok": true' "$out/report.json" 2>/dev/null; then
  echo "smoke: FAILED — log: $out/app.log, server log: $XDG_CONFIG_HOME/Sessionary/logs/backend.log" >&2
  tail -20 "$out/app.log" >&2 || true
  exit 1
fi
echo "smoke: passed"

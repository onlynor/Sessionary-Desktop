#!/usr/bin/env bash
# smoke-native.sh APP [OUT_DIR] — the smoke test for macOS and Windows, whose runners have a desktop session: runs
# the packaged executable once with a fresh home and the smoke report switched on, and fails when it does.
# (Linux has its own, with a virtual display: smoke.sh.) The app checks itself — server answers, page loads and
# reaches it, a killed server is replaced — and exits non-zero otherwise.
set -euo pipefail
app=$1
out=${2:-${RUNNER_TEMP:-/tmp}/smoke}
rm -rf "$out"; mkdir -p "$out/home"
export HOME="$out/home" USERPROFILE="$out/home" APPDATA="$out/home/AppData/Roaming" LOCALAPPDATA="$out/home/AppData/Local"
export SESSIONARY_DESKTOP_SMOKE="$out/report.json" SESSIONARY_DESKTOP_ATTACH=0
unset SESSIONARY_NODE SESSIONARY_SERVER SESSIONARY_RESOURCES SESSIONARY_DESKTOP_DRIVE
status=0
perl -e 'alarm 300; exec @ARGV' -- "$app" >"$out/app.log" 2>&1 || status=$?
cat "$out/report.json" 2>/dev/null || echo "smoke: no report"
echo "app exit status: $status"
[ "$status" -eq 0 ] || { tail -40 "$out/app.log"; exit 1; }

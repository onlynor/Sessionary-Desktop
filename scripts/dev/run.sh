#!/usr/bin/env bash
# Builds and starts a development app that serves the built frontend (frontend/dist) and listens on the drive
# socket, for screenshots and checks. For live editing use `bun run dev` instead.
set -euo pipefail
cd "$(dirname "$0")/.."
export SESSIONARY_DESKTOP_DRIVE=${SESSIONARY_DESKTOP_DRIVE:-/tmp/sessionary-desktop-drive.sock}
mkdir -p .run
go build -o .run/sessionary-desktop .
pkill -f '.run/sessionary-desktop' 2>/dev/null || true
sleep 0.3
nohup .run/sessionary-desktop > .run/app.log 2>&1 &
echo "pid $!"
python3 scripts/drive.py wait-ready 90
# niri tiles windows: float this one at a fixed size so screenshots compare
if command -v niri >/dev/null && [ -n "${NIRI_SOCKET:-}" ]; then
  id=$(niri msg -j windows | python3 -c "import json,sys,os;w=[x for x in json.load(sys.stdin) if x['app_id'] in ('sessionary','sessionary-desktop','Sessionary')];print(w[-1]['id'] if w else '')")
  if [ -n "$id" ]; then
    niri msg action move-window-to-floating --id "$id" >/dev/null
    niri msg action set-window-width --id "$id" "${W:-1440}" >/dev/null
    niri msg action set-window-height --id "$id" "${H:-900}" >/dev/null
    niri msg action center-window --id "$id" >/dev/null 2>&1 || true
    echo "niri window $id"
  fi
fi

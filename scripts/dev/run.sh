#!/usr/bin/env bash
# Builds and starts a development app that serves the built frontend (frontend/dist) and listens on the drive
# socket, for screenshots and checks. For live editing use `bun run dev` instead.
set -euo pipefail
cd "$(dirname "$0")/../.."
export SESSIONARY_DESKTOP_DRIVE=${SESSIONARY_DESKTOP_DRIVE:-/tmp/sessionary-desktop-drive.sock}
mkdir -p .run
if [ -f .run/app.pid ]; then kill "$(cat .run/app.pid)" 2>/dev/null || true; sleep 0.3; fi
go build -o .run/sessionary-desktop ./cmd/sessionary-desktop
nohup .run/sessionary-desktop > .run/app.log 2>&1 &
echo $! > .run/app.pid
echo "pid $!"
python3 scripts/dev/drive.py wait-ready 90

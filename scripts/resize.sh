#!/usr/bin/env bash
# resize.sh W H — the app's window, in logical pixels (niri)
id=$(niri msg -j windows | python3 -c "import json,sys;print([x for x in json.load(sys.stdin) if x['app_id']=='sessionary-desktop'][-1]['id'])")
niri msg action set-window-width --id "$id" "$1" >/dev/null
niri msg action set-window-height --id "$id" "$2" >/dev/null
sleep 0.8

#!/usr/bin/env bash
# shotwin.sh APP_ID OUT.png — the compositor's screenshot of the app's window (niri), floated at 1440x900
id=$(niri msg -j windows | python3 -c "import json,sys;w=[x for x in json.load(sys.stdin) if x['app_id']=='$1'];print(w[-1]['id'] if w else '')")
[ -z "$id" ] && { echo "no window $1" >&2; exit 1; }
if [ "${FLOAT:-1}" = 1 ]; then niri msg action move-window-to-floating --id "$id" >/dev/null 2>&1; niri msg action set-window-width --id "$id" 1440 >/dev/null; niri msg action set-window-height --id "$id" 900 >/dev/null; fi
niri msg action screenshot-window --id "$id" -d true --path "$(realpath -m "$2")" >/dev/null
sleep 0.4

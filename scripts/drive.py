#!/usr/bin/env python3
"""Drive a development build of Sessionary Desktop through its drive socket.

    drive.py eval 'location.hash'            # evaluate JavaScript in the page
    drive.py go '#/m/local/projects'         # navigate (hash route) and settle
    drive.py theme dark|light|system         # the app's appearance
    drive.py webtheme paper|ember|graphite   # Sessionary's own theme setting
    drive.py size 1440 900                   # resize the window
    drive.py shot out.png                    # capture the page
    drive.py window hide|show|close|quit|state
                                         # window actions; close hides to the tray
    drive.py tray                            # whether the tray is installed, and whether the window shows
    drive.py status                          # the backend's status
"""
import json, os, socket, sys, time

SOCK = os.environ.get('SESSIONARY_DESKTOP_DRIVE', '/tmp/sessionary-desktop-drive.sock')

def call(req):
    s = socket.socket(socket.AF_UNIX)
    s.connect(SOCK)
    s.sendall((json.dumps(req) + '\n').encode())
    buf = b''
    while not buf.endswith(b'\n'):
        chunk = s.recv(65536)
        if not chunk:
            break
        buf += chunk
    s.close()
    res = json.loads(buf)
    if 'error' in res:
        raise SystemExit('drive: ' + res['error'])
    return res.get('value')

def main(argv):
    cmd, args = argv[0], argv[1:]
    if cmd == 'eval':
        print(json.dumps(call({'eval': args[0]}), indent=1))
    elif cmd == 'go':
        call({'eval': f'location.hash = {json.dumps(args[0])}; true'})
        time.sleep(float(args[1]) if len(args) > 1 else 1.2)
    elif cmd == 'theme':
        call({'theme': args[0]})
    elif cmd == 'webtheme':
        call({'eval': f"localStorage.setItem('sessionary:theme', JSON.stringify({json.dumps(args[0])})); location.reload(); true"})
        time.sleep(2)
    elif cmd == 'size':
        call({'size': [int(args[0]), int(args[1])]})
        time.sleep(0.6)
    elif cmd == 'shot':
        # the window is usually not focused while a script runs: show it as the active window
        call({'eval': "document.documentElement.removeAttribute('data-inactive'); true"})
        time.sleep(0.15)
        call({'shot': os.path.abspath(args[0])})
        print(os.path.abspath(args[0]))
    elif cmd == 'tour':
        # tour <prefix> [theme]: the main surfaces, each as <prefix>-<name>.png
        prefix = args[0]
        if len(args) > 1:
            call({'eval': f"localStorage.setItem('sessionary:theme', JSON.stringify({json.dumps(args[1])})); location.hash='#/'; setTimeout(()=>location.reload(), 300); true"})
            time.sleep(3)
        sid = call({'eval': "fetch('/api/sessions').then(r=>r.json()).then(s=>(s.find(x=>x.messageCount>20)||s[0]).id)"})
        routes = [('home', '#/'), ('projects', '#/m/local/projects'), ('sessions', '#/m/local/sessions'), ('session', '#/m/local/s/' + sid),
                  ('agents', '#/m/local/agents'), ('models', '#/models'), ('routing', '#/routing'), ('usage', '#/usage'), ('machines', '#/machines')]
        for name, h in routes:
            main(['go', h, '1.6'])
            main(['shot', f'{prefix}-{name}.png'])
    elif cmd == 'window':
        print(json.dumps(call({'window': args[0] if args else 'state'}), indent=1))
    elif cmd == 'tray':
        print(json.dumps(call({'tray': True}), indent=1))
    elif cmd == 'status':
        print(json.dumps(call({'status': True}), indent=1))
    elif cmd == 'wait-ready':
        deadline = time.time() + float(args[0] if args else 60)
        while time.time() < deadline:
            try:
                if call({'status': True})['phase'] == 'ready':
                    return
            except (OSError, SystemExit, json.JSONDecodeError):
                pass
            time.sleep(0.3)
        raise SystemExit('drive: not ready')
    else:
        raise SystemExit(__doc__)

if __name__ == '__main__':
    main(sys.argv[1:] or ['help'])

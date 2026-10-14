import { isMyGo, runtime } from 'mygo-runtime'
import { Desktop, events, type Info, type Status } from '../mygo'

/** Whether the page runs in the desktop app, not in a browser tab of the dev server. */
export const inApp = isMyGo()

export const platform: 'darwin' | 'linux' | 'win32' | 'web' = inApp ? (runtime().platform as 'darwin' | 'linux' | 'win32') : 'web'

let info: Info | undefined
export const appInfo = async () => (info ??= inApp ? await Desktop.info() : { platform: 'web', version: 'dev', dev: true, blur: true, tray: false })

/** Marks the document for the desktop layer's styles, before anything paints. */
export function markDocument() {
  const root = document.documentElement
  root.classList.add('desktop')
  root.dataset.platform = platform
  if (/mac/i.test(navigator.platform) && platform === 'web') root.dataset.platform = 'darwin-web'
}

// ---------- settings that outlive the page's origin ----------
const PREFIX = 'sessionary:'

/**
 * Restores the page's settings from the app before the client reads them, then mirrors every change back.
 * The page's origin is a loopback port that changes between launches, so localStorage alone would forget them.
 */
export async function restorePrefs() {
  if (!inApp) return
  try {
    const saved = await Desktop.prefs()
    for (const [k, v] of Object.entries(saved)) localStorage.setItem(k, v)
  } catch { /* first launch, or the file was unreadable: start fresh */ }

  const pending = new Map<string, string | null>()
  let timer: number | undefined
  const flush = () => {
    timer = undefined
    if (!pending.size) return
    const changes = Object.fromEntries(pending)
    pending.clear()
    Desktop.savePrefs(changes).catch(() => {})
  }
  const queue = (k: string, v: string | null) => {
    if (!k.startsWith(PREFIX)) return
    pending.set(k, v)
    timer ??= window.setTimeout(flush, 250)
  }
  const proto = Storage.prototype
  const set = proto.setItem, remove = proto.removeItem
  proto.setItem = function (this: Storage, k: string, v: string) { set.call(this, k, v); if (this === localStorage) queue(k, String(v)) }
  proto.removeItem = function (this: Storage, k: string) { remove.call(this, k); if (this === localStorage) queue(k, null) }
  addEventListener('pagehide', flush)
}

/** Desktop apps follow the system's appearance unless told otherwise. */
export function preferSystemTheme() {
  if (localStorage.getItem(PREFIX + 'theme') == null) localStorage.setItem(PREFIX + 'theme', JSON.stringify('system'))
}

// ---------- the window follows the page's theme ----------
/** The window paints its own background while resizing and before the page draws: keep it the page's. */
export function syncWindowBackground() {
  if (!inApp) return
  let last = ''
  const apply = () => {
    const color = getComputedStyle(document.documentElement).getPropertyValue('--canvas').trim() || getComputedStyle(document.body).backgroundColor
    if (color && color !== last) { last = color; Desktop.setBackground(color).catch(() => {}) }
  }
  new MutationObserver(apply).observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })
  apply()
}

// ---------- the backend ----------
export async function backendStatus(): Promise<Status> {
  if (!inApp) return { phase: 'ready', url: '', port: 0, attached: true, detail: '', attempt: 0, generation: 1, node: '', server: '', logPath: '' }
  return Desktop.backend()
}
export const onBackend = (fn: (s: Status) => void) => (inApp ? events.backend.on(fn) : () => {})

/** Quit the app for good — the tray, the macOS app menu and Ctrl Q all lead here. */
export const quitApp = () => {
  if (inApp) Desktop.quit()
}

// ---------- what the app's menus and the system ask of the page ----------
/**
 * The client's own Settings entry, wherever it draws it: the foot of the sidebar (web/src/Nav.tsx). The app's menu
 * bar and Ctrl+, open it here so that both platforms reach the same menu, not a second settings system.
 */
export const openSettings = () => document.querySelector<HTMLButtonElement>('.nav-foot button:last-of-type')?.click()

const ROUTES: Record<string, string> = {
  home: '#/',
  projects: '#/m/local/projects',
  sessions: '#/m/local/sessions',
  models: '#/models',
  routing: '#/routing',
  gateway: '#/gateway',
  usage: '#/usage',
}
export function listenForCommands() {
  if (!inApp) return
  events.command.on((cmd) => {
    if (cmd === 'palette') dispatchEvent(new CustomEvent('sessionary:palette'))
    else if (cmd.startsWith('go:')) location.hash = ROUTES[cmd.slice(3)] ?? cmd.slice(3)
    else if (cmd === 'sidebar') dispatchEvent(new KeyboardEvent('keydown', { key: '[' }))
    else if (cmd === 'new-session') dispatchEvent(new CustomEvent('sessionary:new-session', { detail: {} }))
    else if (cmd === 'settings') openSettings()
    else dispatchEvent(new CustomEvent('sessionary-desktop:' + cmd))
  })
}

// ---------- notifications through the system, not the web view ----------
/**
 * The client notifies with the web's Notification API. In the app it goes through the desktop's own notification
 * service instead: it is the one the user configured, and a click on it brings this window forward.
 */
export function routeNotifications() {
  if (!inApp) return
  let seq = 0
  class DesktopNotification {
    static permission: NotificationPermission = 'granted'
    static requestPermission = async (): Promise<NotificationPermission> => 'granted'
    onclick: ((this: DesktopNotification, ev: Event) => unknown) | null = null
    onclose: ((this: DesktopNotification, ev: Event) => unknown) | null = null
    onerror: ((this: DesktopNotification, ev: Event) => unknown) | null = null
    onshow: ((this: DesktopNotification, ev: Event) => unknown) | null = null
    readonly title: string
    readonly body: string
    readonly tag: string
    constructor(title: string, opts: NotificationOptions = {}) {
      this.title = title
      this.body = opts.body ?? ''
      this.tag = opts.tag || `n${++seq}`
      Desktop.notify(this.tag, title, this.body).catch(() => {})
    }
    close() {}
    addEventListener() {}
    removeEventListener() {}
  }
  Object.defineProperty(window, 'Notification', { value: DesktopNotification, configurable: true, writable: true })
}

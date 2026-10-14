import { api } from '@web/api'
import { t } from '@web/i18n'
import { href } from '@web/route'
import { Desktop } from '../mygo'
import { inApp, onPref } from './runtime'
import { recentIds, TRAY_LABELS } from './trayLabels'

/** how many recent sessions the tray offers */
const RECENT = 3

/**
 * Keeps the app's tray menu in the user's language and offers the last sessions. The words are the client's own
 * catalog (`t`); the sessions are what the client itself lists (`api.sessions`), opened by the client's own routes.
 * The tray implements no navigation: it sends these routes and commands back through the app's command channel.
 */
export function syncTray() {
  if (!inApp) return
  let timer: number | undefined
  const push = async () => {
    timer = undefined
    const labels = Object.fromEntries(Object.entries(TRAY_LABELS).map(([k, en]) => [k, t(en)]))
    let recents: { id: string; title: string; route: string }[] = []
    try {
      const ids = recentIds(localStorage.getItem('sessionary:recent'))
      if (ids.length) {
        const by = new Map((await api.sessions()).map((s) => [s.id, s]))
        recents = ids.flatMap((id) => {
          const s = by.get(id)
          return s ? [{ id, title: s.title, route: href.session('local', id) }] : []
        }).slice(0, RECENT)
      }
    } catch { /* the server is not there yet: the words alone */ }
    Desktop.setTray(labels, recents).catch(() => {})
  }
  const later = () => { timer ??= window.setTimeout(push, 400) }
  new MutationObserver(later).observe(document.documentElement, { attributes: true, attributeFilter: ['lang'] })
  onPref((k) => { if (k === 'sessionary:recent' || k === 'sessionary:lang') later() })
  addEventListener('blur', later) // the user is about to use the tray: its sessions are current
  later()
}

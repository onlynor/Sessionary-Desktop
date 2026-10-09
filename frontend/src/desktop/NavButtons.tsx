import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useEffect, useState } from 'react'
import { t, useT } from '@web/i18n'

/**
 * Where the window is in its own history. Pages navigate by hash; each new entry is numbered in history.state, so
 * the toolbar knows whether there is anywhere to go back or forward to — which the History API does not tell.
 */
const nav = { index: 0, max: 0 }
const subs = new Set<() => void>()
function track() {
  const s = history.state as { dtIndex?: number } | null
  if (typeof s?.dtIndex === 'number') {
    nav.index = s.dtIndex
  } else {
    nav.index = nav.index + 1
    nav.max = nav.index // a new entry drops everything that was forward of it
    history.replaceState({ ...(s ?? {}), dtIndex: nav.index }, '')
  }
  nav.max = Math.max(nav.max, nav.index)
  subs.forEach((f) => f())
}
export function trackHistory() {
  history.replaceState({ ...(history.state ?? {}), dtIndex: 0 }, '')
  addEventListener('popstate', track)
  addEventListener('hashchange', () => { if (typeof (history.state as { dtIndex?: number } | null)?.dtIndex !== 'number') track() })
}

/** Back and forward, at the leading end of the window's toolbar, as in Finder and System Settings. */
export function NavButtons() {
  useT()
  const [, tick] = useState(0)
  useEffect(() => { const f = () => tick((n) => n + 1); subs.add(f); return () => { subs.delete(f) } }, [])
  const canBack = nav.index > 0
  const canForward = nav.index < nav.max
  return (
    <div className="dt-navbtns" role="group" aria-label={t('Navigation')}>
      <button className="dt-navbtn" disabled={!canBack} onClick={() => history.back()} title={`${t('Back')}  Alt ←`} aria-label={t('Back')}><ChevronLeft size={18} strokeWidth={1.6} absoluteStrokeWidth /></button>
      <button className="dt-navbtn" disabled={!canForward} onClick={() => history.forward()} title={`${t('Forward')}  Alt →`} aria-label={t('Forward')}><ChevronRight size={18} strokeWidth={1.6} absoluteStrokeWidth /></button>
    </div>
  )
}

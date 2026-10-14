import { currentWindow } from 'mygo-runtime'
import { useEffect, useLayoutEffect, useState } from 'react'
import { t, useT } from '@web/i18n'
import { Desktop, events, type WindowState } from '../mygo'
import { inApp, platform } from './runtime'

type Button = 'minimize' | 'maximize' | 'close'

/** the window's state, as the system reports it (focus, maximized, full screen) */
export function useWindowState() {
  const [state, setState] = useState<WindowState>({ maximized: false, fullScreen: false, focused: true })
  useEffect(() => {
    if (!inApp) return
    Desktop.windowState().then(setState).catch(() => {})
    return events.window.on(setState)
  }, [])
  useEffect(() => {
    const root = document.documentElement
    root.dataset.window = state.fullScreen ? 'fullscreen' : state.maximized ? 'maximized' : 'normal'
    // the client dims selection and symbols in an inactive window; the system knows best when that is
    if (inApp) root.toggleAttribute('data-inactive', !state.focused)
  }, [state])
  return state
}


/**
 * The title buttons, where the page draws them: on Linux the window is frameless (MyGo announces client-side
 * decorations, so no compositor adds a title bar of its own); macOS and Windows keep the system's. One fixed set,
 * no query of the desktop, the compositor or the display server; each calls the window's own method. They leave
 * room for themselves through --dt-ctl-end.
 */
export function WindowControls({ state }: { state: WindowState }) {
  useT()
  // Minimize hides the window in the tray, which is the one way back that every compositor honours; with no tray
  // (or none displayed) there is nothing reliable to do, so the button is not drawn rather than doing nothing
  const [tray, setTray] = useState(false)
  useEffect(() => {
    if (!inApp || platform !== 'linux') return
    Desktop.info().then((i) => setTray(i.tray)).catch(() => {})
    return events.tray.on(setTray) // the tray's host can appear, vanish and come back
  }, [])
  const buttons: Button[] = tray ? ['minimize', 'maximize', 'close'] : ['maximize', 'close']
  const shown = inApp && platform === 'linux' && !state.fullScreen
  useLayoutEffect(() => {
    document.documentElement.style.setProperty('--dt-ctl-end', shown ? `${buttons.length * 34 + 14}px` : '0px')
  }, [shown, buttons.length])

  if (!shown) return null
  return (
    <div className="dt-wc dt-wc-end" role="group" aria-label={t('Window')}>
      {buttons.map((b) => <ControlButton key={b} kind={b} maximized={state.maximized} />)}
    </div>
  )
}

function ControlButton({ kind, maximized }: { kind: Button; maximized: boolean }) {
  const label = kind === 'close' ? t('Close') : kind === 'minimize' ? t('Minimize') : maximized ? t('Restore') : t('Maximize')
  const act = () => {
    if (kind === 'close') currentWindow.close()
    else if (kind === 'minimize') Desktop.minimize()
    else currentWindow.toggleMaximize()
  }
  return (
    <button className={`dt-wc-btn dt-wc-${kind}`} onClick={act} title={label} aria-label={label}
      // a press here must not start a window drag or a double-click maximize
      onMouseDown={(e) => e.stopPropagation()} onDoubleClick={(e) => e.stopPropagation()}>
      <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
        {kind === 'close' && <path d="M2.5 2.5l7 7M9.5 2.5l-7 7" />}
        {kind === 'minimize' && <path d="M2.5 6h7" />}
        {kind === 'maximize' && !maximized && <rect x="2.5" y="2.5" width="7" height="7" rx="1.2" />}
        {kind === 'maximize' && maximized && <><rect x="2.5" y="4" width="5.5" height="5.5" rx="1" /><path d="M4.5 2.5h4a1 1 0 0 1 1 1v4" /></>}
      </svg>
    </button>
  )
}

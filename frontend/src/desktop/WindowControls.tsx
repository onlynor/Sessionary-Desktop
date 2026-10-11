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

const parse = (layout: string): [Button[], Button[]] => {
  const side = (s = '') => s.split(',').filter((b): b is Button => b === 'minimize' || b === 'maximize' || b === 'close')
  const [start, end] = layout.split(':')
  return [side(start), side(end)]
}

/**
 * The title buttons, where the page draws them (Linux: the window is frameless, so that no compositor adds a
 * title bar of its own). Which buttons and on which side follow the desktop's layout (gtk-decoration-layout); each
 * calls the window's own method. They leave room for themselves through --dt-ctl-start / --dt-ctl-end.
 */
export function WindowControls({ state }: { state: WindowState }) {
  useT()
  const [layout, setLayout] = useState<[Button[], Button[]]>([[], []])
  useEffect(() => {
    if (!inApp || platform !== 'linux') return
    Desktop.chrome().then((c) => setLayout(parse(c.buttons))).catch(() => {})
  }, [])

  const [start, end] = state.fullScreen ? [[], []] as [Button[], Button[]] : layout
  useLayoutEffect(() => {
    const root = document.documentElement.style
    root.setProperty('--dt-ctl-start', start.length ? `${start.length * 34 + 14}px` : '0px')
    root.setProperty('--dt-ctl-end', end.length ? `${end.length * 34 + 14}px` : '0px')
  }, [start.length, end.length])

  if (!start.length && !end.length) return null
  const group = (buttons: Button[], side: 'start' | 'end') => buttons.length > 0 && (
    <div className={`dt-wc dt-wc-${side}`} role="group" aria-label={t('Window')}>
      {buttons.map((b) => <ControlButton key={b} kind={b} maximized={state.maximized} />)}
    </div>
  )
  return <>{group(start, 'start')}{group(end, 'end')}</>
}

function ControlButton({ kind, maximized }: { kind: Button; maximized: boolean }) {
  const label = kind === 'close' ? t('Close') : kind === 'minimize' ? t('Minimize') : maximized ? t('Restore') : t('Maximize')
  const act = () => {
    if (kind === 'close') currentWindow.close()
    else if (kind === 'minimize') currentWindow.minimize()
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

import '@fontsource-variable/inter/opsz.css'
import '@fontsource-variable/jetbrains-mono/wght.css'
import '@web/styles.css'
import './desktop/desktop.css'
import { useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { installBehaviors } from './desktop/behaviors'
import { registerStrings } from './desktop/i18n'
import { BackendState } from './desktop/Launch'
import { WindowControls, useWindowState } from './desktop/WindowControls'
import { NavButtons, trackHistory } from './desktop/NavButtons'
import { appInfo, backendStatus, inApp, listenForCommands, markDocument, onBackend, preferSystemTheme, restorePrefs, routeNotifications, syncWindowBackground } from './desktop/runtime'
import type { Status } from './mygo'
import { LANGS, getLang, setLang, type Lang } from '@web/i18n'

/**
 * Sessionary Desktop's page: Sessionary's web client, unchanged, inside the desktop layer. The layer restores the
 * settings, marks the document for its styles, shows the launch screen until the server answers, and connects the
 * page to the app (menus, notifications, the window's background).
 */
async function boot() {
  markDocument()
  registerStrings()
  await restorePrefs()
  preferSystemTheme()
  applyThemeEarly()
  const info = await appInfo()
  if (!info.blur) document.documentElement.dataset.noBlur = ''
  installBehaviors({ dev: info.dev })
  routeNotifications()
  listenForCommands()
  trackHistory()
  // the client reads its settings when its modules load: only now, after they are restored
  const { App } = await import('@web/App')
  createRoot(document.getElementById('root')!).render(<Root App={App} />)
  syncWindowBackground()
}

/** the same as index.html's first script, again now that the saved settings are back */
function applyThemeEarly() {
  try {
    let t = JSON.parse(localStorage.getItem('sessionary:theme') || '"system"')
    if (t === 'system') t = matchMedia('(prefers-color-scheme: light)').matches ? 'paper' : 'ember'
    document.documentElement.dataset.theme = t
    // the client's i18n read the language when this module loaded, before the saved settings were back
    const l = JSON.parse(localStorage.getItem('sessionary:lang') || 'null') as Lang | null
    if (l && LANGS.some((x) => x.id === l) && l !== getLang()) setLang(l)
    if (l) document.documentElement.lang = l
  } catch { /* defaults */ }
}

/**
 * The window: the client once Sessionary has answered, the backend's state over it (or instead of it, before
 * then), and the title buttons above everything, so that the window can be closed, moved and resized whatever
 * the server is doing.
 */
function Root({ App }: { App: () => React.JSX.Element }) {
  const [status, setStatus] = useState<Status>()
  const win = useWindowState()
  // the server the client was mounted against: a new one (after a restart) has a new token, no chats and a
  // fresh index, so the client starts over with it — on the same page, which lives in the URL
  const [mounted, setMounted] = useState(0)
  useEffect(() => {
    const take = (s: Status) => {
      setStatus(s)
      if (s.phase === 'ready') setMounted((m) => (s.generation !== m ? s.generation : m))
    }
    const off = onBackend(take)
    backendStatus().then(take)
    return off
  }, [])
  const ready = status?.phase === 'ready'
  return (
    <>
      {mounted > 0 && <div className="dt-app" inert={!ready || undefined}><App key={mounted} /><NavButtons /></div>}
      {status && !ready && <BackendState status={status} launched={mounted > 0} />}
      <WindowControls state={win} />
    </>
  )
}

boot()

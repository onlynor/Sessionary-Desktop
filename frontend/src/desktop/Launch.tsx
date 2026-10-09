import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { t, useT } from '@web/i18n'
import type { Status } from '../mygo'
import { Desktop } from '../mygo'

/** what to say about a status: a title, and the line under it */
function describe(s: Status, launched: boolean): { title: string; detail: string; busy: boolean } {
  switch (s.phase) {
    case 'starting':
      return { title: 'Sessionary', detail: t('Starting Sessionary…'), busy: true }
    case 'connecting':
      return { title: launched ? t('Restarting Sessionary') : 'Sessionary', detail: t(s.detail || 'Connecting…'), busy: true }
    case 'reconnecting':
      return {
        title: t('Reconnecting'),
        detail: s.attached ? t('Waiting for the development server on port {port}…', { port: s.port }) : t(s.detail || 'Sessionary isn’t responding…'),
        busy: true,
      }
    case 'restarting':
      return {
        title: t('Restarting Sessionary'),
        detail: s.attempt > 1 ? t('Sessionary stopped. Starting it again (attempt {n})…', { n: s.attempt }) : t('Sessionary stopped. Starting it again…'),
        busy: true,
      }
    case 'failed':
      return { title: launched ? t('Sessionary is unavailable') : t('Sessionary couldn’t start'), detail: t(s.detail), busy: false }
    case 'stopped':
      return { title: 'Sessionary', detail: t('Quitting…'), busy: true }
    default:
      return { title: 'Sessionary', detail: '', busy: false }
  }
}

/**
 * The backend's state while the client cannot work: before Sessionary first answers, the launch screen; after,
 * a panel over the (inert) client, which stays mounted underneath. Either way the window keeps working: its
 * buttons, moving it, and the actions here.
 */
export function BackendState({ status, launched }: { status: Status; launched: boolean }) {
  useT()
  // a moment's hiccup shows nothing; a server that answers at once never shows the launch screen
  const [shown, setShown] = useState(false)
  useEffect(() => { const id = setTimeout(() => setShown(true), launched ? 700 : 350); return () => clearTimeout(id) }, [launched])
  const [logs, setLogs] = useState<string[]>()
  const [retrying, setRetrying] = useState(false)
  const failed = status.phase === 'failed'
  const { title, detail, busy } = describe(status, launched)
  useEffect(() => { if (!failed) setLogs(undefined) }, [failed])
  // the newest output is what explains the failure
  const pre = useRef<HTMLPreElement>(null)
  useLayoutEffect(() => { if (pre.current) pre.current.scrollTop = pre.current.scrollHeight }, [logs])

  const retry = async () => {
    setRetrying(true)
    try { await Desktop.retry() } finally { setTimeout(() => setRetrying(false), 800) }
  }
  const showLogs = async () => setLogs(logs ? undefined : await Desktop.backendLogs().catch(() => []))
  const visible = shown || failed

  return (
    <div className={`dt-launch ${launched ? 'over' : ''}`} data-shown={visible ? '' : undefined} role="status" aria-live="polite">
      <div className="dt-launch-body">
        {!launched && <img className="dt-launch-icon" src="/favicon.png" alt="" width={72} height={72} draggable={false} />}
        <h1>{title}</h1>
        <p className="dt-launch-detail">{detail}</p>
        {busy && <div className="dt-progress" role="progressbar" aria-label={detail}><span /></div>}
        {failed && (
          <div className="dt-launch-actions">
            <button className="btn primary" onClick={retry} disabled={retrying}>{retrying ? t('Starting…') : t('Try Again')}</button>
            <button className="btn" onClick={showLogs}>{logs ? t('Hide Details') : t('Show Details')}</button>
            <button className="btn" onClick={() => Desktop.quit()}>{t('Quit')}</button>
          </div>
        )}
        {logs && (
          <>
            <pre ref={pre} className="dt-launch-logs">{logs.length ? logs.slice(-80).join('\n') : t('Nothing was printed.')}</pre>
            {status.logPath && <p className="dt-launch-path">{t('The full log is in {path}', { path: status.logPath })}</p>}
          </>
        )}
      </div>
    </div>
  )
}

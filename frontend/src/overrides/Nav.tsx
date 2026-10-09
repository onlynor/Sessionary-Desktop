import { useState } from 'react'
import { t, useT } from '@web/i18n'
import { Icon } from '@web/Icon'
import { useMachines } from '@web/machines'
import { go, href, type MachineTab, type Route } from '@web/route'
import { NotifyPanel, useNotify } from '@web/notify'
import { Popover } from '@web/SettingsMenu'
import { MachineIcon, StatusDot, machineTarget } from '@web/ui'
import { useControl } from '@web/control'

/**
 * The desktop's sidebar: the web client's navigation as a source list.
 *
 * Differences from the web's (`web/src/Nav.tsx`), on purpose:
 * - no brand row: the window has a title bar and the app has an icon; the row is the title bar, holding the toggle;
 * - Trash belongs to the machine whose sessions it holds, so it sits in that machine's group;
 * - Nodes are machines, reached from Machines and the machine switcher, not a row of their own;
 * - notifications and settings are tools of the window, not places: one quiet row at the foot, with the status.
 * Keep the props and the places it links to in step with the web's.
 */
export function Nav({ route, machineId, collapsed, onCollapse, onPalette, onSettings, trashCount }: {
  route: Route; machineId: string; collapsed: boolean; onCollapse: () => void; onPalette: () => void; onSettings: (anchor: HTMLElement) => void; trashCount: number
}) {
  useT()
  const { machines } = useMachines()
  const [pick, setPick] = useState<{ left: number; top: number }>()
  const notify = useNotify()
  const control = useControl()
  const live = (() => { const open = new Set<string>(); for (const e of control.events) { if (e.phase === 'answering' || e.phase === 'trying') open.add(e.id); else open.delete(e.id) } return open.size > 0 })()
  const [bell, setBell] = useState<{ left: number; bottom: number }>()
  const machine = machines.find((m) => m.id === machineId) ?? machines[0]
  const tab: MachineTab | null = route.page === 'machine' ? route.tab : route.page === 'agent' ? 'agents' : route.page === 'session' ? 'sessions' : null
  const online = machines.filter((m) => m.state === 'online' || m.kind === 'local').length
  const kbd = navigator.platform.toLowerCase().includes('mac') ? '⌘K' : 'Ctrl K'

  const item = (id: string, icon: string, label: string, to: string, on: boolean, meta?: React.ReactNode) => (
    <a key={id} className={`nav-i ${on ? 'on' : ''}`} href={to} aria-current={on ? 'page' : undefined} title={collapsed ? label : undefined}>
      <Icon name={icon} size={collapsed ? 18 : 16} /><span className="nav-label">{label}</span>{meta != null && <span className="meta">{meta}</span>}
    </a>
  )
  const onMachine = (id: string) => (route.page === 'machine' || route.page === 'agent' || route.page === 'session') && (route as { machine: string }).machine === id
  const scoped = (tabId: MachineTab, icon: string, label: string, meta?: React.ReactNode) => item(tabId, icon, label, href.machine(machine?.id ?? 'local', tabId), tab === tabId && onMachine(machine?.id ?? ''), meta)
  const elsewhere = machines.filter((m) => m.id !== machine?.id).reduce((n, m) => n + notify.unreadFor(m.id), 0)

  return (
    <nav className="nav dt-nav" aria-label={t('Navigation')}>
      <div className="nav-head">
        <button className="tb-btn nav-collapse" onClick={onCollapse} title={`${collapsed ? t('Expand sidebar') : t('Collapse sidebar')}  [`} aria-label={collapsed ? t('Expand sidebar') : t('Collapse sidebar')}><Icon name={collapsed ? 'sidebar-open' : 'sidebar'} /></button>
      </div>

      <button className="nav-search" onClick={onPalette} title={`${t('Search all sessions and messages')}  ${kbd}`}>
        <Icon name="search" size={collapsed ? 18 : 14} /><span className="nav-label">{t('Search')}</span><kbd className="nav-label">{kbd}</kbd>
      </button>

      <div className="nav-group">
        {item('home', 'home', t('Home'), href.home, route.page === 'home')}
        {item('machines', 'layers', t('Machines'), href.machines, route.page === 'machines' || route.page === 'nodes', machines.length)}
      </div>

      {machine && (
        <div className="nav-group nav-machine">
          <button className="machine-switch" onClick={(e) => { const r = e.currentTarget.getBoundingClientRect(); setPick(pick ? undefined : { left: r.left, top: r.bottom + 6 }) }}
            aria-haspopup="menu" aria-expanded={!!pick} title={collapsed ? machine.name : t('Switch machine')}>
            <MachineIcon machine={machine} size={collapsed ? 28 : 24} />
            <span className="nav-label ms-text"><span className="ms-name">{machine.name}</span><span className="ms-sub"><StatusDot state={machine.state} />{machineTarget(machine)}</span></span>
            <span className="nav-label ms-chev">{elsewhere > 0 && <span className="alert-dot" title={t('Something on another machine needs you')} />}<Icon name="updown" size={13} /></span>
            {elsewhere > 0 && <span className="alert-dot corner" aria-hidden="true" />}
          </button>
          {scoped('agents', 'task', t('Agents'))}
          {scoped('projects', 'folder', t('Projects'))}
          {scoped('sessions', 'message', t('Sessions'))}
          {scoped('terminal', 'terminal', t('Terminal'), machine && notify.unreadFor(machine.id) > 0 ? <span className="alert-dot" title={t('Something here needs you')} /> : undefined)}
          {scoped('monitor', 'activity', t('Monitor'))}
          {scoped('trash', 'trash', t('Trash'), trashCount > 0 ? trashCount : undefined)}
        </div>
      )}

      <div className="nav-group">
        <div className="nav-section nav-label">{t('Model Control')}</div>
        <span className="nav-sep" aria-hidden="true" />
        {item('models', 'models', t('Models'), href.models(), route.page === 'models', control.state?.providers.length || undefined)}
        {item('routing', 'route', t('Routing'), href.routing(), route.page === 'routing', control.state?.groups.length || undefined)}
        {item('gateway', 'gateway', t('Gateway'), href.gateway, route.page === 'gateway', live ? <span className="live-dot" title={t('Answering now')} /> : undefined)}
        {item('usage', 'usage', t('Usage'), href.usage(), route.page === 'usage')}
      </div>

      <span className="grow" />
      <div className="dt-nav-foot">
        <a className="dt-status nav-label" href={href.machines} title={t('Machines')}>
          <span className={`dot ${online === machines.length ? 'ok' : 'warn'}`} />
          <span className="ellip">{t('{n} of {total} machines online', { n: online, total: machines.length })}</span>
        </a>
        <button className={`tb-btn dt-foot-btn ${notify.unread > 0 ? 'alert' : ''}`} aria-haspopup="menu" title={t('Notifications')} aria-label={t('Notifications')}
          onClick={(e) => { const r = e.currentTarget.getBoundingClientRect(); setBell(bell ? undefined : { left: r.left, bottom: innerHeight - r.top + 6 }) }}>
          <Icon name={notify.settings.enabled ? 'bell' : 'bell-off'} size={16} />
          {notify.unread > 0 && <span className="dt-badge">{notify.unread > 9 ? '9+' : notify.unread}</span>}
        </button>
        <button className="tb-btn dt-foot-btn" onClick={(e) => onSettings(e.currentTarget)} title={t('Settings')} aria-label={t('Settings')}><Icon name="settings" size={16} /></button>
      </div>

      {bell && <NotifyPanel at={bell} onClose={() => setBell(undefined)} />}
      {pick && (
        <Popover at={pick} onClose={() => setPick(undefined)} label={t('Switch machine')} width={264}>
          <div className="menu-label">{t('Machines')}</div>
          {machines.map((m) => (
            <button key={m.id} className="menu-item machine-pick" role="menuitemradio" aria-checked={m.id === machine?.id} onClick={() => { setPick(undefined); go(href.machine(m.id, tab ?? 'agents')) }}>
              <StatusDot state={m.state} /><span className="grow"><b>{m.name}</b><span className="mp-sub">{machineTarget(m)}</span></span>
              {notify.unreadFor(m.id) > 0 && m.id !== machine?.id && <span className="meta badge">{notify.unreadFor(m.id)}</span>}
              {m.id === machine?.id && <span className="check"><Icon name="check" size={14} /></span>}
            </button>
          ))}
          <div className="menu-sep" />
          <button className="menu-item" onClick={() => { setPick(undefined); go(href.nodes({ add: '1' })) }}><Icon name="other" /><span className="grow">{t('Add Node')}</span></button>
          <button className="menu-item" onClick={() => { setPick(undefined); go(href.nodes()) }}><Icon name="server" /><span className="grow">{t('Manage nodes')}</span></button>
        </Popover>
      )}
    </nav>
  )
}

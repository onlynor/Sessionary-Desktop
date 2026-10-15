import { readFileSync } from 'node:fs'
import path from 'node:path'
import { describe, expect, test } from 'bun:test'

// Overrides replace a web client file wholesale (vite.config.ts). These tests undo the documented differences and
// compare the rest with the web client's file, so that a change upstream is carried into the desktop's copy
// rather than silently drifting from it.
const SESSIONARY = path.resolve(process.env.SESSIONARY_DIR ?? path.join(import.meta.dir, '../../../Sessionary'))
// a Windows checkout has CRLF line endings; the comparison is of the text, not of the platform's newlines
const read = (file: string) => readFileSync(file, 'utf8').replace(/\r\n/g, '\n')
const web = (f: string) => read(path.join(SESSIONARY, 'web/src', f))
const own = (f: string) => read(path.join(import.meta.dir, '../src/overrides', f))
// an override imports the client's modules by alias instead of relatively
const aliased = (src: string) => src.replace(/from '\.\/([A-Za-z0-9]+)'/g, "from '@web/$1'")
const header = /^\/\*\*[\s\S]*?\*\/\n/

describe('overrides stay in step with the web client', () => {
  test("MachinePage.tsx is the web client's, but for the agents' display order", () => {
    const undone = own('MachinePage.tsx')
      .replace(header, '')
      .replace("import { orderAgents } from '../desktop/agentOrder'\n", '')
      .replace(
        '{orderAgents(agents, (a) => a.id).map((a) => {',
        '{agents.map((a) => {',
      )
    // a failure here: web/src/MachinePage.tsx changed upstream; copy it again and re-apply the documented changes
    expect(undone).toBe(aliased(web('MachinePage.tsx')))
  })

  test("Nav.tsx is the web client's navigation, but for the platform's shortcut and the desktop marker", () => {
    const undone = own('Nav.tsx')
      .replace(header, '')
      // the search shortcut: the platform's, where the web writes Ctrl K
      .replace("  const kbd = navigator.platform.toLowerCase().includes('mac') ? '⌘K' : 'Ctrl K'\n", '')
      .replaceAll('${kbd}', 'Ctrl K')
      .replaceAll('{kbd}', 'Ctrl K')
      // the marker class the desktop layer finds the sidebar by
      .replace('"nav dt-nav"', '"nav"')
    // a failure here: web/src/Nav.tsx changed upstream; copy it again and re-apply the documented changes
    expect(undone).toBe(aliased(web('Nav.tsx')))
  })

  test('the desktop keeps the web client\'s navigation structure', () => {
    const nav = own('Nav.tsx')
    // the order the sidebar reads in: Home and Machines, the machine's Workspace, Model Control, the foot
    const order = [
      "item('home'", "item('machines'", "t('Workspace')", "scoped('agents'", "scoped('projects'", "scoped('sessions'",
      "scoped('terminal'", "scoped('monitor'", "t('Model Control')", "item('models'", "item('routing'", "item('gateway'",
      "item('usage'", 'nav-foot', "item('trash'", "item('nodes'", 'nav-status',
    ]
    let at = -1
    for (const marker of order) {
      const found = nav.indexOf(marker, at + 1)
      expect(found, `${marker} is in the sidebar, after the one before it`).toBeGreaterThan(at)
      at = found
    }
    // the brand row: the mark, the wordmark, and the collapse control
    expect(nav).toContain('brand-mark')
    expect(nav).toContain('brand-name')
    expect(nav).toContain('nav-collapse')
    // Notifications and Settings are labelled entries, as on the web (not icon-only window tools)
    expect(nav).toContain("t('Notifications')")
    expect(nav).toContain("t('Settings')")
  })
})

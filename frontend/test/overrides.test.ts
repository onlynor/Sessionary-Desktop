import { createHash } from 'node:crypto'
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { describe, expect, test } from 'bun:test'

// Overrides replace a web client file wholesale (vite.config.ts). These tests fail when the web client's file
// changes, so that the desktop's copy is brought up to date rather than silently drifting from it.
const SESSIONARY = path.resolve(process.env.SESSIONARY_DIR ?? path.join(import.meta.dir, '../../../Sessionary'))
const web = (f: string) => readFileSync(path.join(SESSIONARY, 'web/src', f), 'utf8')
const own = (f: string) => readFileSync(path.join(import.meta.dir, '../src/overrides', f), 'utf8')
// an override imports the client's modules by alias instead of relatively
const aliased = (src: string) => src.replace(/from '\.\/([A-Za-z0-9]+)'/g, "from '@web/$1'")

describe('overrides stay in step with the web client', () => {
  test('MachinePage.tsx is the web client\'s, but for the agents\' display order', () => {
    const undone = own('MachinePage.tsx')
      .replace(/^\/\*\*[\s\S]*?\*\/\n/, '')
      .replace("import { orderAgents } from '../desktop/agentOrder'\n", '')
      .replace(
        '{orderAgents(agents, (a) => a.id).map((a) => {',
        '{agents.map((a) => {',
      )
    // a failure here: web/src/MachinePage.tsx changed upstream; copy it again and re-apply the documented changes
    expect(undone).toBe(aliased(web('MachinePage.tsx')))
  })

  test('Nav.tsx was redesigned from the web client\'s current Nav', () => {
    // a failure here: web/src/Nav.tsx changed upstream; carry its change into overrides/Nav.tsx, then update the hash
    expect(createHash('sha256').update(web('Nav.tsx')).digest('hex')).toBe('9b51030da1335523b58f58a8eae07a16349989f3414f10b333d1e42c2a65fe15')
  })
})

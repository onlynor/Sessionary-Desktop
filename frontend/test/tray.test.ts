import { readFileSync } from 'node:fs'
import path from 'node:path'
import { describe, expect, test } from 'bun:test'
import { recentIds, TRAY_LABELS } from '../src/desktop/trayLabels'

const SESSIONARY = path.resolve(process.env.SESSIONARY_DIR ?? path.join(import.meta.dir, '../../../Sessionary'))
const root = path.join(import.meta.dir, '../..')
const catalog = (readFileSync(path.join(SESSIONARY, 'web/src/locales.ts'), 'utf8') + readFileSync(path.join(root, 'frontend/src/desktop/i18n.ts'), 'utf8')).replace(/\r\n/g, '\n')
const goDefaults = readFileSync(path.join(root, 'cmd/sessionary-desktop/traymenu.go'), 'utf8').replace(/\r\n/g, '\n')

// a catalog row is ['English', '简体', '繁體', '日本語'] — all four languages, or the row does not translate
const row = (en: string) => {
  const m = catalog.match(new RegExp(`\\['${en.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}',\\s*'([^']+)',\\s*'([^']+)',\\s*'([^']+)'\\]`))
  return m ? m.slice(1) : null
}

describe('the tray menu speaks the catalog', () => {
  test('every label is a catalog row with all three translations, except the brand name', () => {
    for (const [key, en] of Object.entries(TRAY_LABELS)) {
      if (en === 'Sessionary') continue
      const r = row(en)
      expect(r, `${key}: "${en}" has no row with zh-Hans, zh-Hant and ja`).not.toBeNull()
    }
  })

  test('the app\'s English defaults are the page\'s keys and strings', () => {
    const block = goDefaults.slice(goDefaults.indexOf('var trayDefaults'), goDefaults.indexOf('}', goDefaults.indexOf('var trayDefaults')))
    const go = Object.fromEntries([...block.matchAll(/"(\w+)":\s*"([^"]+)"/g)].map((m) => [m[1], m[2]]))
    expect(go).toEqual({ ...TRAY_LABELS })
  })
})

describe('recent sessions', () => {
  test('are read as the client wrote them, newest first', () => {
    expect(recentIds('["b","a","c"]')).toEqual(['b', 'a', 'c'])
  })
  test('never throw on what is not a list of ids', () => {
    expect(recentIds(null)).toEqual([])
    expect(recentIds('not json')).toEqual([])
    expect(recentIds('{"a":1}')).toEqual([])
    expect(recentIds('["a",3,null,"b"]')).toEqual(['a', 'b'])
  })
})

/**
 * The words of the app's native tray menu, as Sessionary's catalog keys: `key` is what the app (Go,
 * cmd/sessionary-desktop/traymenu.go) asks for, the value is the catalog's English source string, which `t()`
 * translates into the language the user chose. Nothing is translated here: there is one catalog.
 */
export const TRAY_LABELS = {
  show: 'Show Sessionary',
  hide: 'Hide Sessionary',
  new: 'New session',
  recent: 'Recent sessions',
  models: 'Model Control',
  usage: 'Usage',
  settings: 'Settings',
  about: 'About Sessionary',
  quit: 'Quit Sessionary',
  tooltip: 'Sessionary',
} as const

export type TrayLabelKey = keyof typeof TRAY_LABELS

/** the ids the client keeps of the sessions the user opened last, newest first */
export function recentIds(raw: string | null): string[] {
  try {
    const v = JSON.parse(raw ?? '[]')
    return Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : []
  } catch { return [] }
}

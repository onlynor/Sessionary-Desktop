import { currentWindow } from 'mygo-runtime'
import { inApp, openSettings, quitApp } from './runtime'

/**
 * What makes the page behave like an app's window rather than a browser tab: no browser context menu, no
 * dragging of links and images, back and forward from the keyboard and the mouse, toolbars that take on a material
 * when content scrolls under them, and scrollbars that show while scrolling, as overlay scrollbars do.
 */
export function installBehaviors(opts: { dev: boolean }) {
  // ---- the browser's context menu: only where it does something (editing, a selection to copy) ----
  addEventListener('contextmenu', (e) => {
    if (e.defaultPrevented) return // the client's own menu
    const t = e.target as HTMLElement
    const editable = t.closest('input, textarea, [contenteditable="true"], .xterm')
    const selected = (getSelection()?.toString() ?? '').length > 0
    if (editable || selected || (opts.dev && e.shiftKey)) return
    e.preventDefault()
  })

  // ---- links and images are not things to drag out of an app ----
  addEventListener('dragstart', (e) => {
    const t = e.target as HTMLElement | null
    if (t && (t.tagName === 'A' || t.tagName === 'IMG') && !t.closest('[draggable="true"]')) e.preventDefault()
  })

  // ---- back and forward ----
  addEventListener('keydown', (e) => {
    const typing = (e.target as HTMLElement).closest?.('input, textarea, [contenteditable="true"], .xterm')
    if (e.altKey && !e.ctrlKey && !e.metaKey && !e.shiftKey && !typing) {
      if (e.key === 'ArrowLeft') { e.preventDefault(); history.back() }
      else if (e.key === 'ArrowRight') { e.preventDefault(); history.forward() }
    }
    const mod = e.ctrlKey || e.metaKey
    if (mod && !e.altKey && !e.shiftKey && !typing) {
      if (e.key === '[') { e.preventDefault(); history.back() }
      else if (e.key === ']') { e.preventDefault(); history.forward() }
    }
    // the shortcuts a menu bar would carry, where there is none (Linux, Windows); macOS's menu sends the same
    if (mod && !e.altKey && !e.shiftKey) {
      const go = (h: string) => { e.preventDefault(); location.hash = h }
      if (e.key === ',') { e.preventDefault(); openSettings() }
      // an explicit quit, where there is no menu bar to carry one (the tray also offers it)
      else if (e.key === 'q' || e.key === 'Q') { e.preventDefault(); quitApp() }
      else if (e.key === 'n' || e.key === 'N') { e.preventDefault(); dispatchEvent(new CustomEvent('sessionary:new-session', { detail: {} })) }
      else if (e.key === '1') go('#/')
      else if (e.key === '2') go('#/m/local/projects')
      else if (e.key === '3') go('#/m/local/sessions')
      else if (e.key === '4') go('#/models')
      else if (e.key === '5') go('#/usage')
    }
    // a browser's keys that make no sense in an app
    if (mod && !opts.dev && (e.key === 'r' || e.key === 'R' || e.key === 'p' || e.key === 'u' || e.key === 's')) e.preventDefault()
    if (e.key === 'F5' && !opts.dev) e.preventDefault()
    // full screen, as desktop apps on Linux and Windows take it (macOS's menu has its own)
    if (e.key === 'F11' && inApp) { e.preventDefault(); currentWindow.toggleFullScreen() }
  }, true)
  addEventListener('mouseup', (e) => {
    if (e.button === 3) { e.preventDefault(); history.back() }
    else if (e.button === 4) { e.preventDefault(); history.forward() }
  })

  // ---- scroll state: toolbars over scrolled content, overlay scrollbars ----
  const timers = new WeakMap<Element, number>()
  addEventListener('scroll', (e) => {
    const el = e.target
    if (!(el instanceof HTMLElement)) return
    el.toggleAttribute('data-scrolled', el.scrollTop > 2)
    el.toggleAttribute('data-scrolled-far', el.scrollTop > 64)
    el.setAttribute('data-scrolling', '')
    clearTimeout(timers.get(el))
    timers.set(el, window.setTimeout(() => el.removeAttribute('data-scrolling'), 900))
  }, { capture: true, passive: true })

  // ---- a new page starts at its top, with no material over it ----
  addEventListener('hashchange', () => {
    requestAnimationFrame(() => {
      for (const el of document.querySelectorAll<HTMLElement>('.page, .scroll')) {
        el.toggleAttribute('data-scrolled', el.scrollTop > 2)
        el.toggleAttribute('data-scrolled-far', el.scrollTop > 64)
      }
    })
  })

  if (inApp) document.documentElement.toggleAttribute('data-app', true)
}

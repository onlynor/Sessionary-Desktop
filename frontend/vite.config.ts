import react from '@vitejs/plugin-react'
import { existsSync } from 'node:fs'
import path from 'node:path'
import { defineConfig, type Plugin } from 'vite'

/**
 * The desktop frontend is Sessionary's web client (`<sessionary>/web/src`, used in place, never copied) with a
 * desktop layer on top:
 *
 * - `src/` — the desktop's own code: the launch screen, the window's toolbar, native integration, styles;
 * - `src/overrides/<path>` — a file that replaces `<sessionary>/web/src/<path>` wherever the web client imports
 *   it. Overrides are the exception: prefer a stylesheet or a wrapper. An override that needs the original
 *   imports it as `@original/<path>`.
 *
 * Packages resolve from Sessionary's own install, so the two never run two copies of React.
 */
const SESSIONARY = path.resolve(process.env.SESSIONARY_DIR ?? path.join(__dirname, '../../Sessionary'))
const WEB = path.join(SESSIONARY, 'web')
const WEB_SRC = path.join(WEB, 'src')
const OWN_SRC = path.join(__dirname, 'src')
const OVERRIDES = path.join(OWN_SRC, 'overrides')
const MODULES = path.join(SESSIONARY, 'node_modules')

if (!existsSync(WEB_SRC)) throw new Error(`Sessionary's web client was not found at ${WEB_SRC}; set SESSIONARY_DIR`)
if (!existsSync(MODULES)) throw new Error(`Sessionary's packages are not installed: run \`pnpm install\` in ${SESSIONARY}`)

const EXTS = ['', '.tsx', '.ts', '.css']
const isOwn = (file: string) => path.normalize(file).startsWith(path.normalize(OWN_SRC + path.sep))
const findOverride = (file: string) => {
  const rel = path.relative(WEB_SRC, file).replace(/\.(tsx|ts)$/, '')
  if (rel.startsWith('..')) return null
  for (const ext of EXTS) {
    const candidate = path.join(OVERRIDES, rel + ext)
    if (ext && existsSync(candidate)) return candidate
  }
  return null
}

function desktopLayer(): Plugin {
  return {
    name: 'sessionary-desktop-layer',
    enforce: 'pre',
    async resolveId(source, importer, options) {
      if (source.startsWith('@original/')) {
        const r = await this.resolve(path.join(WEB_SRC, source.slice('@original/'.length)), importer, { ...options, skipSelf: true })
        return r ? { ...r, id: r.id } : null
      }
      if (!importer || source.startsWith('\0')) return null
      const r = await this.resolve(source, importer, { ...options, skipSelf: true })
      if (r && !r.external) {
        const file = r.id.split('?')[0]!
        const override = findOverride(file)
        // the override itself, and only it, still reaches the original through @original/
        if (override && importer !== override) return override
        return r
      }
      // a package imported by the desktop's own code: from Sessionary's install
      // (Vite hands importers over with forward slashes, even on Windows, where OWN_SRC has backslashes)
      if (!r && isOwn(importer) && !source.startsWith('.') && !source.startsWith('/')) {
        return this.resolve(source, path.join(WEB_SRC, 'main.tsx'), { ...options, skipSelf: true })
      }
      return r
    },
  }
}

export default defineConfig({
  root: __dirname,
  publicDir: path.join(WEB, 'public'),
  plugins: [desktopLayer(), react()],
  resolve: {
    alias: [
      { find: /^@web\//, replacement: WEB_SRC + '/' },
      { find: /^react$/, replacement: path.join(MODULES, 'react') },
      { find: /^react\/(.*)$/, replacement: path.join(MODULES, 'react') + '/$1' },
      { find: /^react-dom$/, replacement: path.join(MODULES, 'react-dom') },
      { find: /^react-dom\/(.*)$/, replacement: path.join(MODULES, 'react-dom') + '/$1' },
    ],
  },
  server: {
    port: 5174,
    strictPort: true,
    fs: { allow: [__dirname, SESSIONARY] },
  },
  build: { outDir: 'dist', emptyOutDir: true, target: 'es2022', chunkSizeWarningLimit: 2000 },
})

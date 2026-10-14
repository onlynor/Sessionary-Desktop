// Builds the Sessionary server into .stage/server/cli.mjs: one self-contained file (its few packages bundled in),
// which the app runs with Node.js and ships among its resources.
import { createRequire } from 'node:module'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.join(here, '../..')
const sessionary = path.resolve(process.env.SESSIONARY_DIR ?? path.join(root, '../Sessionary'))
const require = createRequire(path.join(sessionary, 'package.json'))
const { build } = await import(require.resolve('tsup'))

await build({
  entry: [path.join(sessionary, 'src/cli.ts')],
  format: ['esm'],
  target: 'node22',
  platform: 'node',
  outDir: path.join(root, '.stage/server'),
  clean: true,
  noExternal: [/.*/],
  // bundled CommonJS dependencies may still call require()
  banner: { js: "import { createRequire as __cr } from 'node:module'; const require = __cr(import.meta.url);" },
  silent: true,
})
console.log('.stage/server/cli.mjs ←', path.relative(root, path.join(sessionary, 'src/cli.ts')))

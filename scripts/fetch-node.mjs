// Stages the Node.js runtime that ships inside the app: .stage/runtime/{bin/node,LICENSE}.
//
// The version and every archive's SHA-256 are pinned here, so builds are reproducible and a tampered or
// truncated download fails the build. Archives are cached in ~/.cache/sessionary-desktop/node (XDG_CACHE_HOME), so
// only the first build downloads. Debug symbols are stripped where `strip` exists (about 18 MB off the binary; the dynamic symbol table addons link against stays).
//
//   node scripts/fetch-node.mjs [os/arch]     default: this machine's, e.g. linux/amd64
import { createHash } from 'node:crypto'
import { execFileSync } from 'node:child_process'
import { chmodSync, copyFileSync, existsSync, mkdirSync, readFileSync, renameSync, rmSync, writeFileSync } from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

export const NODE_VERSION = '24.21.0'
// from https://nodejs.org/dist/v24.21.0/SHASUMS256.txt
const ARCHIVES = {
  'linux/amd64': ['linux-x64.tar.xz', 'fd8e59d5a511510f6a298afb548f18c7d2b1be404d8b4a27d94fbe49f56cb2d6'],
  'linux/arm64': ['linux-arm64.tar.xz', '6ad1325edbdb5649c379b75a237147a666c95d4f9ae8d340fef2d1575d289ad2'],
  'darwin/amd64': ['darwin-x64.tar.gz', '1462cb3b3046b815cf8ea436d3da450ec1a9f11dac7e5a46b0ada5305d7e8097'],
  'darwin/arm64': ['darwin-arm64.tar.gz', 'bed7eea5325e1108f32ce5228ddd6a5f0f08a499ee42aa7442aea583702f6057'],
  'windows/amd64': ['win-x64.zip', '158f7685b44de51f6c0df1d153526cbcd3e1bc739a8dfc607721cef75de9e541'],
  'windows/arm64': ['win-arm64.zip', '8779b1bde1d39f8d420e3b57aa657b39891af434d3de44a919044cec06785921'],
}

const root = path.join(path.dirname(fileURLToPath(import.meta.url)), '..')
const goArch = { x64: 'amd64', arm64: 'arm64' }[process.arch] ?? process.arch
const goOS = { win32: 'windows' }[process.platform] ?? process.platform
const target = process.argv[2] ?? `${goOS}/${goArch}`
const entry = ARCHIVES[target]
if (!entry) throw new Error(`no pinned Node.js for ${target}`)
const [suffix, sha256] = entry
const name = `node-v${NODE_VERSION}-${suffix}`
const cacheDir = path.join(process.env.XDG_CACHE_HOME ?? path.join(os.homedir(), '.cache'), 'sessionary-desktop', 'node')
const archive = path.join(cacheDir, name)
const out = path.join(root, '.stage', 'runtime')

const digest = (file) => createHash('sha256').update(readFileSync(file)).digest('hex')

// ---- the archive, verified ----
mkdirSync(cacheDir, { recursive: true })
if (!existsSync(archive) || digest(archive) !== sha256) {
  const url = `https://nodejs.org/dist/v${NODE_VERSION}/${name}`
  console.log(`downloading ${url}`)
  const res = await fetch(url)
  if (!res.ok) throw new Error(`${url}: ${res.status}`)
  const tmp = archive + '.part'
  writeFileSync(tmp, Buffer.from(await res.arrayBuffer()))
  const got = digest(tmp)
  if (got !== sha256) { rmSync(tmp); throw new Error(`${name}: sha256 ${got}, expected ${sha256}`) }
  renameSync(tmp, archive)
}

// ---- already staged? ----
const stamp = path.join(out, '.version')
if (existsSync(stamp) && readFileSync(stamp, 'utf8') === `${NODE_VERSION} ${target}`) {
  console.log(`.stage/runtime ← Node.js ${NODE_VERSION} (${target}, staged)`)
  process.exit(0)
}

// ---- stage only what the app runs: the binary and its license ----
rmSync(out, { recursive: true, force: true })
const work = path.join(cacheDir, `extract-${process.pid}`)
rmSync(work, { recursive: true, force: true })
mkdirSync(work, { recursive: true })
const dir = name.replace(/\.(tar\.xz|tar\.gz|zip)$/, '')
const windows = target.startsWith('windows/')
const binary = windows ? `${dir}/node.exe` : `${dir}/bin/node`
// tar reads all three: GNU tar the .tar.xz of Linux, bsdtar (macOS, Windows) the .tar.gz and .zip
execFileSync('tar', ['-xf', archive, '-C', work, binary, `${dir}/LICENSE`])
mkdirSync(path.join(out, 'bin'), { recursive: true })
const dest = path.join(out, 'bin', windows ? 'node.exe' : 'node')
copyFileSync(path.join(work, binary), dest)
copyFileSync(path.join(work, dir, 'LICENSE'), path.join(out, 'LICENSE'))
rmSync(work, { recursive: true, force: true })
if (!windows) chmodSync(dest, 0o755)
if (target.startsWith('linux/') && target === `${goOS}/${goArch}`) {
  try { execFileSync('strip', [dest]) } catch { console.warn('strip not found: shipping the runtime unstripped') }
}
writeFileSync(stamp, `${NODE_VERSION} ${target}`)
if (target === `${goOS}/${goArch}`) {
  const v = execFileSync(dest, ['--version']).toString().trim()
  if (v !== `v${NODE_VERSION}`) throw new Error(`staged runtime says ${v}`)
}
console.log(`.stage/runtime ← Node.js ${NODE_VERSION} (${target})`)

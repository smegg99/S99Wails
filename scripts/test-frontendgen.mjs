import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import test from 'node:test'
import { runInNewContext } from 'node:vm'
import ts from 'typescript'

const require = createRequire(import.meta.url)

function asset(name, imports = {}, globals = {}) {
  const source = readFileSync(new URL(`../frontendgen/assets/${name}.ts`, import.meta.url), 'utf8')
  const { outputText } = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  })
  const exports = {}
  runInNewContext(outputText, { exports, require: name => imports[name] ?? require(name), ...globals })
  return exports
}

function themeHarness(native = true) {
  let listener
  let cleanup
  let resolve
  let subscribed = false
  const promise = new Promise((done) => {
    resolve = done
  })
  const vue = require('vue')
  const { useSystemTheme } = asset('systemTheme', {
    './environment': { isWails: () => native },
    'vue': { ...vue, getCurrentInstance: () => ({}), onUnmounted: (fn) => {
      cleanup = fn
    } },
    '@wailsio/runtime': { Events: { On: (_name, fn) => {
      subscribed = true
      listener = fn
      return () => {
        listener = undefined
      }
    } } },
  })
  const { theme } = useSystemTheme({ getTheme: () => {
    assert.equal(subscribed, true, 'subscribe before taking the initial snapshot')
    return promise
  } })
  return { theme, resolve, emit: data => listener?.({ data }), dispose: () => cleanup?.() }
}

const light = { dark_mode: false, accent_color: '#123456', scheme: 'light', accent_watch_supported: true }
const dark = { ...light, dark_mode: true, accent_color: '#00ff00', scheme: 'dark' }

test('initial theme loads and subsequent events remain live', async () => {
  const h = themeHarness()
  h.resolve(light)
  await Promise.resolve()
  assert.equal(h.theme.value.dark_mode, false)
  h.emit([dark])
  assert.equal(h.theme.value.accent_color, '#00ff00')
})

test('newer event wins over delayed startup snapshot', async () => {
  const h = themeHarness()
  h.emit(dark)
  h.resolve(light)
  await Promise.resolve()
  assert.equal(h.theme.value.dark_mode, true)
  assert.equal(h.theme.value.accent_color, '#00ff00')
})

test('unmount ignores pending startup and unsubscribes events', async () => {
  const h = themeHarness()
  h.dispose()
  h.emit(dark)
  h.resolve(light)
  await Promise.resolve()
  assert.equal(h.theme.value, null)
})

test('web context makes no native calls', () => {
  assert.equal(themeHarness(false).theme.value, null)
})

test('hex channels use full supported values and ignore only valid alpha', () => {
  const { parseHex, foregroundOn } = asset('contrast')
  for (const value of ['#123456', '123456', ' #123456ab ']) {
    assert.deepEqual({ ...parseHex(value) }, { r: 18, g: 52, b: 86 })
  }
  for (const value of ['#abc', '#abcd', 'ABC']) {
    assert.deepEqual({ ...parseHex(value) }, { r: 170, g: 187, b: 204 })
  }
  for (const value of ['#12345', '#1234567', '#123456789', '#123xyz', 'red', '', '#']) {
    assert.equal(parseHex(value), null, value)
  }
  assert.equal(foregroundOn('#00ff00'), '#000000')
  assert.equal(foregroundOn('#000000'), '#ffffff')
  assert.equal(foregroundOn('#ffffff'), '#000000')
})

function heartbeatHarness(visibility) {
  const calls = []
  const listeners = {}
  const timers = []
  const document = {
    visibilityState: visibility,
    addEventListener: (name, fn) => { listeners[name] = fn },
    removeEventListener: (name) => { delete listeners[name] },
  }
  const { installCrashHeartbeat } = asset('crashScreen', {
    '@wailsio/runtime': {
      Call: {
        ByName: (name) => {
          calls.push(name.split('.').pop())
          return Promise.resolve()
        },
      },
      Events: {},
    },
    './environment': { isWails: () => true },
  }, {
    document,
    setInterval: fn => timers.push(fn),
    clearInterval: () => { timers.length = 0 },
  })
  return { calls, listeners, timers, document, stop: installCrashHeartbeat() }
}

test('crash heartbeat beats while visible and rests while hidden', () => {
  const h = heartbeatHarness('visible')
  h.timers[0]()
  assert.deepEqual(h.calls, ['Beat', 'Beat'])
  h.document.visibilityState = 'hidden'
  h.listeners.visibilitychange()
  h.timers[0]()
  assert.deepEqual(h.calls, ['Beat', 'Beat', 'Rest'])
  h.document.visibilityState = 'visible'
  h.listeners.visibilitychange()
  h.stop()
  assert.deepEqual(h.calls, ['Beat', 'Beat', 'Rest', 'Beat', 'Rest'])
  assert.equal(h.timers.length, 0)
  assert.equal(h.listeners.visibilitychange, undefined)
})

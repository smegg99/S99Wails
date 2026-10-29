// frontendgen/tests/crash-screen.spec.ts
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { crashActions, installCrashHeartbeat, onCrashReport } from '../assets/crashScreen'
import { calls, emit, resetRuntime } from './runtime'

vi.mock('@wailsio/runtime', async () => (await import('./runtime')).runtime)
vi.mock('../assets/environment', () => ({ isWails: () => true }))

const crash = 'github.com/smegg99/s99wails/v2/shell/crash'

function visibility(state: DocumentVisibilityState) {
  Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => state })
  document.dispatchEvent(new Event('visibilitychange'))
}

describe('installCrashHeartbeat', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] })
    visibility('visible')
  })
  afterEach(() => {
    vi.useRealTimers()
    resetRuntime()
  })

  it('beats while visible and rests while hidden', () => {
    const stop = installCrashHeartbeat()
    vi.advanceTimersByTime(2000)
    visibility('hidden')
    vi.advanceTimersByTime(2000)
    visibility('visible')
    stop()
    vi.advanceTimersByTime(4000)
    visibility('hidden')
    expect(calls.map(call => call.name)).toEqual([
      `${crash}.Heartbeat.Beat`,
      `${crash}.Heartbeat.Beat`,
      `${crash}.Heartbeat.Rest`,
      `${crash}.Heartbeat.Beat`,
      `${crash}.Heartbeat.Rest`,
    ])
  })
})

describe('crash screen calls', () => {
  afterEach(resetRuntime)

  it('call the crash service by name', async () => {
    await crashActions.setTitle('stopped')
    expect(calls).toEqual([{ name: `${crash}.Service.SetTitle`, args: ['stopped'] }])
  })

  it('hand each later report to the callback', () => {
    const seen: unknown[] = []
    const stop = onCrashReport(report => seen.push(report.kind))
    emit('s99crash:report', { kind: 'hang' })
    stop()
    emit('s99crash:report', { kind: 'crash' })
    expect(seen).toEqual(['hang'])
  })
})

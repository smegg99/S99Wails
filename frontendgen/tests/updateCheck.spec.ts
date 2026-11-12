// frontendgen/tests/updateCheck.spec.ts
import { afterEach, describe, expect, it, vi } from 'vitest'
import { updateActions } from '../assets/updateCheck'
import { calls, emit, listeners, resetRuntime, runtime } from './runtime'

vi.mock('@wailsio/runtime', async () => (await import('./runtime')).runtime)

const service = 'github.com/smegg99/s99wails/v2/shell/update.Service'

describe('updateActions', () => {
  afterEach(resetRuntime)

  it('requests state in the current language by service name', async () => {
    await updateActions.state('pl')
    expect(calls).toEqual([{ name: `${service}.State`, args: ['pl'] }])
  })

  it('calls each update action by name', async () => {
    await updateActions.ignore('1.5.0')
    await updateActions.notesShown()
    await updateActions.download()
    await updateActions.check()
    expect(calls).toEqual([
      { name: `${service}.Ignore`, args: ['1.5.0'] },
      { name: `${service}.NotesShown`, args: [] },
      { name: `${service}.Download`, args: [] },
      { name: `${service}.Check`, args: [] },
    ])
  })

  it('passes event data and unsubscribes on disposal', () => {
    const seen: unknown[] = []
    const release = { version: '1.5.0', url: 'https://example.test/downloads' }
    const stop = updateActions.onAvailable(value => seen.push(value))
    expect(runtime.Events.On).toHaveBeenCalledWith('s99update:available', expect.any(Function))
    emit('s99update:available', release)
    stop()
    emit('s99update:available', release)
    expect(seen).toEqual([release])
    expect(listeners.size).toBe(0)
  })
})

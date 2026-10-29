// frontendgen/tests/edit-keys.spec.ts
import { afterEach, describe, expect, it, vi } from 'vitest'
import { installDesktopEditKeys } from '../assets/editKeys'
import { emit, listeners, resetRuntime } from './runtime'

vi.mock('@wailsio/runtime', async () => (await import('./runtime')).runtime)

describe('installDesktopEditKeys', () => {
  afterEach(() => {
    resetRuntime()
    document.body.innerHTML = ''
  })

  it('replays the native key on the focused element and stops on dispose', () => {
    const exec = vi.fn()
    Object.defineProperty(document, 'execCommand', { configurable: true, value: exec })
    const field = document.createElement('input')
    document.body.append(field)
    field.focus()
    const keys: string[] = []
    field.addEventListener('keydown', event => keys.push(`${event.ctrlKey ? 'ctrl+' : ''}${event.shiftKey ? 'shift+' : ''}${event.key}`))
    const stop = installDesktopEditKeys()
    emit('edit:key', 'undo')
    emit('edit:key', 'redo')
    stop()
    emit('edit:key', 'undo')
    expect(keys).toEqual(['ctrl+z', 'ctrl+shift+Z'])
    expect(exec.mock.calls).toEqual([['undo'], ['redo']])
    expect(listeners.size).toBe(0)
  })

  it('keeps a middle release from pasting into a field it is not over', () => {
    const field = document.createElement('textarea')
    const other = document.createElement('div')
    document.body.append(field, other)
    field.focus()
    const stop = installDesktopEditKeys()
    const release = new MouseEvent('mouseup', { button: 1, bubbles: true, cancelable: true })
    other.dispatchEvent(release)
    stop()
    expect(release.defaultPrevented).toBe(true)
  })
})

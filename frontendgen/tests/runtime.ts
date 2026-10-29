// frontendgen/tests/runtime.ts
// A fake @wailsio/runtime: records calls and lets a spec emit events.
import { vi } from 'vitest'

type Listener = (event: { data?: unknown }) => void

export const listeners = new Map<string, Listener>()
export const calls: { name: string, args: unknown[] }[] = []

export const runtime = {
  Events: {
    On: vi.fn((name: string, fn: Listener) => {
      listeners.set(name, fn)
      return () => listeners.delete(name)
    }),
  },
  Call: {
    ByName: vi.fn((name: string, ...args: unknown[]) => {
      calls.push({ name, args })
      return Promise.resolve()
    }),
  },
}

/** Sends data to the listener of name, as the Go side would. */
export function emit(name: string, data: unknown) {
  listeners.get(name)?.({ data })
}

export function resetRuntime() {
  listeners.clear()
  calls.length = 0
  runtime.Events.On.mockClear()
  runtime.Call.ByName.mockClear()
}

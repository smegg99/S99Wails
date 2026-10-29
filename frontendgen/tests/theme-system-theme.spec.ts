// frontendgen/tests/theme-system-theme.spec.ts
import { createApp, defineComponent, type Ref } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useSystemTheme, type SystemThemeInfo } from '../assets/systemTheme'
import { emit, listeners, resetRuntime, runtime } from './runtime'

const native = vi.hoisted(() => ({ on: true }))
vi.mock('@wailsio/runtime', async () => (await import('./runtime')).runtime)
vi.mock('../assets/environment', () => ({ isWails: () => native.on }))

const light: SystemThemeInfo = { dark_mode: false, accent_color: '#123456', scheme: 'light', accent_watch_supported: true }
const dark: SystemThemeInfo = { ...light, dark_mode: true, accent_color: '#00ff00', scheme: 'dark' }

// mounted runs useSystemTheme in a component, so unmounting it disposes the composable.
function mounted() {
  let resolve: (info: SystemThemeInfo) => void = () => {}
  let subscribed = false
  let theme!: Readonly<Ref<SystemThemeInfo | null>>
  const app = createApp(defineComponent({
    setup() {
      ({ theme } = useSystemTheme({
        getTheme: () => {
          subscribed = listeners.size > 0
          return new Promise(done => (resolve = done))
        },
      }))
      return () => null
    },
  }))
  app.mount(document.createElement('div'))
  return { theme, resolve: (info: SystemThemeInfo) => resolve(info), subscribed: () => subscribed, unmount: () => app.unmount() }
}

describe('useSystemTheme', () => {
  afterEach(() => {
    resetRuntime()
    native.on = true
  })

  it('subscribes before it asks for the first snapshot', () => {
    expect(mounted().subscribed()).toBe(true)
  })

  it('loads the first theme and stays live for events', async () => {
    const h = mounted()
    h.resolve(light)
    await Promise.resolve()
    expect(h.theme.value?.dark_mode).toBe(false)
    emit('theme:changed', [dark])
    expect(h.theme.value?.accent_color).toBe('#00ff00')
  })

  it('lets a newer event win over a late startup snapshot', async () => {
    const h = mounted()
    emit('theme:changed', dark)
    h.resolve(light)
    await Promise.resolve()
    expect(h.theme.value?.dark_mode).toBe(true)
  })

  it('ignores a pending snapshot and unsubscribes on unmount', async () => {
    const h = mounted()
    h.unmount()
    emit('theme:changed', dark)
    h.resolve(light)
    await Promise.resolve()
    expect(h.theme.value).toBeNull()
    expect(listeners.size).toBe(0)
  })

  it('makes no native calls on the web', () => {
    native.on = false
    expect(mounted().theme.value).toBeNull()
    expect(runtime.Events.On).not.toHaveBeenCalled()
  })
})

// Opaque view transfers between S99AB tab collections and native Wails windows.
import { Events } from '@wailsio/runtime'
import { computed, nextTick, onScopeDispose, ref, shallowRef, watch } from 'vue'
import type { NativeTabDrag, WorkspaceTab, TabWorkspace, TabStripGeometry } from '@smegg99/s99ab/tabs'
import { isWails } from './environment'

export interface WindowInfo { name: string, label: string, ready: boolean }
export interface WindowOffer {
  id: string
  source: string
  target: string
  tab: string
  slot: number
  payload: unknown
  claimed: boolean
}
export interface WindowDrop { id: string, tab: string, target: string, slot: number, x: number, y: number }
interface TransferPayload<P> { owner: string, value: P }

/** Supply generated bindings; the bridge never imports a consuming app. */
export interface WindowBindings {
  Name: () => Promise<string>
  Ready: (owner: string, title: string, ready: boolean) => Promise<WindowOffer[]>
  List: () => Promise<WindowInfo[]>
  Move: (owner: string, target: string, tab: string, slot: number, payload: string) => Promise<WindowOffer>
  TearOff: (owner: string, tab: string, payload: string, x: number, y: number, positioned: boolean) => Promise<WindowOffer>
  Claim: (owner: string, id: string) => Promise<void>
  Released: (id: string) => Promise<void>
  Cancel: (id: string) => Promise<void>
  CloseReady: (token: string, ready: boolean) => Promise<void>
  BeginDrag: (owner: string, id: string, tab: string) => Promise<boolean>
  CancelDrag: (id: string) => Promise<void>
  Strip: (geometry: TabStripGeometry) => Promise<void>
}
export interface DesktopTabsOptions<T extends WorkspaceTab, P> {
  bindings: WindowBindings
  workspace: TabWorkspace<T> & { release: (id: string) => void }
  owner: () => string
  title: string
  layoutReady: () => boolean
  flush: () => Promise<void>
  settle: () => Promise<void>
  capture: (tab: T) => P
  adopt: (payload: P, id: string, slot: number) => Promise<void>
  blocked: (tab: T) => string | undefined
  errorKey: (error: unknown) => string
  report: (operation: string, error: unknown) => void
  /** Event transport can also be injected for host integration tests. */
  events?: (name: string, callback: (event: { data: unknown }) => void) => () => void
}

export function bridgeDesktopTabs<T extends WorkspaceTab, P>(options: DesktopTabsOptions<T, P>) {
  const bindings = options.bindings
  const ws = options.workspace
  const cleanup: (() => void)[] = []
  const listen = (name: string, callback: (event: { data: unknown }) => void) => {
    cleanup.push((options.events ?? Events.On)(name, callback))
  }
  onScopeDispose(() => {
    cancelDrag()
    for (const dispose of cleanup) dispose()
  })
  const name = ref('')
  const list = shallowRef<WindowInfo[]>([])
  const closing = ref(false)
  const incoming = shallowRef<WindowDrop | null>(null)
  const dragging = ref('')
  const failure = ref('')
  const transfers = new Map<string, string>()
  const pending = shallowRef<Record<string, WindowOffer>>({})
  const handling = new Set<string>()
  const finished = new Set<string>()
  let installed = false
  const owner = options.owner
  let title = options.title
  const cancellable = computed(() => Object.values(pending.value).filter(offer => offer.source === name.value && !offer.claimed))
  const remember = (offer: WindowOffer) => {
    pending.value = { ...pending.value, [offer.id]: offer }
  }
  function forget(id: string) {
    const rest = { ...pending.value }
    delete rest[id]
    pending.value = rest
    transfers.delete(id)
    finished.add(id)
  }
  watch(owner, () => {
    pending.value = {}
    transfers.clear()
    finished.clear()
    closing.value = false
    incoming.value = null
    if (dragging.value) void bindings.CancelDrag(dragging.value)
    dragging.value = ''
    failure.value = ''
  })

  async function arrive(offer: WindowOffer) {
    if (offer.target !== name.value || offer.claimed || handling.has(offer.id) || finished.has(offer.id)) return
    const payload = offer.payload as TransferPayload<P>
    if (payload.owner !== owner()) return
    handling.add(offer.id)
    remember(offer)
    try {
      const id = `transfer:${offer.id}`
      await options.adopt(payload.value, id, offer.slot < 0 ? ws.tabs.length : offer.slot)
      transfers.set(offer.id, id)
      await nextTick()
      await options.flush()
      if (payload.owner !== owner()) return
      await bindings.Claim(payload.owner, offer.id)
    } catch (err) {
      failure.value = options.errorKey(err)
    } finally {
      handling.delete(offer.id)
    }
  }

  async function release(offer: WindowOffer) {
    if (offer.source !== name.value || !offer.claimed || handling.has(offer.id) || finished.has(offer.id)) return
    if ((offer.payload as TransferPayload<P>).owner !== owner()) return
    handling.add(offer.id)
    remember(offer)
    ws.release(offer.tab)
    await nextTick()
    try {
      await options.flush()
      await bindings.Released(offer.id)
    } catch (err) {
      failure.value = options.errorKey(err)
    } finally {
      handling.delete(offer.id)
    }
  }

  async function ready(phrase?: string) {
    if (phrase !== undefined) title = phrase
    if (!isWails()) return
    const pending = await bindings.Ready(owner(), title, options.layoutReady())
    list.value = await bindings.List()
    for (const offer of pending) {
      remember(offer)
      if (offer.claimed) await release(offer)
      else await arrive(offer)
    }
  }

  async function init() {
    if (installed || !isWails()) return
    installed = true
    const blockKeys = (event: KeyboardEvent) => {
      if (!closing.value) return
      event.preventDefault()
      event.stopImmediatePropagation()
    }
    window.addEventListener('keydown', blockKeys, true)
    cleanup.push(() => window.removeEventListener('keydown', blockKeys, true))
    // Listeners exist before Ready, so the new window cannot miss an offer.
    listen('windows:changed', (event) => {
      list.value = event.data as WindowInfo[]
    })
    listen('tab:arrive', (event) => {
      void arrive(event.data as WindowOffer)
    })
    listen('tab:release', (event) => {
      void release(event.data as WindowOffer)
    })
    listen('tab:cancel', (event) => {
      const id = transfers.get(event.data as string)
      if (id) ws.release(id)
      forget(event.data as string)
      void options.flush().catch((err) => {
        failure.value = options.errorKey(err)
      })
    })
    listen('tab:cancelled', (event) => {
      const offer = event.data as WindowOffer
      const tab = ws.tabs.find(tab => tab.id === offer.tab)
      if (tab) tab.moving = false
      forget(offer.id)
    })
    listen('tab:done', event => forget(event.data as string))
    listen('tab:hover', event => (incoming.value = event.data as WindowDrop | null))
    listen('tab:drag-end', () => (dragging.value = ''))
    listen('tab:drop', (event) => {
      const drop = event.data as WindowDrop
      if (drop.id !== dragging.value) return
      dragging.value = ''
      const tab = ws.tabs.find(tab => tab.id === drop.tab)
      if (!tab) return
      if (drop.target === name.value) {
        ws.move(tab.id, drop.slot)
        ws.activate(tab.id)
      } else void move(tab, drop.target, drop.slot, drop.target ? undefined : { x: drop.x, y: drop.y, positioned: true })
    })
    listen('window:blocked', (event) => {
      failure.value = event.data as string
    })
    listen('window:resume', () => {
      closing.value = false
    })
    listen('window:close', async (event) => {
      closing.value = true
      failure.value = ''
      try {
        // Finish a field's ordinary blur save before closing its native view.
        if (document.activeElement instanceof HTMLElement) document.activeElement.blur()
        await options.settle()
        await options.flush()
        await bindings.CloseReady(event.data as string, true)
      } catch (err) {
        closing.value = false
        failure.value = options.errorKey(err)
        await bindings.CloseReady(event.data as string, false).catch(error => options.report('window close', error))
      }
    })
    name.value = await bindings.Name()
    await ready()
  }

  async function move(tab: T, target = '', slot = -1, position?: { x: number, y: number, positioned: boolean }) {
    if (tab.pinned || tab.moving) return
    const blocked = options.blocked(tab)
    if (blocked) {
      failure.value = blocked
      return
    }
    const moveOwner = owner()
    failure.value = ''
    tab.moving = true
    try {
      await options.settle()
      if (owner() !== moveOwner) {
        tab.moving = false
        return
      }
      const payload: TransferPayload<P> = { owner: owner(), value: options.capture(tab) }
      const offer = position
        ? await bindings.TearOff(owner(), tab.id, JSON.stringify(payload), position.x, position.y, position.positioned)
        : await bindings.Move(owner(), target, tab.id, slot, JSON.stringify(payload))
      if (!finished.has(offer.id)) remember(offer)
    } catch (err) {
      tab.moving = false
      failure.value = options.errorKey(err)
    }
  }

  async function retry() {
    failure.value = ''
    try {
      await ready()
    } catch (err) { failure.value = options.errorKey(err) }
  }

  async function cancel(id: string) {
    try {
      await bindings.Cancel(id)
    } catch (err) {
      failure.value = options.errorKey(err)
      await retry()
    }
  }

  async function beginDrag(tab: T): Promise<boolean> {
    if (tab.moving || tab.pinned || closing.value || options.blocked(tab) || !isWails()) return false
    const id = crypto.randomUUID()
    dragging.value = id
    try {
      const supported = await bindings.BeginDrag(owner(), id, tab.id)
      if (dragging.value !== id) {
        await bindings.CancelDrag(id)
        return supported
      }
      return supported
    } catch (err) {
      dragging.value = ''
      failure.value = options.errorKey(err)
      return true
    }
  }
  function cancelDrag() {
    const id = dragging.value
    dragging.value = ''
    if (id) void bindings.CancelDrag(id).catch(err => (failure.value = options.errorKey(err)))
  }
  const dragHooks = computed<NativeTabDrag<T> | undefined>(() => isWails()
    ? {
        incoming: incoming.value, dragging: dragging.value, closing: closing.value,
        allowed: tab => !options.blocked(tab), beginDrag, cancelDrag,
        tearOff: (tab) => { void move(tab, '', -1, { x: 0, y: 0, positioned: false }) },
        strip: (geometry) => { void bindings.Strip(geometry).catch(err => options.report('tab strip', err)) },
      }
    : undefined)
  return { name, list, closing, failure, incoming, dragging, dragHooks, cancellable, init, ready, move, retry, cancel, beginDrag, cancelDrag }
}

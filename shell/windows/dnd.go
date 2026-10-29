// shell/windows/dnd.go
// Wayland tab drag: the source window follows the tab in its own coordinates; past it a
// GTK drag with a private target takes over and each window reports hovers in its own.
package windows

import (
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type dndKind int

const (
	dndMotion dndKind = iota
	dndLeave
	dndDrop
	dndFailed
	dndEnd
)

// GtkDragResult values a drag-failed signal carries.
const (
	dragResultSuccess = iota
	dragResultNoTarget
	dragResultUserCancelled
	dragResultTimeoutExpired
	dragResultGrabBroken
	dragResultError
)

type dndEvent struct {
	kind         dndKind
	handle       uintptr
	x, y, result int
}

// dndSink queues the GTK callbacks of the one drag in progress. Posting from the GUI thread never
// blocks and nothing is dropped: a motion only replaces the motion queued right before it.
var dndSink struct {
	mu     sync.Mutex
	wake   chan struct{}
	events []dndEvent
}

func dndPost(e dndEvent) {
	dndSink.mu.Lock()
	wake := dndSink.wake
	if wake == nil {
		dndSink.mu.Unlock()
		return
	}
	n := len(dndSink.events)
	if e.kind == dndMotion && n > 0 && dndSink.events[n-1].kind == dndMotion && dndSink.events[n-1].handle == e.handle {
		dndSink.events[n-1] = e
	} else {
		dndSink.events = append(dndSink.events, e)
	}
	dndSink.mu.Unlock()
	select {
	case wake <- struct{}{}:
	default:
	}
}

// dndListen starts queueing callbacks; the returned channel signals that dndTake has events.
func dndListen() <-chan struct{} {
	dndSink.mu.Lock()
	defer dndSink.mu.Unlock()
	dndSink.wake, dndSink.events = make(chan struct{}, 1), nil
	return dndSink.wake
}

func dndStop() {
	dndSink.mu.Lock()
	defer dndSink.mu.Unlock()
	dndSink.wake, dndSink.events = nil, nil
}

// dndTake returns the queued callbacks in order and empties the queue.
func dndTake() []dndEvent {
	dndSink.mu.Lock()
	defer dndSink.mu.Unlock()
	events := dndSink.events
	dndSink.events = nil
	return events
}

// dragEnding is the event the source window gets when GTK ends the drag. Wayland reports a drop
// outside the app and Escape as one cancellation; both tear the tab off, as on X11. A drag GTK
// broke (grab lost, timeout, error) ends with the tab left in place.
func dragEnding(d *nativeDrag, dropped *Drop, failed int) (string, any) {
	switch failed {
	case dragResultTimeoutExpired, dragResultGrabBroken, dragResultError:
		return "tab:drag-end", d.id
	}
	if dropped != nil {
		return "tab:drop", *dropped
	}
	return "tab:drop", Drop{ID: d.id, Tab: d.tab, Slot: -1}
}

// handle names a window for the C callbacks, which cannot carry a Go string.
func (s *Service) handle(name string) uintptr {
	if s.handles == nil {
		s.handles, s.handleNames = map[string]uintptr{}, map[uintptr]string{}
	}
	if h, ok := s.handles[name]; ok {
		return h
	}
	s.nextHandle++
	s.handles[name], s.handleNames[s.nextHandle] = s.nextHandle, name
	return s.nextHandle
}

// target resolves a drop site to the window under it and its strip slot.
func (s *Service) target(d *nativeDrag, handle uintptr, x, y int) Drop {
	s.mu.Lock()
	name := s.handleNames[handle]
	strip := s.strips[name]
	s.mu.Unlock()
	drop := Drop{ID: d.id, Tab: d.tab, Slot: -1, X: x, Y: y}
	source := ""
	if name == d.source {
		source = d.tab
	}
	if slot := strip.DropSlot(x, y, source); name != "" && slot >= 0 {
		drop.Target, drop.Slot = name, slot
	}
	return drop
}

func (s *Service) followWayland(d *nativeDrag) {
	wake := dndListen()
	ticker := time.NewTicker(16 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(time.Minute)
	defer deadline.Stop()
	started := false
	lastTarget := ""
	seen := 0
	// Over its own window the page draws the tab and the drag's picture is blank; the
	// GTK drag holds the pointer, so the page is told where it is.
	overSource := true
	var dropped *Drop
	failed := dragResultSuccess
	defer func() {
		dndStop()
		application.InvokeSync(func() {
			if started {
				dndCancel()
			}
			dndClear()
		})
		if lastTarget != "" && lastTarget != d.source {
			s.registry.Emit(lastTarget, "tab:hover", nil)
		}
		s.mu.Lock()
		if s.drag == d {
			s.drag = nil
		}
		s.mu.Unlock()
	}()
	hover := func(drop Drop) {
		if drop.Target != lastTarget && lastTarget != "" && lastTarget != d.source {
			s.registry.Emit(lastTarget, "tab:hover", nil)
		}
		lastTarget = drop.Target
		if drop.Target != "" && drop.Target != d.source {
			drop.Face = d.face
			s.registry.Emit(drop.Target, "tab:hover", drop)
		}
	}
	// track moves the pointer onto the source window or off it.
	track := func(inside bool, x, y int) {
		if inside != overSource {
			overSource = inside
			application.InvokeSync(func() { dndIcon(!inside) })
		}
		s.registry.Emit(d.source, "tab:track", Track{Inside: inside, X: x, Y: y})
	}
	for {
		select {
		case <-d.cancel:
			return
		case <-deadline.C:
			s.registry.Emit(d.source, "tab:drag-end", d.id)
			return
		case <-wake:
			for _, e := range dndTake() {
				if !started {
					continue
				}
				s.mu.Lock()
				name := s.handleNames[e.handle]
				s.mu.Unlock()
				switch e.kind {
				case dndMotion:
					if name == d.source {
						track(true, e.x, e.y)
					} else if overSource {
						track(false, 0, 0)
					}
					hover(s.target(d, e.handle, e.x, e.y))
				case dndLeave:
					if name == d.source && overSource {
						track(false, 0, 0)
					}
					if name == lastTarget {
						hover(Drop{})
					}
				case dndDrop:
					drop := s.target(d, e.handle, e.x, e.y)
					dropped = &drop
				case dndFailed:
					failed = e.result
				case dndEnd:
					started = false
					event, payload := dragEnding(d, dropped, failed)
					s.registry.Emit(d.source, event, payload)
					return
				}
			}
		case <-ticker.C:
			image, version := s.pendingImage(d, seen)
			if started {
				if version != seen {
					seen = version
					application.InvokeSync(func() {
						if image != nil && dndImage(image.png, image.scale, image.x, image.y) {
							dndIcon(!overSource)
						}
					})
				}
				continue
			}
			s.mu.Lock()
			native := make(map[string]*application.WebviewWindow, len(s.native))
			handles := make(map[string]uintptr, len(s.native))
			for name, win := range s.native {
				native[name], handles[name] = win, s.handle(name)
			}
			strip := s.strips[d.source]
			s.mu.Unlock()
			source := native[d.source]
			if source == nil {
				s.registry.Emit(d.source, "tab:drag-end", d.id)
				return
			}
			drop := Drop{ID: d.id, Tab: d.tab, Slot: -1}
			var held, inside, ok bool
			application.InvokeSync(func() {
				for name, win := range native {
					dndAttach(win.NativeWindow(), handles[name])
				}
				if version != seen && image != nil {
					dndImage(image.png, image.scale, image.x, image.y)
				}
				drop.X, drop.Y, held, inside, ok = dndLocal(source.NativeWindow())
				if ok && held && !inside {
					started = dndBegin(source.NativeWindow())
				}
			})
			seen = version
			if started {
				track(false, 0, 0)
			}
			select {
			case <-d.cancel:
				return
			default:
			}
			if !ok {
				s.registry.Emit(d.source, "tab:drag-end", d.id)
				return
			}
			if held {
				continue
			}
			// Released before leaving the window, or where no GTK drag could start.
			if inside {
				if slot := strip.DropSlot(drop.X, drop.Y, d.tab); slot >= 0 {
					drop.Target, drop.Slot = d.source, slot
				}
			}
			s.registry.Emit(d.source, "tab:drop", drop)
			return
		}
	}
}

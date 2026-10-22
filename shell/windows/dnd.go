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

type dndEvent struct {
	kind         dndKind
	handle       uintptr
	x, y, result int
}

// dndSink receives the GTK callbacks of the one drag in progress. Posting from the GUI
// thread never blocks; a dropped motion is superseded by the next one.
var dndSink struct {
	mu     sync.Mutex
	events chan dndEvent
}

func dndPost(e dndEvent) {
	dndSink.mu.Lock()
	events := dndSink.events
	dndSink.mu.Unlock()
	if events == nil {
		return
	}
	select {
	case events <- e:
	default:
	}
}

func dndListen(events chan dndEvent) {
	dndSink.mu.Lock()
	dndSink.events = events
	dndSink.mu.Unlock()
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
	events := make(chan dndEvent, 1024)
	dndListen(events)
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
	defer func() {
		dndListen(nil)
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
		case e := <-events:
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
			case dndEnd:
				// Wayland reports a drop outside the app and Escape as one
				// cancellation; both tear the tab off, as on X11.
				started = false
				drop := Drop{ID: d.id, Tab: d.tab, Slot: -1}
				if dropped != nil {
					drop = *dropped
				}
				s.registry.Emit(d.source, "tab:drop", drop)
				return
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

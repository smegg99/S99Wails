package windows

import (
	"context"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type nativeDrag struct {
	id, source, tab string
	cancel          chan struct{}
}

func init() {
	application.RegisterEvent[Drop]("tab:drop")
	application.RegisterEvent[*Drop]("tab:hover")
}

// Strip records client-relative rectangles, not native screen coordinates.
func (s *Service) Strip(ctx context.Context, strip Strip) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.strips == nil {
		s.strips = map[string]Strip{}
	}
	s.strips[CallingName(ctx)] = strip
}

// BeginDrag follows the native pointer through other webviews and the desktop.
// False means global coordinates are unavailable (native Wayland).
func (s *Service) BeginDrag(ctx context.Context, owner, id, tab string) (bool, error) {
	if err := s.owns(owner); err != nil {
		return false, err
	}
	var supported bool
	application.InvokeSync(func() { _, _, _, supported = Pointer() })
	if !supported {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	name := CallingName(ctx)
	if s.owner != owner || s.drag != nil || s.quitting || s.closeToken[name] != "" {
		return false, ErrBusy
	}
	drag := &nativeDrag{id: id, source: name, tab: tab, cancel: make(chan struct{})}
	s.drag = drag
	go s.followDrag(drag)
	return true, nil
}

func (s *Service) CancelDrag(ctx context.Context, id string) {
	s.cancelDrag(CallingName(ctx), id)
}
func (s *Service) cancelDrag(source, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d := s.drag; d != nil && (source == "" || source == d.source) && (id == "" || id == d.id) {
		close(d.cancel)
		s.drag = nil
	}
}

func (s *Service) followDrag(d *nativeDrag) {
	ticker := time.NewTicker(16 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(time.Minute)
	defer deadline.Stop()
	lastTarget := ""
	defer func() {
		if lastTarget != "" {
			s.registry.Emit(lastTarget, "tab:hover", nil)
		}
		s.mu.Lock()
		if s.drag == d {
			s.drag = nil
		}
		s.mu.Unlock()
	}()
	for {
		select {
		case <-d.cancel:
			return
		case <-deadline.C:
			s.registry.Emit(d.source, "tab:drag-end", d.id)
			return
		case <-ticker.C:
		}
		s.mu.Lock()
		native := make(map[string]*application.WebviewWindow, len(s.native))
		strips := make(map[string]Strip, len(s.strips))
		for name, win := range s.native {
			native[name] = win
			strips[name] = s.strips[name]
		}
		s.mu.Unlock()
		drop := Drop{ID: d.id, Tab: d.tab, Slot: -1}
		held, supported := false, false
		application.InvokeSync(func() {
			drop.X, drop.Y, held, supported = Pointer()
			if !supported {
				return
			}
			for name, win := range native {
				x, y, under := Under(win.NativeWindow())
				if !under {
					continue
				}
				source := ""
				if name == d.source {
					source = d.tab
				}
				if slot := strips[name].DropSlot(drop.X-x, drop.Y-y, source); slot >= 0 {
					drop.Target, drop.Slot = name, slot
				}
				break
			}
		})
		select {
		case <-d.cancel:
			return
		default:
		}
		if !supported {
			return
		}
		if drop.Target != lastTarget && lastTarget != "" {
			s.registry.Emit(lastTarget, "tab:hover", nil)
		}
		lastTarget = drop.Target
		if !held {
			s.registry.Emit(d.source, "tab:drop", drop)
			return
		}
		if drop.Target != "" && drop.Target != d.source {
			s.registry.Emit(drop.Target, "tab:hover", drop)
		}
	}
}

func (s *Service) TearOff(ctx context.Context, owner, tab, payload string, x, y int, positioned bool) (Offer, error) {
	var geometry *Geometry
	if positioned {
		geometry = &Geometry{X: x - 100, Y: y - 20, Width: s.options.Width, Height: s.options.Height}
		for _, screen := range application.Get().Screen.GetAll() {
			area := screen.WorkArea
			if x >= area.X && y >= area.Y && x < area.X+area.Width && y < area.Y+area.Height {
				geometry.Monitor = screen.ID
				break
			}
		}
	}
	return s.move(ctx, owner, "", tab, -1, payload, geometry)
}

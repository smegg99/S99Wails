// shell/windows/servicedrag.go

package windows

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type nativeDrag struct {
	id, source, tab string
	// face is the page's opaque tab description, passed on with every hover.
	face   string
	cancel chan struct{}
	// image is the page's picture of the dragged tab; version counts its
	// arrivals, so a follower applies each one once. Both under Service.mu.
	image   *dragImage
	version int
}

// dragImage is a PNG at scale device pixels per logical pixel, held at X, Y from its top left.
type dragImage struct {
	png   []byte
	scale int
	x, y  int
}

// Track tells the source window whether the pointer is outside or back over it at X, Y in its own coordinates.
type Track struct {
	Inside bool `json:"inside"`
	X      int  `json:"x"`
	Y      int  `json:"y"`
}

func init() {
	application.RegisterEvent[Drop](eventTabDrop)
	application.RegisterEvent[*Drop](eventTabHover)
	application.RegisterEvent[Track](eventTabTrack)
}

// DragImage hands the native drag its picture outside the source: a base64 PNG at scale, held at x, y.
func (s *Service) DragImage(ctx context.Context, id, png string, scale, x, y int) error {
	return s.dragImage(CallingName(ctx), id, png, scale, x, y)
}

func (s *Service) dragImage(source, id, png string, scale, x, y int) error {
	data, err := base64.StdEncoding.DecodeString(png)
	if err != nil || len(data) == 0 {
		return ErrTransfer
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	d := s.drag
	if d == nil || d.id != id || d.source != source {
		return nil
	}
	d.image = &dragImage{png: data, scale: max(scale, 1), x: x, y: y}
	d.version++
	return nil
}

// pendingImage returns the drag's picture when it changed since version seen.
func (s *Service) pendingImage(d *nativeDrag, seen int) (*dragImage, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d.version == seen {
		return nil, seen
	}
	return d.image, d.version
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

// BeginDrag follows the pointer across windows (global pointer on X11, GTK dnd on Wayland); false means neither is available.
func (s *Service) BeginDrag(ctx context.Context, owner, id, tab, face string) (bool, error) {
	if err := s.owns(owner); err != nil {
		return false, err
	}
	var supported, wayland bool
	application.InvokeSync(func() {
		_, _, _, supported = Pointer()
		wayland = !supported && dndWayland()
	})
	if !supported && !wayland {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	name := CallingName(ctx)
	if s.owner != owner || s.drag != nil || s.quitting || s.closeToken[name] != "" {
		return false, ErrBusy
	}
	drag := &nativeDrag{id: id, source: name, tab: tab, face: face, cancel: make(chan struct{})}
	s.drag = drag
	if wayland {
		go s.followWayland(drag)
	} else {
		go s.followDrag(drag)
	}
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
	seen := 0
	// The page follows the pointer over its own window; past it the floating
	// picture does, and the page is told to hide its own.
	overSource := true
	defer func() {
		application.InvokeSync(dndClear)
		if lastTarget != "" {
			s.registry.Emit(lastTarget, eventTabHover, nil)
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
			s.registry.Emit(d.source, eventTabDragEnd, d.id)
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
		held, supported, over := false, false, false
		var origin [2]int
		image, version := s.pendingImage(d, seen)
		application.InvokeSync(func() {
			if version != seen && image != nil {
				dndImage(image.png, image.scale, image.x, image.y)
			}
			drop.X, drop.Y, held, supported = Pointer()
			if !supported {
				return
			}
			if win := native[d.source]; win != nil {
				origin[0], origin[1], over = Under(win.NativeWindow())
			}
			dndFloat(held && !over, drop.X, drop.Y)
			for name, win := range native {
				x, y, under := Under(win.NativeWindow())
				if !under {
					continue
				}
				source := ""
				if name == d.source {
					source = d.tab
				}
				if slot := strips[name].dropSlot(drop.X-x, drop.Y-y, source); slot >= 0 {
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
		seen = version
		if !supported {
			return
		}
		if held && over != overSource {
			overSource = over
			s.registry.Emit(d.source, eventTabTrack, Track{Inside: over, X: drop.X - origin[0], Y: drop.Y - origin[1]})
		}
		if drop.Target != lastTarget && lastTarget != "" {
			s.registry.Emit(lastTarget, eventTabHover, nil)
		}
		lastTarget = drop.Target
		if !held {
			s.registry.Emit(d.source, eventTabDrop, drop)
			return
		}
		if drop.Target != "" && drop.Target != d.source {
			drop.Face = d.face
			s.registry.Emit(drop.Target, eventTabHover, drop)
		}
	}
}

func (s *Service) TearOff(ctx context.Context, owner, tab, payload string, x, y int, positioned bool) (Offer, error) {
	var geometry *Geometry
	// A Wayland client cannot place a window; the compositor does.
	var wayland bool
	application.InvokeSync(func() { wayland = dndWayland() })
	if positioned && !wayland {
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

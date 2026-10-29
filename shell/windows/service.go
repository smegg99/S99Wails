// shell/windows/service.go
package windows

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Service connects opaque tab transfers to the app's native workspaces.
type Service struct {
	registry    *Registry
	options     Options
	keep        func() error
	quit        func()
	mu          sync.Mutex
	movement    sync.Mutex
	restoring   sync.Mutex
	create      func(string, *Geometry) *application.WebviewWindow
	restored    string
	owner       string
	native      map[string]*application.WebviewWindow
	stopping    map[string]func()
	closeToken  map[string]string
	closeEpoch  map[string]uint64
	quitEpoch   uint64
	approved    map[string]bool
	drag        *nativeDrag
	strips      map[string]Strip
	quitting    bool
	quitReady   bool
	quitWaiting map[string]bool
	// handles name windows for the native Wayland drag (dnd.go).
	handles     map[string]uintptr
	handleNames map[uintptr]string
	nextHandle  uintptr
}

// Options inject application persistence and account identity. The library
// never reads credentials, application documents or layout payloads.
type Options struct {
	Owner                              func() string
	Layouts                            func(owner string) []SavedWindow
	Drop                               func(owner, name string) error
	Keep                               func() error
	Denied                             error
	BlockedKey                         string
	Width, Height, MinWidth, MinHeight int
}
type SavedWindow struct {
	Name     string
	Geometry *Geometry
}

func NewService(options Options) *Service {
	s := &Service{registry: New(), options: options, keep: options.Keep,
		quit:   func() { application.Get().Quit() },
		native: map[string]*application.WebviewWindow{}, stopping: map[string]func(){},
		closeToken: map[string]string{}, closeEpoch: map[string]uint64{}, approved: map[string]bool{}}
	if s.keep == nil {
		s.keep = func() error { return nil }
	}
	if s.options.Denied == nil {
		s.options.Denied = errors.New("windows: owner changed")
	}
	SetOwner(s, options.Owner())
	return s
}

// SetOwner invalidates previous-owner readiness, drags and close replies.
func SetOwner(s *Service, owner string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owner == owner {
		return
	}
	s.owner, s.restored = owner, ""
	if s.drag != nil {
		close(s.drag.cancel)
		s.drag = nil
	}
	s.strips = map[string]Strip{}
	s.closeToken, s.closeEpoch = map[string]string{}, map[string]uint64{}
	s.quitting, s.quitReady = false, false
	s.quitWaiting = nil
	s.registry.Reset()
}

func (s *Service) ServiceName() string { return "windows" }

// Configure supplies the native factory after application.New.
func Configure(s *Service, create func(string, *Geometry) *application.WebviewWindow) {
	s.create = create
}

// RegistryOf exposes native shell operations without binding them to JS.
func RegistryOf(s *Service) *Registry { return s.registry }

type nativeSurface struct{ win *application.WebviewWindow }

func (s nativeSurface) Name() string  { return s.win.Name() }
func (s nativeSurface) Show()         { s.win.Show() }
func (s nativeSurface) Hide()         { s.win.Hide() }
func (s nativeSurface) Focus()        { s.win.Focus() }
func (s nativeSurface) Focused() bool { return s.win.IsFocused() }
func (s nativeSurface) Visible() bool { return s.win.IsVisible() }
func (s nativeSurface) Emit(name string, data any) {
	s.win.DispatchWailsEvent(&application.CustomEvent{Name: name, Data: data})
}
func (s nativeSurface) Close() { s.win.Close() }

// Register installs the close handshake before the window is first shown.
func Register(s *Service, win *application.WebviewWindow, stop func(), closeToTray func() bool) {
	_ = s.registry.Add(nativeSurface{win})
	s.mu.Lock()
	s.native[win.Name()], s.stopping[win.Name()] = win, stop
	s.approved[win.Name()] = false
	s.mu.Unlock()
	win.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		s.mu.Lock()
		allowed := s.approved[win.Name()]
		s.mu.Unlock()
		if allowed {
			return
		}
		event.Cancel()
		if win.Name() == "main" {
			if closeToTray() {
				s.registry.hideAll()
			} else {
				RequestQuit(s)
			}
			return
		}
		s.requestClose(win.Name())
	})
}

func (s *Service) Ready(ctx context.Context, owner, label string, workspace bool) ([]Offer, error) {
	s.restoring.Lock()
	defer s.restoring.Unlock()
	name := CallingName(ctx)
	if !workspace {
		return []Offer{}, nil
	}
	currentOwner := s.options.Owner()
	if err := s.owns(owner); err != nil {
		return nil, err
	}
	if currentOwner != "" && name == "main" {
		s.mu.Lock()
		restore := s.restored != owner
		s.restored = owner
		s.mu.Unlock()
		if restore {
			// Secondary windows belong to the previous workspace until this
			// point. Close them without deleting that account's saved layout.
			for _, window := range s.registry.List() {
				if window.Name == "main" {
					continue
				}
				s.mu.Lock()
				win, stop := s.native[window.Name], s.stopping[window.Name]
				s.approved[window.Name] = true
				delete(s.native, window.Name)
				delete(s.stopping, window.Name)
				s.mu.Unlock()
				_ = s.registry.Remove(window.Name)
				if stop != nil {
					stop()
				}
				if win != nil {
					win.Close()
				}
			}
			for _, saved := range s.options.Layouts(owner) {
				if saved.Name == "main" {
					s.mu.Lock()
					main := s.native["main"]
					s.mu.Unlock()
					if main != nil && saved.Geometry != nil {
						PlaceWindow(main, saved.Geometry, s.options.MinWidth, s.options.MinHeight)
					}
					continue
				}
				s.mu.Lock()
				exists := s.native[saved.Name] != nil
				s.mu.Unlock()
				if !exists {
					s.create(saved.Name, saved.Geometry)
				}
			}
		}
	}
	s.mu.Lock()
	if s.owner != owner {
		s.mu.Unlock()
		return nil, s.options.Denied
	}
	pending, err := s.registry.Ready(name, label)
	s.mu.Unlock()
	s.broadcast()
	return pending, err
}
func (s *Service) Name(ctx context.Context) string { return CallingName(ctx) }

func (s *Service) List(context.Context) []Info { return s.registry.List() }

func (s *Service) Move(ctx context.Context, owner, target, tab string, slot int, payload string) (Offer, error) {
	return s.move(ctx, owner, target, tab, slot, payload, nil)
}

func (s *Service) move(ctx context.Context, owner, target, tab string, slot int, payload string, geometry *Geometry) (Offer, error) {
	s.movement.Lock()
	defer s.movement.Unlock()
	s.mu.Lock()
	blocked := s.quitting || s.closeToken[CallingName(ctx)] != "" || s.closeToken[target] != ""
	s.mu.Unlock()
	if blocked {
		return Offer{}, ErrBusy
	}
	if err := s.owns(owner); err != nil {
		return Offer{}, err
	}
	if target == "" {
		target = Name()
		s.create(target, geometry)
	}
	s.mu.Lock()
	if s.owner != owner {
		s.mu.Unlock()
		return Offer{}, s.options.Denied
	}
	offer, err := s.registry.Move(CallingName(ctx), target, tab, slot, json.RawMessage(payload))
	s.mu.Unlock()
	if err == nil {
		s.registry.Focus(target)
		s.broadcast()
	}
	return offer, err
}
func (s *Service) Claim(ctx context.Context, owner, id string) error {
	if err := s.owns(owner); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owner != owner {
		return s.options.Denied
	}
	return s.registry.Claim(CallingName(ctx), id)
}
func (s *Service) Released(ctx context.Context, id string) error {
	return s.registry.Released(CallingName(ctx), id)
}
func (s *Service) Cancel(ctx context.Context, id string) error {
	return s.registry.Cancel(CallingName(ctx), id)
}

func (s *Service) owns(owner string) error {
	currentOwner := s.options.Owner()
	if currentOwner == "" || owner != currentOwner {
		return s.options.Denied
	}
	return nil
}
func (s *Service) broadcast() {
	for _, window := range s.registry.List() {
		s.registry.Emit(window.Name, eventWindowsChanged, s.registry.List())
	}
}

// CallingName resolves the invoking webview; unbound calls address main.
func CallingName(ctx context.Context) string {
	if win, ok := ctx.Value(application.WindowKey).(application.Window); ok {
		return win.Name()
	}
	return "main"
}

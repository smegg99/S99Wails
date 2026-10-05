// services/title/title.go

// Package title sets each window title to an app name plus a translated phrase from the frontend.
package title

import (
	"context"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// DefaultSep joins the base name and the phrase.
const DefaultSep = " - "

// Options configures a Service.
type Options struct {
	// Base leads the title. Empty uses the Wails application name, resolved
	// lazily because the service is built before application.New.
	Base string
	// Sep joins Base and the phrase. Empty means DefaultSep.
	Sep string
}

// Service methods are safe for concurrent use.
type Service struct {
	mu     sync.Mutex
	base   string
	sep    string
	states map[uint]*windowTitle

	fallbackBase string

	// applyMu serializes apply, a blocking main-thread call that mu must not
	// span. The holder re-reads state so pushes land in order.
	applyMu sync.Mutex

	apply    func(context.Context, string) // test seam
	windowID func(context.Context) uint
}

type windowTitle struct {
	phrase  string
	applied string
	dirty   bool
}

func New(opts Options) *Service {
	s := &Service{base: opts.Base, sep: opts.Sep, states: map[uint]*windowTitle{}}
	if s.sep == "" {
		s.sep = DefaultSep
	}
	s.windowID = func(ctx context.Context) uint {
		if win, ok := ctx.Value(application.WindowKey).(application.Window); ok {
			return win.ID()
		}
		return 0
	}
	s.apply = func(ctx context.Context, text string) {
		if win, ok := ctx.Value(application.WindowKey).(application.Window); ok {
			win.SetTitle(text)
		}
	}
	return s
}

func (s *Service) stateLocked(id uint) *windowTitle {
	if s.states[id] == nil {
		s.states[id] = &windowTitle{}
	}
	return s.states[id]
}

// Set replaces the phrase after the base name; empty shows the base alone. The
// error is always nil and exists for the generated binding.
func (s *Service) Set(ctx context.Context, text string) error {
	s.mu.Lock()
	id := s.windowID(ctx)
	s.stateLocked(id).phrase = text
	s.mu.Unlock()

	s.applyMu.Lock()
	defer s.applyMu.Unlock()

	s.mu.Lock()
	state := s.stateLocked(id)
	resolved := s.resolveLocked(state)
	if state.dirty && resolved == state.applied {
		s.mu.Unlock()
		return nil
	}
	state.applied = resolved
	state.dirty = true
	s.mu.Unlock()

	s.apply(ctx, resolved)
	return nil
}

func (s *Service) Title(ctx context.Context) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.resolveLocked(s.stateLocked(s.windowID(ctx)))
}

func (s *Service) resolveLocked(state *windowTitle) string {
	base := s.resolveBaseLocked()
	switch {
	case state.phrase == "":
		return base
	case base == "":
		return state.phrase
	default:
		return base + s.sep + state.phrase
	}
}

// resolveBaseLocked retries the fallback until it is non-empty, then caches it.
func (s *Service) resolveBaseLocked() string {
	if s.base != "" {
		return s.base
	}
	if s.fallbackBase == "" {
		s.fallbackBase = appNameFn()
	}
	return s.fallbackBase
}

func appName() string {
	app := application.Get()
	if app == nil {
		return ""
	}
	return app.Config().Name
}

// appNameFn is a test seam.
var appNameFn = appName

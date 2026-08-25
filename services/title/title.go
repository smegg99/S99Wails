// services/title/title.go

// Package title sets each window title to an app name plus a translated phrase from the frontend.
package title

import (
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
	// Window resolves the window whose title is set. A provider rather than a
	// window because the window is built from the app, and the app is built
	// with its services. Nil, or returning nil, makes the service inert.
	Window func() *application.WebviewWindow
}

// Service methods are safe for concurrent use.
type Service struct {
	mu      sync.Mutex
	base    string
	sep     string
	phrase  string
	applied string
	dirty   bool // applied means nothing until the first push

	fallbackBase string

	// applyMu serializes apply, a blocking main-thread call that mu must not
	// span. The holder re-reads state so pushes land in order.
	applyMu sync.Mutex

	apply func(string) // test seam
}

func New(opts Options) *Service {
	s := &Service{base: opts.Base, sep: opts.Sep}
	if s.sep == "" {
		s.sep = DefaultSep
	}

	s.apply = func(text string) {
		if opts.Window == nil {
			return
		}
		if win := opts.Window(); win != nil {
			win.SetTitle(text)
		}
	}

	return s
}

// Set replaces the phrase after the base name; empty shows the base alone. The
// error is always nil and exists for the generated binding.
func (s *Service) Set(text string) error {
	s.mu.Lock()
	s.phrase = text
	s.mu.Unlock()

	s.applyMu.Lock()
	defer s.applyMu.Unlock()

	s.mu.Lock()
	resolved := s.resolveLocked()
	if s.dirty && resolved == s.applied {
		s.mu.Unlock()
		return nil
	}
	s.applied = resolved
	s.dirty = true
	s.mu.Unlock()

	s.apply(resolved)
	return nil
}

func (s *Service) Title() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.resolveLocked()
}

func (s *Service) resolveLocked() string {
	base := s.resolveBaseLocked()
	switch {
	case s.phrase == "":
		return base
	case base == "":
		return s.phrase
	default:
		return base + s.sep + s.phrase
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

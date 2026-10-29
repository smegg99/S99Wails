// shell/crash/show.go

package crash

import (
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Show opens the screen for f in this process and returns once it is closed. Call it only when no other
// Wails application was created.
func Show(opts Options, f Failure) error {
	svc := newService(opts, build(opts, f, described(), local.snapshot(), time.Now()), controls{})
	app, window := newApp(opts, svc)
	window.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) { app.Quit() })
	return app.Run()
}

// newApp is the screen's application: the app's own options with only the crash service bound.
func newApp(opts Options, svc *Service) (*application.App, *application.WebviewWindow) {
	options := opts.App
	options.Services = []application.Service{application.NewService(svc)}
	options.SingleInstance = nil
	options.Mac.ApplicationShouldTerminateAfterLastWindowClosed = true
	app := application.New(options)
	width, height := opts.Width, opts.Height
	if width == 0 || height == 0 {
		width, height = 960, 640
	}
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: options.Name, Width: width, Height: height, MinWidth: width, MinHeight: height, URL: "/?" + Query,
	})
	return app, window
}

// wailsScreen is the watcher's screen. The loop calls show, hide and quit; run owns the main goroutine.
type wailsScreen struct {
	opts  Options
	ctl   controls
	open  chan Report
	ready chan struct{}
	// opened is read and written only by the loop goroutine.
	opened bool
	app    *application.App
	window *application.WebviewWindow
	svc    *Service
}

func newWailsScreen(opts Options) *wailsScreen {
	return &wailsScreen{opts: opts, ctl: newControls(), open: make(chan Report), ready: make(chan struct{})}
}

func (s *wailsScreen) show(r Report) {
	if !s.opened {
		s.opened = true
		s.open <- r
		<-s.ready
		return
	}
	s.svc.set(r)
	s.app.Event.Emit(reportEvent, r)
	s.window.Show()
	s.window.Focus()
}

func (s *wailsScreen) hide() {
	if s.opened {
		s.window.Hide()
	}
}

func (s *wailsScreen) quit() {
	if s.opened {
		s.app.Quit()
	}
}

// run starts the screen with first and blocks until it quits. Closing the window while the process
// still runs means Wait.
func (s *wailsScreen) run(first Report) error {
	s.svc = newService(s.opts, first, s.ctl)
	s.app, s.window = newApp(s.opts, s.svc)
	s.window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if s.svc.Report().Alive {
			e.Cancel()
			s.window.Hide()
			s.ctl.signal(s.ctl.wait)
			return
		}
		s.app.Quit()
	})
	close(s.ready)
	return s.app.Run()
}

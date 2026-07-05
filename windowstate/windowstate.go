// windowstate.go
//
// Package windowstate persists window position, size and maximised state through a Store.
package windowstate

import (
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// State is a saved window geometry.
type State struct {
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximised bool `json:"maximised"`
}

// Store loads and saves a State. Load returns ok=false when nothing has been
// saved yet.
type Store interface {
	Load() (state State, ok bool, err error)
	Save(State) error
}

// Window is the subset of *application.WebviewWindow this package needs.
// It exists so the tracking logic is unit-testable with a fake.
type Window interface {
	Position() (int, int)
	Size() (int, int)
	IsMaximised() bool
	SetPosition(x, y int)
	SetSize(width, height int) application.Window
	Maximise() application.Window
	OnWindowEvent(eventType events.WindowEventType, callback func(event *application.WindowEvent)) func()
}

// Options configures Track.
type Options struct {
	// Debounce delays saving after the last move or resize. Defaults to 500ms.
	Debounce time.Duration
	// OnError receives Store errors. Optional; errors are dropped when nil.
	OnError func(error)
}

// Restore applies a previously saved state to the window. Call it after
// creating the window and before showing it. Restoring an unsaved store is a
// no-op. Saved coordinates are applied verbatim; if the monitor layout
// changed since the save, the window manager's placement rules take over.
func Restore(win Window, store Store) error {
	state, ok, err := store.Load()
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	if state.Width > 0 && state.Height > 0 {
		win.SetSize(state.Width, state.Height)
	}
	win.SetPosition(state.X, state.Y)
	if state.Maximised {
		win.Maximise()
	}
	return nil
}

// Track saves the window geometry on move/resize (debounced) and on
// maximise/unmaximise and close. While the window is maximised the last
// un-maximised geometry is kept, so restoring later yields a sensible size
// after un-maximising. Geometry is read inside the window event callbacks
// (the GUI thread); the debounce timer only persists the already-captured
// state. Returns a stop function that unregisters the listeners and flushes
// a final save; it is safe to call more than once.
func Track(win Window, store Store, opts Options) (stop func()) {
	if opts.Debounce <= 0 {
		opts.Debounce = 500 * time.Millisecond
	}

	t := &tracker{win: win, store: store, opts: opts}

	// Seed from the current geometry so a maximise before any move still has
	// an un-maximised size to fall back to. Track is called from the app's
	// setup path, which is main-thread territory.
	t.capture()

	unregister := []func(){
		win.OnWindowEvent(events.Common.WindowDidMove, func(_ *application.WindowEvent) { t.captureAndSchedule() }),
		win.OnWindowEvent(events.Common.WindowDidResize, func(_ *application.WindowEvent) { t.captureAndSchedule() }),
		win.OnWindowEvent(events.Common.WindowMaximise, func(_ *application.WindowEvent) { t.captureAndPersist() }),
		win.OnWindowEvent(events.Common.WindowUnMaximise, func(_ *application.WindowEvent) { t.captureAndPersist() }),
		win.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) { t.captureAndPersist() }),
	}

	var once sync.Once
	return func() {
		once.Do(func() {
			for _, u := range unregister {
				u()
			}
			t.mu.Lock()
			t.stopped = true
			if t.timer != nil {
				t.timer.Stop()
			}
			t.mu.Unlock()
			// Final flush; persist serializes with a timer callback that
			// slipped past Stop.
			t.persist(true)
		})
	}
}

// Manage restores the saved state and starts tracking. The returned stop
// function is Track's.
func Manage(win Window, store Store, opts Options) (stop func(), err error) {
	err = Restore(win, store)
	stop = Track(win, store, opts)
	return stop, err
}

type tracker struct {
	win   Window
	store Store
	opts  Options

	mu      sync.Mutex
	last    State
	timer   *time.Timer
	stopped bool

	// saveMu serializes Store.Save with the final stop flush.
	saveMu sync.Mutex
}

// capture refreshes the pending state from the window. It must run on the
// GUI thread (window event callbacks or setup). Geometry is only taken while
// un-maximised; the maximised flag always reflects the window.
func (t *tracker) capture() {
	maximised := t.win.IsMaximised()

	t.mu.Lock()
	defer t.mu.Unlock()
	if !maximised {
		x, y := t.win.Position()
		w, h := t.win.Size()
		t.last = State{X: x, Y: y, Width: w, Height: h}
	}
	t.last.Maximised = maximised
}

func (t *tracker) captureAndSchedule() {
	t.capture()
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped {
		return
	}
	if t.timer != nil {
		t.timer.Stop()
	}
	t.timer = time.AfterFunc(t.opts.Debounce, func() { t.persist(false) })
}

func (t *tracker) captureAndPersist() {
	t.capture()
	t.persist(false)
}

// persist writes the last captured state. final bypasses the stopped check
// so the stop flush still runs after stopped is set.
func (t *tracker) persist(final bool) {
	t.mu.Lock()
	if t.stopped && !final {
		t.mu.Unlock()
		return
	}
	state := t.last
	t.mu.Unlock()

	t.saveMu.Lock()
	defer t.saveMu.Unlock()
	if err := t.store.Save(state); err != nil && t.opts.OnError != nil {
		t.opts.OnError(err)
	}
}

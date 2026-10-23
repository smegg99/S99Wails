// shell/windowstate/windowstate_test.go

package windowstate

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type fakeWindow struct {
	x, y, w, h int
	maximised  bool
	handlers   map[events.WindowEventType][]func(*application.WindowEvent)
	hooks      map[events.WindowEventType][]func(*application.WindowEvent)
}

func newFakeWindow() *fakeWindow {
	return &fakeWindow{x: 10, y: 20, w: 800, h: 600, handlers: map[events.WindowEventType][]func(*application.WindowEvent){}, hooks: map[events.WindowEventType][]func(*application.WindowEvent){}}
}

func (f *fakeWindow) Position() (int, int) { return f.x, f.y }
func (f *fakeWindow) Size() (int, int)     { return f.w, f.h }
func (f *fakeWindow) IsMaximised() bool    { return f.maximised }
func (f *fakeWindow) SetPosition(x, y int) { f.x, f.y = x, y }
func (f *fakeWindow) SetSize(w, h int) application.Window {
	f.w, f.h = w, h
	return nil
}
func (f *fakeWindow) Maximise() application.Window {
	f.maximised = true
	return nil
}

func (f *fakeWindow) OnWindowEvent(et events.WindowEventType, cb func(*application.WindowEvent)) func() {
	f.handlers[et] = append(f.handlers[et], cb)
	return func() { f.handlers[et] = nil }
}

func (f *fakeWindow) RegisterHook(et events.WindowEventType, cb func(*application.WindowEvent)) func() {
	f.hooks[et] = append(f.hooks[et], cb)
	return func() { f.hooks[et] = nil }
}

func (f *fakeWindow) fire(et events.WindowEventType) {
	for _, cb := range f.hooks[et] {
		if cb != nil {
			cb(nil)
		}
	}
	for _, cb := range f.handlers[et] {
		if cb != nil {
			cb(nil)
		}
	}
}

type memStore struct {
	mu       sync.Mutex
	state    State
	ok       bool
	saves    int
	failWith error
}

func (s *memStore) Load() (State, bool, error) { return s.state, s.ok, nil }
func (s *memStore) Save(state State) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failWith != nil {
		return s.failWith
	}
	s.state = state
	s.ok = true
	s.saves++
	return nil
}

func (s *memStore) snapshot() (State, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state, s.saves
}

func TestRestoreAppliesSavedStateOnShow(t *testing.T) {
	win := newFakeWindow()
	store := &memStore{state: State{X: 100, Y: 50, Width: 1024, Height: 768, Maximised: true}, ok: true}

	if err := Restore(win, store); err != nil {
		t.Fatal(err)
	}
	// Nothing may touch the window before it enters its run loop: geometry
	// calls dispatch to the GUI main thread, which does not exist yet.
	if win.x != 10 || win.w != 800 || win.maximised {
		t.Errorf("state applied before WindowShow: %+v", win)
	}

	win.fire(events.Common.WindowShow)
	if win.x != 100 || win.y != 50 || win.w != 1024 || win.h != 768 || !win.maximised {
		t.Errorf("state not applied on WindowShow: %+v", win)
	}

	// A second show must not re-apply (one-shot).
	win.x = 1
	win.fire(events.Common.WindowShow)
	if win.x != 1 {
		t.Error("restore re-applied on second WindowShow")
	}
}

func TestConfigureAppliesSavedState(t *testing.T) {
	store := &memStore{state: State{X: 7, Y: 8, Width: 640, Height: 480, Maximised: true}, ok: true}
	opts := application.WebviewWindowOptions{Width: 1280, Height: 800}

	applied, err := Configure(&opts, store)
	if err != nil {
		t.Fatal(err)
	}
	if !applied {
		t.Error("Configure reported no saved state")
	}
	if opts.X != 7 || opts.Y != 8 || opts.Width != 640 || opts.Height != 480 {
		t.Errorf("options not configured: %+v", opts)
	}
	if opts.InitialPosition != application.WindowXY {
		t.Error("saved coordinates left at the centred default")
	}
	if opts.StartState != application.WindowStateMaximised {
		t.Error("maximised state not configured")
	}
}

func TestConfigureUnsavedStoreIsNoop(t *testing.T) {
	opts := application.WebviewWindowOptions{Width: 1280, Height: 800}
	applied, err := Configure(&opts, &memStore{})
	if err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Error("Configure reported a saved state there was none of")
	}
	if opts.Width != 1280 || opts.Height != 800 || opts.X != 0 {
		t.Errorf("options changed without saved state: %+v", opts)
	}
	// WindowCentered is the zero value, so a first run must not be pinned to 0,0.
	if opts.InitialPosition != application.WindowCentered {
		t.Error("unsaved store still asked for explicit coordinates")
	}
}

func TestRestoreNoSavedStateIsNoop(t *testing.T) {
	win := newFakeWindow()
	if err := Restore(win, &memStore{}); err != nil {
		t.Fatal(err)
	}
	if win.x != 10 || win.w != 800 {
		t.Errorf("window changed without saved state: %+v", win)
	}
}

func TestTrackDebouncedSaveOnMove(t *testing.T) {
	win := newFakeWindow()
	store := &memStore{}
	stop := Track(win, store, Options{Debounce: 20 * time.Millisecond})
	defer stop()

	win.x = 300
	win.fire(events.Common.WindowDidMove)
	if _, saves := store.snapshot(); saves != 0 {
		t.Error("save should be debounced, not immediate")
	}

	time.Sleep(60 * time.Millisecond)
	state, saves := store.snapshot()
	if saves != 1 || state.X != 300 {
		t.Errorf("debounced save missing: saves=%d state=%+v", saves, state)
	}
}

func TestTrackKeepsUnmaximisedGeometry(t *testing.T) {
	win := newFakeWindow()
	store := &memStore{}
	stop := Track(win, store, Options{Debounce: 10 * time.Millisecond})

	win.maximised = true
	win.w, win.h = 1920, 1080 // maximised geometry must not be captured
	win.fire(events.Common.WindowMaximise)

	stop()
	state, _ := store.snapshot()
	if !state.Maximised {
		t.Error("maximised flag lost")
	}
	if state.Width != 800 || state.Height != 600 {
		t.Errorf("un-maximised geometry lost: %+v", state)
	}
}

func TestStopFlushesAndSilencesLateTimer(t *testing.T) {
	win := newFakeWindow()
	store := &memStore{}
	stop := Track(win, store, Options{Debounce: 15 * time.Millisecond})

	win.x = 42
	win.fire(events.Common.WindowDidMove)
	stop()
	stop() // idempotent

	_, savesAfterStop := store.snapshot()
	if savesAfterStop != 1 {
		t.Errorf("stop should flush exactly once, got %d saves", savesAfterStop)
	}

	time.Sleep(40 * time.Millisecond)
	if _, saves := store.snapshot(); saves != savesAfterStop {
		t.Errorf("late timer saved after stop: %d -> %d", savesAfterStop, saves)
	}

	state, _ := store.snapshot()
	if state.X != 42 {
		t.Errorf("final flush missing captured state: %+v", state)
	}
}

func TestTrackReportsStoreErrors(t *testing.T) {
	win := newFakeWindow()
	sentinel := errors.New("disk full")
	var got error
	stop := Track(win, &memStore{failWith: sentinel}, Options{
		Debounce: time.Millisecond,
		OnError:  func(err error) { got = err },
	})
	stop() // flush triggers Save -> error
	if !errors.Is(got, sentinel) {
		t.Errorf("OnError not called with store error, got %v", got)
	}
}

// preShowWindow panics on geometry setters before show, as a real window
// crashes before app.Run starts the main thread.
type preShowWindow struct {
	*fakeWindow
	shown bool
}

func (p *preShowWindow) mutate(what string) {
	if !p.shown {
		panic("windowstate: " + what + " before the window was shown")
	}
}

func (p *preShowWindow) SetSize(w, h int) application.Window {
	p.mutate("SetSize")
	return p.fakeWindow.SetSize(w, h)
}

func (p *preShowWindow) SetPosition(x, y int) {
	p.mutate("SetPosition")
	p.fakeWindow.SetPosition(x, y)
}

func (p *preShowWindow) Maximise() application.Window {
	p.mutate("Maximise")
	return p.fakeWindow.Maximise()
}

func (p *preShowWindow) show() {
	p.shown = true
	p.fire(events.Common.WindowShow)
}

func TestManageDoesNotTouchGeometryBeforeShow(t *testing.T) {
	store := &memStore{state: State{X: 5, Y: 6, Width: 1024, Height: 768, Maximised: true}, ok: true}
	win := &preShowWindow{fakeWindow: newFakeWindow()}

	stop, err := Manage(win, store, Options{})
	if err != nil {
		t.Fatalf("Manage: %v", err)
	}
	defer stop()

	win.show()

	if win.w != 1024 || win.h != 768 || win.x != 5 || win.y != 6 || !win.maximised {
		t.Errorf("state not applied on show: %+v", win.fakeWindow)
	}
}

// Wails runs its destruction listener concurrently with other closing listeners.
func TestCloseHookCapturesBeforeNativeWindowDisappears(t *testing.T) {
	win := newFakeWindow()
	store := &memStore{}
	win.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		win.x, win.y, win.w, win.h = 0, 0, 0, 0
	})
	stop := Track(win, store, Options{})
	// A resize whose debounced notification has not arrived yet must still survive close.
	win.x, win.y, win.w, win.h = 50, 60, 1000, 680
	win.fire(events.Common.WindowClosing)
	win.fire(events.Common.WindowDidResize) // queued native event after destruction
	stop()
	got, _ := store.snapshot()
	want := State{X: 50, Y: 60, Width: 1000, Height: 680}
	if got != want {
		t.Fatalf("closed geometry = %+v, want %+v", got, want)
	}
}

func TestTrackWithoutLiveGeometryPreservesSavedRow(t *testing.T) {
	for _, maximised := range []bool{false, true} {
		win := newFakeWindow()
		win.w, win.h, win.maximised = 0, 0, maximised
		want := State{X: 20, Y: 30, Width: 1000, Height: 680, Maximised: maximised}
		store := &memStore{state: want, ok: true}
		stop := Track(win, store, Options{})
		stop()
		if got, saves := store.snapshot(); got != want {
			t.Fatalf("unavailable geometry overwrote saved row: %+v saves=%d", got, saves)
		}
	}
}

func TestManageRestoredSizeSurvivesCloseWithoutResizeEvent(t *testing.T) {
	win := newFakeWindow()
	win.w, win.h = 0, 0 // no native window before Run
	want := State{X: 50, Y: 60, Width: 1000, Height: 680}
	store := &memStore{state: want, ok: true}
	win.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		win.x, win.y, win.w, win.h = 0, 0, 0, 0
	})
	stop, err := Manage(win, store, Options{})
	if err != nil {
		t.Fatal(err)
	}
	win.w, win.h = 1100, 720 // created with app defaults
	win.fire(events.Common.WindowShow)
	win.fire(events.Common.WindowClosing)
	stop()
	if got, _ := store.snapshot(); got != want {
		t.Fatalf("restored geometry lost: %+v", got)
	}
}

func TestTrackSeedsNormalGeometryBeforeFirstMaximise(t *testing.T) {
	win := newFakeWindow()
	win.w, win.h = 0, 0 // tracking starts before the native window exists
	want := State{X: 50, Y: 60, Width: 1000, Height: 680}
	store := &memStore{state: want, ok: true}
	stop := Track(win, store, Options{})
	win.w, win.h, win.maximised = 1920, 1080, true
	win.fire(events.Common.WindowMaximise) // no earlier resize sample
	stop()
	want.Maximised = true
	if got, _ := store.snapshot(); got != want {
		t.Fatalf("maximise lost saved normal geometry/flag: %+v, want %+v", got, want)
	}
}

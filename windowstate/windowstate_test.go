// windowstate_test.go
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
}

func newFakeWindow() *fakeWindow {
	return &fakeWindow{x: 10, y: 20, w: 800, h: 600, handlers: map[events.WindowEventType][]func(*application.WindowEvent){}}
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

func (f *fakeWindow) fire(et events.WindowEventType) {
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

func TestRestoreAppliesSavedState(t *testing.T) {
	win := newFakeWindow()
	store := &memStore{state: State{X: 100, Y: 50, Width: 1024, Height: 768, Maximised: true}, ok: true}

	if err := Restore(win, store); err != nil {
		t.Fatal(err)
	}
	if win.x != 100 || win.y != 50 || win.w != 1024 || win.h != 768 || !win.maximised {
		t.Errorf("state not applied: %+v", win)
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
	win.w, win.h = 1920, 1080 // maximised geometry must NOT be captured
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

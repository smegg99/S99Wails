// windowstate/example_test.go

package windowstate

import (
	"fmt"
	"time"

	"github.com/wailsapp/wails/v3/pkg/events"
)

// Example restores, tracks a move and flushes on stop, using the fake window.
func Example() {
	win := newFakeWindow()
	store := &memStore{state: State{X: 100, Y: 50, Width: 1024, Height: 768}, ok: true}

	stop, err := Manage(win, store, Options{Debounce: 5 * time.Millisecond})
	if err != nil {
		panic(err)
	}
	win.fire(events.Common.WindowShow) // restore applies when the window first shows
	fmt.Printf("restored: %dx%d at %d,%d\n", win.w, win.h, win.x, win.y)

	win.x, win.y = 300, 200
	win.fire(events.Common.WindowDidMove)
	stop() // flushes the captured geometry

	saved, _ := store.snapshot()
	fmt.Printf("saved: %dx%d at %d,%d\n", saved.Width, saved.Height, saved.X, saved.Y)
	// Output:
	// restored: 1024x768 at 100,50
	// saved: 1024x768 at 300,200
}

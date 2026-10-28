// shell/crash/beat.go

package crash

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	beatEvery    = 2 * time.Second
	sourceNative = "native"
	sourcePage   = "frontend"
)

// Beat tells the watcher the app is up, then beats from the main thread every 2 s until the app
// quits or stop is called. Call it after application.New, which runs the single-instance check.
func Beat() (stop func()) {
	ctx := context.Background()
	if app := application.Get(); app != nil {
		ctx = app.Context()
	}
	return beat(ctx)
}

func beat(ctx context.Context) (stop func()) {
	send(lineUp, "")
	var waiting atomic.Bool
	ticker := time.NewTicker(beatEvery)
	quit := make(chan struct{})
	// A quit's cleanup can outlast hangAfter, and the page never rests once it starts.
	stop = sync.OnceFunc(func() {
		ticker.Stop()
		close(quit)
		send(lineQuit, "")
	})
	go func() {
		for {
			select {
			case <-quit:
				return
			case <-ctx.Done():
				stop()
				return
			case <-ticker.C:
				if waiting.Swap(true) {
					continue
				}
				application.InvokeAsync(func() {
					send(lineBeat, sourceNative)
					waiting.Store(false)
				})
			}
		}
	}()
	return stop
}

// Heartbeat is the service the page beats through, by name; register it in the main app.
type Heartbeat struct{}

// Beat tells the watcher the page is alive.
func (Heartbeat) Beat() {
	send(lineBeat, sourcePage)
}

// Rest tells the watcher to stop expecting page beats, while the page is hidden.
func (Heartbeat) Rest() {
	send(lineRest, sourcePage)
}

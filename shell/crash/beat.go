// shell/crash/beat.go

package crash

import (
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

// Beat shows the watcher the main thread is alive: every 2 s a callback on it sends a beat, and a
// callback still waiting is never queued twice. Call it after application.New; stop ends it.
func Beat() (stop func()) {
	var waiting atomic.Bool
	ticker := time.NewTicker(beatEvery)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-quit:
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
	return sync.OnceFunc(func() {
		ticker.Stop()
		close(quit)
	})
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

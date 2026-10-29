// shell/windowstate/persist_test.go

package windowstate

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

// awaitQueuedTimer observes the actual mutex wait, without sleeps or production test hooks.
func awaitQueuedTimer(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	buf := make([]byte, 1<<20)
	for time.Now().Before(deadline) {
		n := runtime.Stack(buf, true)
		for _, stack := range strings.Split(string(buf[:n]), "\n\n") {
			if strings.Contains(stack, "queuedTimerSave(") && strings.Contains(stack, "(*tracker).persist(") && strings.Contains(stack, "sync.Mutex.Lock") {
				return
			}
		}
		runtime.Gosched()
	}
	t.Fatal("timer did not reach the serialized save")
}

func queuedTimerSave(tr *tracker, done chan<- struct{}) {
	defer close(done)
	tr.persist(false)
}

func TestQueuedTimerCannotSaveAfterStop(t *testing.T) {
	store := &memStore{}
	tr := &tracker{store: store, last: State{X: 10, Width: 1000, Height: 680}}
	tr.saveMu.Lock()
	done := make(chan struct{})
	go queuedTimerSave(tr, done)
	awaitQueuedTimer(t)
	tr.mu.Lock()
	tr.last = State{X: 42, Width: 1000, Height: 680}
	tr.stopped = true
	tr.mu.Unlock()
	tr.saveMu.Unlock()
	tr.persist(true)
	<-done
	state, saves := store.snapshot()
	if saves != 1 || state.X != 42 {
		t.Fatalf("queued timer wrote after stop: saves=%d state=%+v", saves, state)
	}
}

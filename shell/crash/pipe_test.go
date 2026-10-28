// shell/crash/pipe_test.go

package crash

import (
	"os"
	"strings"
	"testing"
	"time"
)

// openPipe routes send to a fresh OS pipe for one test.
func openPipe(t *testing.T) (r, w *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	out.Store(newPipe(w))
	t.Cleanup(func() {
		out.Store(nil)
		_ = r.Close()
		_ = w.Close()
	})
	return r, w
}

func TestDoneReturnsWhenTheWriterIsStuck(t *testing.T) {
	openPipe(t)
	for range 64 {
		send(lineLog, strings.Repeat("x", 3000))
	}
	done := make(chan struct{})
	go func() {
		Done()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Done blocked on a writer nobody reads")
	}
}

func TestBeatSendsUpAndRestsOnStop(t *testing.T) {
	r, w := openPipe(t)
	stop := Beat()
	stop()
	Done()
	_ = w.Close()
	events := make(chan event, 16)
	readPipe(r, events)
	close(events)
	var kinds []string
	for e := range events {
		kinds = append(kinds, e.kind+":"+e.payload)
	}
	got := strings.Join(kinds, ",")
	if got != "up:,rest:native,bye:,end:" {
		t.Fatalf("lines = %q, want up first and a native rest before the end", got)
	}
}

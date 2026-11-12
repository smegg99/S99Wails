// shell/crash/pipe_test.go

package crash

import (
	"context"
	"os"
	"runtime/debug"
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
	out.Store(newPipe(w, true))
	t.Cleanup(func() {
		out.Store(nil)
		_ = r.Close()
		_ = w.Close()
	})
	return r, w
}

// attachPipe routes send and the runtime's crash output to a fresh OS pipe, the way spawn does.
func attachPipe(t *testing.T) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := attach(w); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = debug.SetCrashOutput(nil, debug.CrashOptions{})
		out.Store(nil)
		_ = r.Close()
		_ = w.Close()
	})
}

// The runtime makes its crash output blocking, yet a pipe nobody reads still never holds a caller or the exit.
func TestSendAndDoneNeverBlockOnAFullCrashPipe(t *testing.T) {
	attachPipe(t)
	done := make(chan struct{})
	go func() {
		for range 400 {
			send(lineLog, strings.Repeat("x", 400))
		}
		Done()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a full pipe blocked send or Done")
	}
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

func TestBeatSendsUpAndQuitOnStop(t *testing.T) {
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
	if got != "up:,quit:,bye:,end:" {
		t.Fatalf("lines = %q, want up first and quit before the end", got)
	}
}

func TestBeatSendsQuitWhenTheAppContextEnds(t *testing.T) {
	r, _ := openPipe(t)
	events := make(chan event, 16)
	go readPipe(r, events)
	ctx, cancel := context.WithCancel(context.Background())
	stop := beat(ctx)
	defer stop()

	cancel()

	for _, want := range []string{lineUp, lineQuit} {
		select {
		case e := <-events:
			if e.kind != want {
				t.Fatalf("line %q, want %q", e.kind, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("no %q line after the app context ended", want)
		}
	}
}

// A panic's last write gives up on a watcher that stopped reading, so the process still exits.
func TestLastWriteGivesUpWhenNobodyReads(t *testing.T) {
	openPipe(t)
	for range 64 {
		send(lineLog, strings.Repeat("x", 3000))
	}
	p := out.Swap(nil)
	done := make(chan struct{})
	go func() {
		finish(p, strings.Repeat("panic: boom\n", 1000), 100*time.Millisecond)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("the last write blocked on a pipe nobody reads")
	}
}

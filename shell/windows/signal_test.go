package windows

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestQuitOnSignals(t *testing.T) {
	run := func(grace time.Duration, signals ...os.Signal) (bool, int, time.Duration) {
		quit := make(chan struct{})
		s := &Service{quit: func() { close(quit) }}
		ch := make(chan os.Signal, len(signals))
		for _, sig := range signals {
			ch <- sig
		}
		code := make(chan int, 1)
		start := time.Now()
		go s.quitOnSignals(ch, grace, func(c int) { code <- c })
		got := <-code
		select {
		case <-quit:
			return true, got, time.Since(start)
		case <-time.After(time.Second):
			return false, got, time.Since(start)
		}
	}
	if quit, code, took := run(50*time.Millisecond, syscall.SIGTERM); !quit || code != 1 || took < 50*time.Millisecond {
		t.Fatalf("one signal: quit=%v code=%d after %v; want a quit, then exit 1 after grace", quit, code, took)
	}
	if quit, code, took := run(time.Hour, os.Interrupt, os.Interrupt); !quit || code != 1 || took > 10*time.Second {
		t.Fatalf("two signals: quit=%v code=%d after %v; want a quit, then exit 1 at once", quit, code, took)
	}
}

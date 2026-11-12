// shell/update/schedule_test.go

package update

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func scheduleTiming(t *testing.T) {
	t.Helper()
	oldDelay, oldEvery := firstDelay, checkEvery
	firstDelay, checkEvery = 20*time.Millisecond, time.Hour
	t.Cleanup(func() { firstDelay, checkEvery = oldDelay, oldEvery })
}

func TestScheduleChecksAfterTheFirstDelay(t *testing.T) {
	scheduleTiming(t)
	g := newGitHub(t)
	stop := Schedule(New(g.options()))
	t.Cleanup(stop)
	if g.hits.Load() != 0 {
		t.Fatal("checked before the first delay")
	}
	deadline := time.After(time.Second)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for g.hits.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("first check did not run")
		case <-ticker.C:
		}
	}
	time.Sleep(40 * time.Millisecond)
	if n := g.hits.Load(); n != 1 {
		t.Fatalf("requests = %d, want 1", n)
	}
}

func TestScheduleStopsWithoutWaitingForASlowServer(t *testing.T) {
	scheduleTiming(t)
	entered, cancelled := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		select {
		case <-time.After(5 * time.Second):
			_, _ = w.Write([]byte("[]"))
		case <-r.Context().Done():
			close(cancelled)
		}
	}))
	t.Cleanup(server.Close)
	stop := Schedule(New(Options{Owner: "smegg99", Repo: "App", Version: "1.4.3", APIBase: server.URL, HTTPClient: server.Client()}))
	t.Cleanup(stop)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("scheduled request never started")
	}
	start := time.Now()
	stop()
	stop()
	if elapsed := time.Since(start); elapsed >= 500*time.Millisecond {
		t.Fatalf("stop took %s", elapsed)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("in-flight request was not cancelled")
	}
}

func TestScheduleStopBeforeFirstCheckMakesNoRequest(t *testing.T) {
	scheduleTiming(t)
	g := newGitHub(t)
	stop := Schedule(New(g.options()))
	stop()
	time.Sleep(2 * firstDelay)
	if n := g.hits.Load(); n != 0 {
		t.Fatalf("requests after stop = %d", n)
	}
}

func TestDownloadWithoutAnUpdateReturnsAnError(t *testing.T) {
	if err := New(Options{}).Download(); err == nil {
		t.Fatal("Download without a release succeeded")
	}
}

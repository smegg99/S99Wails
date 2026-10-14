// shell/windows/registry_test.go
package windows

import (
	"encoding/json"
	"sync"
	"testing"
)

type fake struct {
	name   string
	mu     sync.Mutex
	events []string
}

func (f *fake) Name() string { return f.name }
func (*fake) Show()          {}
func (*fake) Hide()          {}
func (*fake) Focus()         {}
func (*fake) Focused() bool  { return false }
func (*fake) Visible() bool  { return true }
func (*fake) Close()         {}
func (f *fake) Emit(event string, _ any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}
func registry(t *testing.T) (*Registry, *fake, *fake) {
	t.Helper()
	r := New()
	a, b := &fake{name: "main"}, &fake{name: "other"}
	if err := r.Add(a); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(b); err != nil {
		t.Fatal(err)
	}
	return r, a, b
}
func TestSourceIsRetainedUntilTheDestinationAcknowledges(t *testing.T) {
	r, source, target := registry(t)
	offer, err := r.Move("main", "other", "tab:1", 2, json.RawMessage(`{"draft":"unfinished"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(target.events) != 0 {
		t.Fatal("sent into unready webview")
	}
	pending, err := r.Ready("other", "School")
	if err != nil || len(pending) != 1 || pending[0].ID != offer.ID {
		t.Fatalf("ready: %v %v", pending, err)
	}
	if r.CanClose("main") || r.CanClose("other") {
		t.Fatal("destroyed transfer participant")
	}
	if err := r.Claim("main", offer.ID); err == nil {
		t.Fatal("source acknowledged itself")
	}
	if err := r.Released("main", offer.ID); err == nil {
		t.Fatal("released before claim")
	}
	if err := r.Claim("other", offer.ID); err != nil {
		t.Fatal(err)
	}
	if len(source.events) != 1 || source.events[0] != "tab:release" {
		t.Fatal("release not routed to source")
	}
	replay, _ := r.Ready("main", "Lessons")
	if len(replay) != 1 || !replay[0].Claimed {
		t.Fatal("missed release could not recover")
	}
	if err := r.Released("main", offer.ID); err != nil {
		t.Fatal(err)
	}
	if !r.CanClose("main") || !r.CanClose("other") {
		t.Fatal("finished transfer retained")
	}
}
func TestConcurrentMovesOfOneTabKeepOneOwner(t *testing.T) {
	r, _, _ := registry(t)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() { _, err := r.Move("main", "other", "tab:1", 1, json.RawMessage(`{}`)); results <- err })
	}
	wg.Wait()
	close(results)
	accepted := 0
	for err := range results {
		if err == nil {
			accepted++
		} else if err != ErrBusy {
			t.Fatal(err)
		}
	}
	if accepted != 1 {
		t.Fatalf("accepted %d transfers", accepted)
	}
}
func TestCancelledMoveKeepsTheSourceAndFreesTheWindows(t *testing.T) {
	r, source, target := registry(t)
	_, _ = r.Ready("other", "School")
	offer, err := r.Move("main", "other", "tab:1", 1, json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Cancel("other", offer.ID); err == nil {
		t.Fatal("target cancelled source transfer")
	}
	if err := r.Cancel("main", offer.ID); err != nil {
		t.Fatal(err)
	}
	if len(source.events) != 1 || source.events[0] != "tab:cancelled" || len(target.events) != 2 || target.events[1] != "tab:cancel" {
		t.Fatal("wrong cancellation routing")
	}
	if !r.CanClose("main") {
		t.Fatal("cancelled transfer blocked close")
	}
}

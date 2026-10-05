// services/title/title_test.go

package title

import (
	"context"
	"sync"
	"testing"
	"time"
)

// recorder replaces the window setter and records applied titles.
func recorder(s *Service) *[]string {
	var got []string
	s.apply = func(_ context.Context, text string) { got = append(got, text) }
	return &got
}

func TestBaseOnlyUntilAPhraseArrives(t *testing.T) {
	s := New(Options{Base: "SmeggTuner"})
	if got := s.Title(t.Context()); got != "SmeggTuner" {
		t.Fatalf("Title() = %q, want %q", got, "SmeggTuner")
	}
}

func TestSetJoinsBaseAndPhrase(t *testing.T) {
	s := New(Options{Base: "SmeggTuner"})
	applied := recorder(s)

	if err := s.Set(t.Context(), "tuning Bugari Armando 12/41"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	want := "SmeggTuner - tuning Bugari Armando 12/41"
	if got := s.Title(t.Context()); got != want {
		t.Fatalf("Title() = %q, want %q", got, want)
	}
	if len(*applied) != 1 || (*applied)[0] != want {
		t.Fatalf("applied %q, want exactly [%q]", *applied, want)
	}
}

func TestCustomSeparator(t *testing.T) {
	s := New(Options{Base: "SmeggTuner", Sep: " | "})
	if err := s.Set(t.Context(), "settings"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := s.Title(t.Context()); got != "SmeggTuner | settings" {
		t.Fatalf("Title() = %q", got)
	}
}

func TestEmptyPhraseRestoresTheBareBase(t *testing.T) {
	s := New(Options{Base: "SmeggTuner"})
	applied := recorder(s)

	_ = s.Set(t.Context(), "workshop")
	_ = s.Set(t.Context(), "")

	if got := s.Title(t.Context()); got != "SmeggTuner" {
		t.Fatalf("Title() = %q, want %q", got, "SmeggTuner")
	}
	if len(*applied) != 2 || (*applied)[1] != "SmeggTuner" {
		t.Fatalf("applied %q, want the bare base last", *applied)
	}
}

// Each apply is a synchronous main-thread round trip, so repeats are skipped.
func TestIdenticalTitleIsNotAppliedTwice(t *testing.T) {
	s := New(Options{Base: "SmeggTuner"})
	applied := recorder(s)

	_ = s.Set(t.Context(), "tuning")
	_ = s.Set(t.Context(), "tuning")
	_ = s.Set(t.Context(), "tuning")

	if len(*applied) != 1 {
		t.Fatalf("applied %d times, want 1: %q", len(*applied), *applied)
	}
}

func TestDifferentTitlesAreEachApplied(t *testing.T) {
	s := New(Options{Base: "SmeggTuner"})
	applied := recorder(s)

	_ = s.Set(t.Context(), "tuning")
	_ = s.Set(t.Context(), "workshop")

	if len(*applied) != 2 {
		t.Fatalf("applied %d times, want 2: %q", len(*applied), *applied)
	}
}

func TestNilWindowIsANoOp(t *testing.T) {
	s := New(Options{Base: "SmeggTuner"})
	if err := s.Set(t.Context(), "tuning"); err != nil {
		t.Fatalf("Set: %v", err)
	}

}

func TestNoBaseLeavesThePhraseAlone(t *testing.T) {
	s := New(Options{})
	if got := s.Title(t.Context()); got != "" {
		t.Fatalf("Title() = %q, want empty", got)
	}

	if err := s.Set(t.Context(), "tuning"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := s.Title(t.Context()); got != "tuning" {
		t.Fatalf("Title() = %q, want %q", got, "tuning")
	}
}

// The service is built before application.New, so the Base fallback is retried
// until it resolves and then cached.
func TestBaseFallbackIsRetriedNotLatched(t *testing.T) {
	orig := appNameFn
	defer func() { appNameFn = orig }()

	calls := 0
	name := ""
	appNameFn = func() string {
		calls++
		return name
	}

	s := New(Options{})

	if got := s.Title(t.Context()); got != "" {
		t.Fatalf("Title() before an application exists = %q, want empty", got)
	}
	if calls == 0 {
		t.Fatalf("appNameFn was never called; the fallback path did not run")
	}

	// The application name becomes available after construction.
	name = "SmeggTuner"
	if got := s.Title(t.Context()); got != "SmeggTuner" {
		t.Fatalf("Title() once an application exists = %q, want %q; the fallback was latched empty at construction", got, "SmeggTuner")
	}

	if err := s.Set(t.Context(), "tuning"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := s.Title(t.Context()); got != "SmeggTuner - tuning" {
		t.Fatalf("Title() = %q, want %q", got, "SmeggTuner - tuning")
	}

	// Once resolved, the name is cached.
	callsAfterResolved := calls
	name = "SomethingElse"
	if got := s.Title(t.Context()); got != "SmeggTuner - tuning" {
		t.Fatalf("Title() = %q, want cached %q", got, "SmeggTuner - tuning")
	}
	if calls != callsAfterResolved {
		t.Fatalf("appNameFn called again after resolving a non-empty base: calls went from %d to %d", callsAfterResolved, calls)
	}
}

func TestConcurrentSetsLeaveAConsistentTitle(t *testing.T) {
	s := New(Options{Base: "SmeggTuner"})
	s.apply = func(context.Context, string) {}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Set(t.Context(), "tuning")
			_ = s.Title(t.Context())
		}()
	}
	wg.Wait()

	if got := s.Title(t.Context()); got != "SmeggTuner - tuning" {
		t.Fatalf("Title() = %q", got)
	}
}

// Set releases mu before apply, so a slow main-thread dispatch does not block
// other Set and Title calls.
func TestSetIsNotQueuedBehindAnInFlightDispatch(t *testing.T) {
	s := New(Options{Base: "SmeggTuner"})

	started := make(chan struct{})
	release := make(chan struct{})
	var startedOnce sync.Once

	s.apply = func(_ context.Context, text string) {
		if text == "SmeggTuner - first" {
			startedOnce.Do(func() { close(started) })
			<-release
		}
	}

	done := make(chan struct{})
	go func() {
		_ = s.Set(t.Context(), "first")
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatalf("apply(first) never started")
	}

	// apply("first") holds applyMu but not mu.
	writeDone := make(chan struct{})
	go func() {
		_ = s.Set(t.Context(), "second")
		close(writeDone)
	}()

	titleResult := make(chan string, 1)
	go func() { titleResult <- s.Title(t.Context()) }()
	select {
	case got := <-titleResult:
		if got != "SmeggTuner - first" && got != "SmeggTuner - second" {
			t.Fatalf("Title() = %q while apply(first) is in flight", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("Title() appears blocked behind the in-flight apply; mu may be held across apply")
	}

	close(release)
	<-done
	<-writeDone

	if got := s.Title(t.Context()); got != "SmeggTuner - second" {
		t.Fatalf("final Title() = %q, want %q", got, "SmeggTuner - second")
	}
}

// Set re-reads state after taking applyMu, so the last apply matches Title.
// The race is repeated because one attempt may not hit the bad ordering.
func TestConcurrentSetsNeverLeaveTheWindowOlderThanTitle(t *testing.T) {
	for attempt := 0; attempt < 200; attempt++ {
		s := New(Options{Base: "SmeggTuner"})

		var amu sync.Mutex
		var lastApplied string
		s.apply = func(_ context.Context, text string) {
			amu.Lock()
			lastApplied = text
			amu.Unlock()
		}

		var wg sync.WaitGroup
		texts := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
		for _, txt := range texts {
			wg.Add(1)
			go func(txt string) {
				defer wg.Done()
				_ = s.Set(t.Context(), txt)
			}(txt)
		}
		wg.Wait()

		amu.Lock()
		got := lastApplied
		amu.Unlock()

		if final := s.Title(t.Context()); got != final {
			t.Fatalf("attempt %d: last apply() call was %q but final Title() = %q; a stale value won the race", attempt, got, final)
		}
	}
}

func TestDifferentWindowsKeepTheirOwnPhrases(t *testing.T) {
	type key struct{}
	s := New(Options{Base: "Goptivum"})
	s.windowID = func(ctx context.Context) uint { return ctx.Value(key{}).(uint) }
	got := map[uint]string{}
	s.apply = func(ctx context.Context, title string) { got[s.windowID(ctx)] = title }
	first := context.WithValue(t.Context(), key{}, uint(1))
	second := context.WithValue(t.Context(), key{}, uint(2))
	_ = s.Set(first, "School")
	_ = s.Set(second, "Timetable")
	_ = s.Set(first, "Lessons")
	if got[1] != "Goptivum - Lessons" || got[2] != "Goptivum - Timetable" {
		t.Fatalf("crossed titles: %v", got)
	}
	if s.Title(second) != got[2] {
		t.Fatal("read another window's phrase")
	}
}

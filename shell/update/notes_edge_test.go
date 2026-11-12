// shell/update/notes_edge_test.go

package update

import (
	"errors"
	"sync"
	"testing"
)

func TestNotesAreOfflineAndResolvedAtEachStateCall(t *testing.T) {
	opts := notesOptions(&memoryStore{value: Memory{Notes: "1.4.3"}, ok: true})
	opts.Enabled = func() bool { return false }
	s := New(opts)
	for _, locale := range []string{"en", "pl", "en"} {
		notes := s.State(locale).Notes
		if notes == nil || notes.Lang != locale {
			t.Fatalf("locale %s: %+v", locale, notes)
		}
	}
}

func TestEmptyNotesAndDowngradesStayQuiet(t *testing.T) {
	for _, tc := range []struct{ previous, text, current string }{
		{"1.4.3", "", "1.5.0"},
		{"1.6.0", changelog, "1.5.0"},
		{"1.5.0", changelog, "1.5.0"},
		{"1.4.3", changelog, ""},
	} {
		opts := notesOptions(&memoryStore{value: Memory{Notes: tc.previous}, ok: true})
		opts.Version, opts.Changelog = tc.current, tc.text
		if notes := New(opts).State("en").Notes; notes != nil {
			t.Fatalf("case %+v: %+v", tc, notes)
		}
	}
}

func TestStateReportsStorageFailuresAndRetries(t *testing.T) {
	failure := errors.New("store unavailable")
	store := &memoryStore{loadErr: failure}
	opts := notesOptions(store)
	var failures []error
	opts.OnError = func(err error) { failures = append(failures, err) }
	s := New(opts)
	if state := s.State("en"); state.Current != opts.Version || state.Notes != nil {
		t.Fatalf("state = %+v", state)
	}
	store.loadErr, store.saveErr = nil, failure
	if state := s.State("en"); state.Notes != nil {
		t.Fatalf("state = %+v", state)
	}
	store.saveErr = nil
	s.State("en")
	memory, ok, err := store.Load()
	if len(failures) != 2 || err != nil || !ok || memory.Notes != opts.Version {
		t.Fatalf("failures = %v, memory = %+v, ok = %v, err = %v", failures, memory, ok, err)
	}
}

func TestNotesAndIgnorePreserveEachOtherUnderConcurrentCalls(t *testing.T) {
	store := &memoryStore{}
	s := New(notesOptions(store))
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Go(func() {
			if err := s.Ignore("v1.6.0"); err != nil {
				t.Error(err)
			}
		})
		wg.Go(func() {
			if err := s.NotesShown(); err != nil {
				t.Error(err)
			}
		})
		wg.Go(func() { s.State("pl") })
	}
	wg.Wait()
	memory, _, _ := store.Load()
	if memory != (Memory{Ignored: "1.6.0", Notes: "1.5.0"}) {
		t.Fatalf("memory = %+v", memory)
	}
}

func TestStateReturnsAnOwnedRelease(t *testing.T) {
	s := New(notesOptions(&memoryStore{}))
	s.latest = &Release{Version: "1.6.0", URL: "https://example.test/downloads"}
	state := s.State("en")
	state.Update.Version = "changed"
	if got := s.State("en").Update.Version; got != "1.6.0" {
		t.Fatalf("stored version = %q", got)
	}
}

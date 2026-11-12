// shell/update/notes_test.go

package update

import (
	"context"
	"strings"
	"testing"
)

const changelog = "<!-- lang:en -->\n## 1.5.0\nFixed.\n<!-- lang:pl -->\n## 1.5.0\nPoprawione.\n"

func notesOptions(store *memoryStore) Options {
	return Options{Version: "1.5.0", Changelog: changelog, Store: store}
}

func TestFreshInstallRecordsItsVersionWithoutNotes(t *testing.T) {
	store := &memoryStore{}
	s := New(notesOptions(store))
	state := s.State("en")
	if state.Current != "1.5.0" || state.Notes != nil {
		t.Fatalf("state = %+v", state)
	}
	memory, ok, err := store.Load()
	if err != nil || !ok || memory.Notes != "1.5.0" {
		t.Fatalf("memory = %+v, ok = %v, err = %v", memory, ok, err)
	}
}

func TestExistingInstallSeesNotesOnce(t *testing.T) {
	opts := notesOptions(&memoryStore{})
	opts.Existing = func() bool { return true }
	s := New(opts)
	state := s.State("en")
	want := Notes{Version: "1.5.0", Text: "## 1.5.0\nFixed.", Lang: "en"}
	if state.Notes == nil || *state.Notes != want {
		t.Fatalf("notes = %+v, want %+v", state.Notes, want)
	}
	if err := s.NotesShown(); err != nil {
		t.Fatal(err)
	}
	if got := s.State("en").Notes; got != nil {
		t.Fatalf("notes after handling = %+v", got)
	}
}

func TestUpgradeShowsNotesInTheRequestedLanguage(t *testing.T) {
	s := New(notesOptions(&memoryStore{value: Memory{Notes: "1.4.3"}, ok: true}))
	notes := s.State("pl").Notes
	if notes == nil || notes.Lang != "pl" || !strings.Contains(notes.Text, "Poprawione") {
		t.Fatalf("notes = %+v", notes)
	}
}

func TestMissingLanguageFallsBackToEnglish(t *testing.T) {
	s := New(notesOptions(&memoryStore{value: Memory{Notes: "1.4.3"}, ok: true}))
	notes := s.State("de").Notes
	if notes == nil || notes.Lang != "en" {
		t.Fatalf("notes = %+v", notes)
	}
}

func TestIgnoreHidesThatVersionFromState(t *testing.T) {
	g := newGitHub(t, release("v1.6.0", "app-1.6.0-linux-amd64.AppImage"))
	opts := g.options()
	store := &memoryStore{}
	opts.Store = store
	s := New(opts)
	if err := s.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.State("en").Update == nil {
		t.Fatal("missing update before Ignore")
	}
	if err := s.Ignore("v1.6.0"); err != nil {
		t.Fatal(err)
	}
	if got := s.State("en").Update; got != nil {
		t.Fatalf("ignored update = %+v", got)
	}
	memory, _, _ := store.Load()
	if memory.Ignored != "1.6.0" {
		t.Fatalf("ignored = %q", memory.Ignored)
	}
}

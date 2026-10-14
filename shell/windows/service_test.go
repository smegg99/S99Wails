// shell/windows/service_test.go
package windows

import (
	"errors"
	"testing"
)

type closeSurface struct {
	name    string
	resumed bool
}

func (s *closeSurface) Name() string { return s.name }
func (*closeSurface) Show()          {}
func (*closeSurface) Hide()          {}
func (*closeSurface) Focus()         {}
func (*closeSurface) Focused() bool  { return false }
func (*closeSurface) Visible() bool  { return true }
func (*closeSurface) Close()         {}
func (s *closeSurface) Emit(event string, _ any) {
	if event == "window:resume" {
		s.resumed = true
	}
}

func closingService(t *testing.T) (*Service, *closeSurface, *closeSurface, *bool) {
	t.Helper()
	quit := false
	s := &Service{registry: New(), closeToken: map[string]string{}, closeEpoch: map[string]uint64{}, keep: func() error { return nil }, quit: func() { quit = true }}
	a, b := &closeSurface{name: "main"}, &closeSurface{name: "other"}
	if err := s.registry.Add(a); err != nil {
		t.Fatal(err)
	}
	if err := s.registry.Add(b); err != nil {
		t.Fatal(err)
	}
	return s, a, b, &quit
}

func TestQuitWaitsForEveryWindowBeforeQuitting(t *testing.T) {
	s, _, _, quit := closingService(t)
	if RequestQuit(s) {
		t.Fatal("quit before flush")
	}
	if err := s.closeReady("main", s.closeToken["main"], true); err != nil {
		t.Fatal(err)
	}
	if *quit {
		t.Fatal("quit while another window was still flushing")
	}
	if err := s.closeReady("other", s.closeToken["other"], true); err != nil {
		t.Fatal(err)
	}
	if !*quit || !RequestQuit(s) {
		t.Fatal("all ready but quit not released")
	}
}

func TestFailedJournalKeepsAllWindowsOpenIncludingLateQuitReplies(t *testing.T) {
	s, a, b, quit := closingService(t)
	failure := errors.New("disk full")
	s.keep = func() error { return failure }
	RequestQuit(s)
	late := s.closeToken["other"]
	if err := s.closeReady("main", s.closeToken["main"], true); !errors.Is(err, failure) {
		t.Fatalf("failure: %v", err)
	}
	if err := s.closeReady("other", late, true); err != nil {
		t.Fatalf("late reply: %v", err)
	}
	if *quit || len(s.registry.List()) != 2 || !a.resumed || !b.resumed {
		t.Fatal("failed quit closed or froze a workspace")
	}
	s.keep = func() error { return nil }
	RequestQuit(s)
	_ = s.closeReady("main", s.closeToken["main"], true)
	_ = s.closeReady("other", s.closeToken["other"], true)
	if !*quit {
		t.Fatal("quit could not retry after recovery")
	}
}

func TestOwnerChangeDuringKeepInvalidatesClose(t *testing.T) {
	s, _, _, quit := closingService(t)
	s.owner = "before"
	dropped := false
	s.options.Drop = func(string, string) error { dropped = true; return nil }
	s.keep = func() error { SetOwner(s, "after"); return nil }
	RequestQuit(s)
	token := s.closeToken["main"]
	if err := s.closeReady("main", token, true); err != nil {
		t.Fatal(err)
	}
	if *quit || dropped || len(s.registry.List()) != 2 {
		t.Fatal("old close affected new workspace")
	}
}

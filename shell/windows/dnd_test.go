// shell/windows/dnd_test.go

package windows

import "testing"

func TestWaylandDropSiteResolvesInItsOwnCoordinates(t *testing.T) {
	s := &Service{registry: New(), strips: map[string]Strip{
		"main":  {Bounds: Rect{0, 0, 600, 40}, Tabs: []TabRect{{ID: "home", X: 0, Width: 40, Pinned: true}, {ID: "a", X: 40, Width: 180}}},
		"other": {Bounds: Rect{0, 0, 600, 40}, Tabs: []TabRect{{ID: "home", X: 0, Width: 40, Pinned: true}, {ID: "b", X: 40, Width: 180}}},
	}}
	main, other := s.handle("main"), s.handle("other")
	if s.handle("main") != main || main == other {
		t.Fatalf("handles not stable and distinct: %d %d", main, other)
	}
	d := &nativeDrag{id: "d", source: "main", tab: "a"}
	if got := s.target(d, other, 50, 20); got.Target != "other" || got.Slot != 1 {
		t.Fatalf("other strip: %+v", got)
	}
	if got := s.target(d, other, 300, 400); got.Target != "other" || got.Slot != 2 {
		t.Fatalf("other body appends: %+v", got)
	}
	if got := s.target(d, main, 300, 400); got.Target != "" || got.Slot != -1 {
		t.Fatalf("source body tears off: %+v", got)
	}
	if got := s.target(d, main, 50, 20); got.Target != "main" || got.Slot != 1 {
		t.Fatalf("source strip reorders: %+v", got)
	}
	if got := s.target(d, 999, 50, 20); got.Target != "" {
		t.Fatalf("unknown handle accepted: %+v", got)
	}
}

// A flood of motions from the GUI thread must never push out the drop and the end that follow it.
func TestDropAndEndSurviveAFloodOfMotions(t *testing.T) {
	wake := dndListen()
	defer dndStop(wake)
	for i := range 5000 {
		dndPost(dndEvent{kind: dndMotion, handle: 1, x: i})
	}
	dndPost(dndEvent{kind: dndDrop, handle: 1, x: 7, y: 8})
	dndPost(dndEvent{kind: dndFailed, result: dragResultNoTarget})
	dndPost(dndEvent{kind: dndEnd})
	select {
	case <-wake:
	default:
		t.Fatal("posting did not wake the drag loop")
	}
	got := dndTake()
	if len(got) != 4 {
		t.Fatalf("events = %+v, want one coalesced motion, the drop, the failure and the end", got)
	}
	if got[0].kind != dndMotion || got[0].x != 4999 {
		t.Fatalf("motion = %+v, want the latest", got[0])
	}
	if got[1].kind != dndDrop || got[2].kind != dndFailed || got[3].kind != dndEnd {
		t.Fatalf("events = %+v", got)
	}
	if len(dndTake()) != 0 {
		t.Fatal("taken events were delivered twice")
	}
}

// A drag GTK broke ends without a tear-off; a drop nowhere or Escape tears the tab off, as on X11.
func TestDragEndingFollowsTheFailure(t *testing.T) {
	d := &nativeDrag{id: "d", source: "main", tab: "a"}
	into := &Drop{ID: "d", Tab: "a", Target: "other", Slot: 1}
	for _, c := range []struct {
		name    string
		dropped *Drop
		failed  int
		event   string
		target  string
	}{
		{"dropped on a window", into, dragResultSuccess, "tab:drop", "other"},
		{"dropped nowhere", nil, dragResultNoTarget, "tab:drop", ""},
		{"escape", nil, dragResultUserCancelled, "tab:drop", ""},
		{"grab broken", nil, dragResultGrabBroken, "tab:drag-end", ""},
		{"error", nil, dragResultError, "tab:drag-end", ""},
	} {
		event, payload := dragEnding(d, c.dropped, c.failed)
		if event != c.event {
			t.Fatalf("%s: event %q, want %q", c.name, event, c.event)
		}
		if drop, ok := payload.(Drop); ok && drop.Target != c.target {
			t.Fatalf("%s: drop %+v", c.name, drop)
		}
		if event == "tab:drag-end" && payload != "d" {
			t.Fatalf("%s: payload %v, want the drag id", c.name, payload)
		}
	}
}

// A finished drag's late cleanup must not wipe the sink of the drag that started after it.
func TestAnOldDragDoesNotStopTheNextOnesSink(t *testing.T) {
	first := dndListen()
	second := dndListen()
	if dndStop(first) {
		t.Fatal("the first drag stopped a sink it no longer owns")
	}
	dndPost(dndEvent{})
	select {
	case <-second:
	default:
		t.Fatal("the second drag's sink stopped listening")
	}
	if !dndStop(second) {
		t.Fatal("the second drag could not stop its own sink")
	}
}

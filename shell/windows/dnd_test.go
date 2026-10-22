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

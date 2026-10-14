package windows

import "testing"

func TestDropOverDestinationBodyAppendsButSourceStillTearsOff(t *testing.T) {
	s := Strip{Bounds: Rect{100, 0, 500, 40}, Tabs: []TabRect{
		{ID: "home", X: 100, Width: 40, Pinned: true}, {ID: "a", X: 140, Width: 180},
	}}
	if got := s.DropSlot(200, 200, ""); got != 2 {
		t.Fatalf("destination body: got %d, want append slot 2", got)
	}
	if got := s.DropSlot(200, 200, "a"); got != -1 {
		t.Fatalf("source body: got %d, want tear-off", got)
	}
	if got := s.DropSlot(150, 20, ""); got != 1 {
		t.Fatalf("destination strip: got %d, want insertion slot 1", got)
	}
	if got := (Strip{}).DropSlot(200, 200, ""); got != -1 {
		t.Fatalf("unmounted workspace accepted drop: %d", got)
	}
}

func TestStripHitKeepsPinnedPrefixAndExcludesSource(t *testing.T) {
	s := Strip{Bounds: Rect{100, 0, 500, 40}, Tabs: []TabRect{
		{ID: "home", X: 100, Width: 40, Pinned: true}, {ID: "a", X: 140, Width: 180}, {ID: "b", X: 320, Width: 180},
	}}
	for _, test := range []struct {
		x, y   int
		source string
		want   int
	}{
		{99, 20, "", -1}, {110, 20, "", 1}, {200, 20, "", 1}, {280, 20, "", 2},
		{400, 20, "", 2}, {499, 20, "", 3}, {400, 40, "", -1}, {499, 20, "a", 2},
	} {
		if got := s.Hit(test.x, test.y, test.source); got != test.want {
			t.Errorf("%+v: got %d", test, got)
		}
	}
}
